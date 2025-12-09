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

// InternetGatewayProvider handles AWS Internet Gateway operations
type InternetGatewayProvider struct {
	awsProvider *Provider
	ec2Client   *ec2.Client
}

// NewInternetGatewayProvider creates a new Internet Gateway provider
func NewInternetGatewayProvider(p *Provider) *InternetGatewayProvider {
	return &InternetGatewayProvider{
		awsProvider: p,
		ec2Client:   ec2.NewFromConfig(p.GetConfig()),
	}
}

// InternetGatewayConfig represents Internet Gateway configuration
type InternetGatewayConfig struct {
	VPCID    string
	Tags     map[string]string
	TenantID string
}

// InternetGatewayResult represents the result of an Internet Gateway operation
type InternetGatewayResult struct {
	InternetGatewayID string
	VPCID             string
	State             string
	Tags              map[string]string
}

// Create creates a new Internet Gateway and attaches it to the VPC
func (i *InternetGatewayProvider) Create(ctx context.Context, config *InternetGatewayConfig, opts *provider.Options) (*InternetGatewayResult, error) {
	i.awsProvider.logger.Info("Creating Internet Gateway",
		zap.String("vpc_id", config.VPCID),
		zap.String("tenant_id", config.TenantID),
	)

	if opts != nil && opts.DryRun {
		return &InternetGatewayResult{
			InternetGatewayID: "igw-dry-run",
			VPCID:             config.VPCID,
			State:             "pending",
		}, nil
	}

	// Build tags
	tags := i.buildTags(config.Tags, config.TenantID, "InternetGateway")

	// Create Internet Gateway
	input := &ec2.CreateInternetGatewayInput{
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInternetGateway,
				Tags:         tags,
			},
		},
	}

	result, err := i.ec2Client.CreateInternetGateway(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create internet gateway: %w", err)
	}

	igwID := *result.InternetGateway.InternetGatewayId
	i.awsProvider.logger.Info("Internet Gateway created", zap.String("igw_id", igwID))

	// Attach to VPC
	_, err = i.ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(igwID),
		VpcId:             aws.String(config.VPCID),
	})
	if err != nil {
		// Clean up the created IGW
		_, _ = i.ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: aws.String(igwID),
		})
		return nil, fmt.Errorf("failed to attach internet gateway to VPC: %w", err)
	}

	i.awsProvider.logger.Info("Internet Gateway attached to VPC",
		zap.String("igw_id", igwID),
		zap.String("vpc_id", config.VPCID),
	)

	return &InternetGatewayResult{
		InternetGatewayID: igwID,
		VPCID:             config.VPCID,
		State:             "attached",
		Tags:              config.Tags,
	}, nil
}

// Get retrieves Internet Gateway information
func (i *InternetGatewayProvider) Get(ctx context.Context, igwID string) (*InternetGatewayResult, error) {
	i.awsProvider.logger.Debug("Getting Internet Gateway", zap.String("igw_id", igwID))

	result, err := i.ec2Client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		InternetGatewayIds: []string{igwID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe internet gateway: %w", err)
	}

	if len(result.InternetGateways) == 0 {
		return nil, fmt.Errorf("internet gateway not found: %s", igwID)
	}

	igw := result.InternetGateways[0]
	tags := make(map[string]string)
	for _, tag := range igw.Tags {
		tags[*tag.Key] = *tag.Value
	}

	var vpcID string
	var state string = "detached"
	if len(igw.Attachments) > 0 {
		vpcID = *igw.Attachments[0].VpcId
		state = string(igw.Attachments[0].State)
	}

	return &InternetGatewayResult{
		InternetGatewayID: *igw.InternetGatewayId,
		VPCID:             vpcID,
		State:             state,
		Tags:              tags,
	}, nil
}

// Delete detaches and deletes an Internet Gateway
func (i *InternetGatewayProvider) Delete(ctx context.Context, igwID string, vpcID string, opts *provider.Options) error {
	i.awsProvider.logger.Info("Deleting Internet Gateway", zap.String("igw_id", igwID))

	if opts != nil && opts.DryRun {
		return nil
	}

	// First, detach from VPC if attached
	if vpcID != "" {
		_, err := i.ec2Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
			InternetGatewayId: aws.String(igwID),
			VpcId:             aws.String(vpcID),
		})
		if err != nil {
			i.awsProvider.logger.Warn("Failed to detach internet gateway", zap.Error(err))
			// Continue anyway - it might already be detached
		} else {
			i.awsProvider.logger.Debug("Internet Gateway detached from VPC",
				zap.String("igw_id", igwID),
				zap.String("vpc_id", vpcID),
			)
		}
	}

	// Delete the Internet Gateway
	_, err := i.ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: aws.String(igwID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete internet gateway: %w", err)
	}

	i.awsProvider.logger.Info("Internet Gateway deleted", zap.String("igw_id", igwID))
	return nil
}

// FindByVPC finds Internet Gateways by VPC ID
func (i *InternetGatewayProvider) FindByVPC(ctx context.Context, vpcID string) ([]InternetGatewayResult, error) {
	i.awsProvider.logger.Debug("Finding Internet Gateways by VPC", zap.String("vpc_id", vpcID))

	result, err := i.ec2Client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("attachment.vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find internet gateways: %w", err)
	}

	igws := make([]InternetGatewayResult, 0, len(result.InternetGateways))
	for _, igw := range result.InternetGateways {
		tags := make(map[string]string)
		for _, tag := range igw.Tags {
			tags[*tag.Key] = *tag.Value
		}

		var state string = "detached"
		if len(igw.Attachments) > 0 {
			state = string(igw.Attachments[0].State)
		}

		igws = append(igws, InternetGatewayResult{
			InternetGatewayID: *igw.InternetGatewayId,
			VPCID:             vpcID,
			State:             state,
			Tags:              tags,
		})
	}

	return igws, nil
}

// buildTags builds tags for Internet Gateway resources
func (i *InternetGatewayProvider) buildTags(customTags map[string]string, tenantID, resourceType string) []types.Tag {
	tags := []types.Tag{
		{Key: aws.String("ManagedBy"), Value: aws.String("panka")},
		{Key: aws.String("panka-resource-type"), Value: aws.String(resourceType)},
	}

	if tenantID != "" {
		tags = append(tags, types.Tag{Key: aws.String("panka-tenant"), Value: aws.String(tenantID)})
	}

	if i.awsProvider.tagHelper != nil {
		for k, val := range i.awsProvider.tagHelper.DefaultTags {
			tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
		}
	}

	for k, val := range customTags {
		tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
	}

	return tags
}

// waitForAttachment waits for IGW to be attached to VPC
func (i *InternetGatewayProvider) waitForAttachment(ctx context.Context, igwID string) error {
	i.awsProvider.logger.Debug("Waiting for Internet Gateway attachment", zap.String("igw_id", igwID))

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(2 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for internet gateway attachment")
		case <-ticker.C:
			result, err := i.Get(ctx, igwID)
			if err != nil {
				return err
			}
			if result.State == "attached" || result.State == "available" {
				return nil
			}
		}
	}
}

