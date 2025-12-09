package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/yourusername/panka/pkg/provider"
	"go.uber.org/zap"
)

// NATGatewayProvider handles AWS NAT Gateway operations
type NATGatewayProvider struct {
	awsProvider *Provider
	ec2Client   *ec2.Client
}

// NewNATGatewayProvider creates a new NAT Gateway provider
func NewNATGatewayProvider(p *Provider) *NATGatewayProvider {
	return &NATGatewayProvider{
		awsProvider: p,
		ec2Client:   ec2.NewFromConfig(p.GetConfig()),
	}
}

// NATGatewayConfig represents NAT Gateway configuration
type NATGatewayConfig struct {
	SubnetID         string // Must be a public subnet
	ConnectivityType string // "public" or "private"
	Tags             map[string]string
	TenantID         string
	Name             string
}

// NATGatewayResult represents the result of a NAT Gateway operation
type NATGatewayResult struct {
	NATGatewayID     string
	SubnetID         string
	ElasticIP        string
	AllocationID     string
	State            string
	ConnectivityType string
	Tags             map[string]string
}

// Create creates a new NAT Gateway
func (n *NATGatewayProvider) Create(ctx context.Context, config *NATGatewayConfig, opts *provider.Options) (*NATGatewayResult, error) {
	n.awsProvider.logger.Info("Creating NAT Gateway",
		zap.String("subnet_id", config.SubnetID),
		zap.String("connectivity", config.ConnectivityType),
		zap.String("tenant_id", config.TenantID),
	)

	if opts != nil && opts.DryRun {
		return &NATGatewayResult{
			NATGatewayID:     "nat-dry-run",
			SubnetID:         config.SubnetID,
			State:            "pending",
			ConnectivityType: config.ConnectivityType,
		}, nil
	}

	// For public NAT Gateway, allocate an Elastic IP first
	var allocationID string
	var elasticIP string

	connectivityType := types.ConnectivityTypePublic
	if config.ConnectivityType == "private" {
		connectivityType = types.ConnectivityTypePrivate
	} else {
		// Allocate Elastic IP
		eipResult, err := n.ec2Client.AllocateAddress(ctx, &ec2.AllocateAddressInput{
			Domain: types.DomainTypeVpc,
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeElasticIp,
					Tags:         n.buildTags(config.Tags, config.TenantID, "ElasticIP-NAT", config.Name),
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to allocate elastic IP: %w", err)
		}
		allocationID = *eipResult.AllocationId
		if eipResult.PublicIp != nil {
			elasticIP = *eipResult.PublicIp
		}
		n.awsProvider.logger.Debug("Elastic IP allocated",
			zap.String("allocation_id", allocationID),
			zap.String("public_ip", elasticIP),
		)
	}

	// Build tags
	tags := n.buildTags(config.Tags, config.TenantID, "NATGateway", config.Name)

	// Create NAT Gateway input
	input := &ec2.CreateNatGatewayInput{
		SubnetId:         aws.String(config.SubnetID),
		ConnectivityType: connectivityType,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeNatgateway,
				Tags:         tags,
			},
		},
	}

	// Add allocation ID for public NAT Gateway
	if allocationID != "" {
		input.AllocationId = aws.String(allocationID)
	}

	// Create NAT Gateway
	result, err := n.ec2Client.CreateNatGateway(ctx, input)
	if err != nil {
		// Clean up the allocated EIP if NAT Gateway creation failed
		if allocationID != "" {
			_, _ = n.ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
				AllocationId: aws.String(allocationID),
			})
		}
		return nil, fmt.Errorf("failed to create NAT gateway: %w", err)
	}

	natGatewayID := *result.NatGateway.NatGatewayId
	n.awsProvider.logger.Info("NAT Gateway created", zap.String("nat_gateway_id", natGatewayID))

	// Wait for NAT Gateway to become available
	if err := n.waitForNATGatewayAvailable(ctx, natGatewayID); err != nil {
		return nil, err
	}

	return &NATGatewayResult{
		NATGatewayID:     natGatewayID,
		SubnetID:         config.SubnetID,
		ElasticIP:        elasticIP,
		AllocationID:     allocationID,
		State:            "available",
		ConnectivityType: string(connectivityType),
		Tags:             config.Tags,
	}, nil
}

// Get retrieves NAT Gateway information
func (n *NATGatewayProvider) Get(ctx context.Context, natGatewayID string) (*NATGatewayResult, error) {
	n.awsProvider.logger.Debug("Getting NAT Gateway", zap.String("nat_gateway_id", natGatewayID))

	result, err := n.ec2Client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
		NatGatewayIds: []string{natGatewayID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe NAT gateway: %w", err)
	}

	if len(result.NatGateways) == 0 {
		return nil, fmt.Errorf("NAT gateway not found: %s", natGatewayID)
	}

	natGw := result.NatGateways[0]
	tags := make(map[string]string)
	for _, tag := range natGw.Tags {
		tags[*tag.Key] = *tag.Value
	}

	var elasticIP, allocationID string
	if len(natGw.NatGatewayAddresses) > 0 {
		addr := natGw.NatGatewayAddresses[0]
		if addr.PublicIp != nil {
			elasticIP = *addr.PublicIp
		}
		if addr.AllocationId != nil {
			allocationID = *addr.AllocationId
		}
	}

	return &NATGatewayResult{
		NATGatewayID:     *natGw.NatGatewayId,
		SubnetID:         *natGw.SubnetId,
		ElasticIP:        elasticIP,
		AllocationID:     allocationID,
		State:            string(natGw.State),
		ConnectivityType: string(natGw.ConnectivityType),
		Tags:             tags,
	}, nil
}

