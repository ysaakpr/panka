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

// VPCProvider handles AWS VPC operations
type VPCProvider struct {
	awsProvider *Provider
	ec2Client   *ec2.Client
}

// NewVPCProvider creates a new VPC provider
func NewVPCProvider(p *Provider) *VPCProvider {
	return &VPCProvider{
		awsProvider: p,
		ec2Client:   ec2.NewFromConfig(p.GetConfig()),
	}
}

// VPCConfig represents VPC configuration
type VPCConfig struct {
	CidrBlock          string
	EnableDNSHostnames bool
	EnableDNSSupport   bool
	Tags               map[string]string
	TenantID           string
}

// VPCResult represents the result of a VPC operation
type VPCResult struct {
	VPCID     string
	CidrBlock string
	State     string
	Tags      map[string]string
}

// Create creates a new VPC
func (v *VPCProvider) Create(ctx context.Context, config *VPCConfig, opts *provider.Options) (*VPCResult, error) {
	v.awsProvider.logger.Info("Creating VPC",
		zap.String("cidr_block", config.CidrBlock),
		zap.String("tenant_id", config.TenantID),
	)

	if opts != nil && opts.DryRun {
		return &VPCResult{
			VPCID:     "vpc-dry-run",
			CidrBlock: config.CidrBlock,
			State:     "pending",
		}, nil
	}

	// Build tags
	tags := v.buildTags(config.Tags, config.TenantID, "VPC")

	// Create VPC
	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(config.CidrBlock),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags:         tags,
			},
		},
	}

	result, err := v.ec2Client.CreateVpc(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create VPC: %w", err)
	}

	vpcID := *result.Vpc.VpcId
	v.awsProvider.logger.Info("VPC created", zap.String("vpc_id", vpcID))

	// Enable DNS hostnames if requested
	if config.EnableDNSHostnames {
		_, err = v.ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
			VpcId:              aws.String(vpcID),
			EnableDnsHostnames: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to enable DNS hostnames: %w", err)
		}
		v.awsProvider.logger.Debug("Enabled DNS hostnames", zap.String("vpc_id", vpcID))
	}

	// Enable DNS support if requested
	if config.EnableDNSSupport {
		_, err = v.ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
			VpcId:            aws.String(vpcID),
			EnableDnsSupport: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to enable DNS support: %w", err)
		}
		v.awsProvider.logger.Debug("Enabled DNS support", zap.String("vpc_id", vpcID))
	}

	// Wait for VPC to be available
	if err := v.waitForVPCAvailable(ctx, vpcID); err != nil {
		return nil, err
	}

	return &VPCResult{
		VPCID:     vpcID,
		CidrBlock: config.CidrBlock,
		State:     "available",
		Tags:      config.Tags,
	}, nil
}

// Get retrieves VPC information
func (v *VPCProvider) Get(ctx context.Context, vpcID string) (*VPCResult, error) {
	v.awsProvider.logger.Debug("Getting VPC", zap.String("vpc_id", vpcID))

	result, err := v.ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe VPC: %w", err)
	}

	if len(result.Vpcs) == 0 {
		return nil, fmt.Errorf("VPC not found: %s", vpcID)
	}

	vpc := result.Vpcs[0]
	tags := make(map[string]string)
	for _, tag := range vpc.Tags {
		tags[*tag.Key] = *tag.Value
	}

	return &VPCResult{
		VPCID:     *vpc.VpcId,
		CidrBlock: *vpc.CidrBlock,
		State:     string(vpc.State),
		Tags:      tags,
	}, nil
}

// Delete deletes a VPC
func (v *VPCProvider) Delete(ctx context.Context, vpcID string, opts *provider.Options) error {
	v.awsProvider.logger.Info("Deleting VPC", zap.String("vpc_id", vpcID))

	if opts != nil && opts.DryRun {
		return nil
	}

	_, err := v.ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
		VpcId: aws.String(vpcID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete VPC: %w", err)
	}

	v.awsProvider.logger.Info("VPC deleted", zap.String("vpc_id", vpcID))
	return nil
}

// FindByTenant finds VPCs by tenant ID
func (v *VPCProvider) FindByTenant(ctx context.Context, tenantID string) ([]VPCResult, error) {
	v.awsProvider.logger.Debug("Finding VPCs by tenant", zap.String("tenant_id", tenantID))

	result, err := v.ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:panka-tenant"),
				Values: []string{tenantID},
			},
			{
				Name:   aws.String("tag:ManagedBy"),
				Values: []string{"panka"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find VPCs: %w", err)
	}

	vpcs := make([]VPCResult, 0, len(result.Vpcs))
	for _, vpc := range result.Vpcs {
		tags := make(map[string]string)
		for _, tag := range vpc.Tags {
			tags[*tag.Key] = *tag.Value
		}

		vpcs = append(vpcs, VPCResult{
			VPCID:     *vpc.VpcId,
			CidrBlock: *vpc.CidrBlock,
			State:     string(vpc.State),
			Tags:      tags,
		})
	}

	return vpcs, nil
}

// waitForVPCAvailable waits for VPC to become available
func (v *VPCProvider) waitForVPCAvailable(ctx context.Context, vpcID string) error {
	v.awsProvider.logger.Debug("Waiting for VPC to be available", zap.String("vpc_id", vpcID))

	waiter := ec2.NewVpcAvailableWaiter(v.ec2Client)
	err := waiter.Wait(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	}, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("VPC did not become available: %w", err)
	}

	return nil
}

// buildTags builds tags for VPC resources
func (v *VPCProvider) buildTags(customTags map[string]string, tenantID, resourceType string) []types.Tag {
	tags := []types.Tag{
		{Key: aws.String("ManagedBy"), Value: aws.String("panka")},
		{Key: aws.String("panka-resource-type"), Value: aws.String(resourceType)},
	}

	if tenantID != "" {
		tags = append(tags, types.Tag{Key: aws.String("panka-tenant"), Value: aws.String(tenantID)})
	}

	// Add default tags from provider
	if v.awsProvider.tagHelper != nil {
		for k, val := range v.awsProvider.tagHelper.DefaultTags {
			tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
		}
	}

	// Add custom tags
	for k, val := range customTags {
		tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
	}

	return tags
}