// Delete deletes a NAT Gateway and releases associated Elastic IP
func (n *NATGatewayProvider) Delete(ctx context.Context, natGatewayID string, opts *provider.Options) error {
	n.awsProvider.logger.Info("Deleting NAT Gateway", zap.String("nat_gateway_id", natGatewayID))

	if opts != nil && opts.DryRun {
		return nil
	}

	// Get NAT Gateway to find allocation ID
	natGw, err := n.Get(ctx, natGatewayID)
	if err != nil {
		return err
	}
	allocationID := natGw.AllocationID

	// Delete NAT Gateway
	_, err = n.ec2Client.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
		NatGatewayId: aws.String(natGatewayID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete NAT gateway: %w", err)
	}

	n.awsProvider.logger.Info("NAT Gateway deletion initiated", zap.String("nat_gateway_id", natGatewayID))

	// Wait for NAT Gateway to be deleted before releasing EIP
	if err := n.waitForNATGatewayDeleted(ctx, natGatewayID); err != nil {
		n.awsProvider.logger.Warn("Error waiting for NAT Gateway deletion", zap.Error(err))
	}

	// Release Elastic IP if present
	if allocationID != "" {
		_, err = n.ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
			AllocationId: aws.String(allocationID),
		})
		if err != nil {
			n.awsProvider.logger.Warn("Failed to release elastic IP",
				zap.String("allocation_id", allocationID),
				zap.Error(err),
			)
		} else {
			n.awsProvider.logger.Debug("Elastic IP released", zap.String("allocation_id", allocationID))
		}
	}

	n.awsProvider.logger.Info("NAT Gateway deleted", zap.String("nat_gateway_id", natGatewayID))
	return nil
}

// FindByVPC finds NAT Gateways in a VPC
func (n *NATGatewayProvider) FindByVPC(ctx context.Context, vpcID string) ([]NATGatewayResult, error) {
	n.awsProvider.logger.Debug("Finding NAT Gateways by VPC", zap.String("vpc_id", vpcID))

	result, err := n.ec2Client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
		Filter: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
			{
				Name:   aws.String("state"),
				Values: []string{"available", "pending"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find NAT gateways: %w", err)
	}

	natGws := make([]NATGatewayResult, 0, len(result.NatGateways))
	for _, natGw := range result.NatGateways {
		tags := make(map[string]string)
		for _, tag := range natGw.Tags {
			tags[*tag.Key] = *tag.Value
		}

		var elasticIP, allocationID string
		if len(natGw.NatGatewayAddresses) > 0 {
			addr := natGw.NatGatewayAddresses[0]
			if addr.PublicIp != nil {
				elasticIP = *addr.PublicIp
			}
			if addr.AllocationId != nil {
				allocationID = *addr.AllocationId
			}
		}

		natGws = append(natGws, NATGatewayResult{
			NATGatewayID:     *natGw.NatGatewayId,
			SubnetID:         *natGw.SubnetId,
			ElasticIP:        elasticIP,
			AllocationID:     allocationID,
			State:            string(natGw.State),
			ConnectivityType: string(natGw.ConnectivityType),
			Tags:             tags,
		})
	}

	return natGws, nil
}

// waitForNATGatewayAvailable waits for NAT Gateway to become available
func (n *NATGatewayProvider) waitForNATGatewayAvailable(ctx context.Context, natGatewayID string) error {
	n.awsProvider.logger.Debug("Waiting for NAT Gateway to be available", zap.String("nat_gateway_id", natGatewayID))

	waiter := ec2.NewNatGatewayAvailableWaiter(n.ec2Client)
	err := waiter.Wait(ctx, &ec2.DescribeNatGatewaysInput{
		NatGatewayIds: []string{natGatewayID},
	}, 10*time.Minute)
	if err != nil {
		return fmt.Errorf("NAT gateway did not become available: %w", err)
	}

	return nil
}

// waitForNATGatewayDeleted waits for NAT Gateway to be deleted
func (n *NATGatewayProvider) waitForNATGatewayDeleted(ctx context.Context, natGatewayID string) error {
	n.awsProvider.logger.Debug("Waiting for NAT Gateway to be deleted", zap.String("nat_gateway_id", natGatewayID))

	waiter := ec2.NewNatGatewayDeletedWaiter(n.ec2Client)
	err := waiter.Wait(ctx, &ec2.DescribeNatGatewaysInput{
		NatGatewayIds: []string{natGatewayID},
	}, 10*time.Minute)
	if err != nil {
		return fmt.Errorf("NAT gateway did not get deleted: %w", err)
	}

	return nil
}

// buildTags builds tags for NAT Gateway resources
func (n *NATGatewayProvider) buildTags(customTags map[string]string, tenantID, resourceType, name string) []types.Tag {
	tags := []types.Tag{
		{Key: aws.String("ManagedBy"), Value: aws.String("panka")},
		{Key: aws.String("panka-resource-type"), Value: aws.String(resourceType)},
	}

	if name != "" {
		tags = append(tags, types.Tag{Key: aws.String("Name"), Value: aws.String(name)})
	}

	if tenantID != "" {
		tags = append(tags, types.Tag{Key: aws.String("panka-tenant"), Value: aws.String(tenantID)})
	}

	if n.awsProvider.tagHelper != nil {
		for k, val := range n.awsProvider.tagHelper.DefaultTags {
			tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
		}
	}

	for k, val := range customTags {
		tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
	}

	return tags
}

