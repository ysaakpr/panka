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

// SubnetProvider handles AWS Subnet operations
type SubnetProvider struct {
	awsProvider *Provider
	ec2Client   *ec2.Client
}

// NewSubnetProvider creates a new Subnet provider
func NewSubnetProvider(p *Provider) *SubnetProvider {
	return &SubnetProvider{
		awsProvider: p,
		ec2Client:   ec2.NewFromConfig(p.GetConfig()),
	}
}

// SubnetConfig represents Subnet configuration
type SubnetConfig struct {
	VPCID            string
	CidrBlock        string
	AvailabilityZone string
	IsPublic         bool // If true, MapPublicIpOnLaunch will be enabled
	Tags             map[string]string
	TenantID         string
	Name             string
}

// SubnetResult represents the result of a Subnet operation
type SubnetResult struct {
	SubnetID         string
	VPCID            string
	CidrBlock        string
	AvailabilityZone string
	State            string
	IsPublic         bool
	Tags             map[string]string
}

// Create creates a new Subnet
func (s *SubnetProvider) Create(ctx context.Context, config *SubnetConfig, opts *provider.Options) (*SubnetResult, error) {
	s.awsProvider.logger.Info("Creating Subnet",
		zap.String("vpc_id", config.VPCID),
		zap.String("cidr_block", config.CidrBlock),
		zap.String("az", config.AvailabilityZone),
		zap.Bool("public", config.IsPublic),
	)

	if opts != nil && opts.DryRun {
		return &SubnetResult{
			SubnetID:         "subnet-dry-run",
			VPCID:            config.VPCID,
			CidrBlock:        config.CidrBlock,
			AvailabilityZone: config.AvailabilityZone,
			State:            "pending",
			IsPublic:         config.IsPublic,
		}, nil
	}

	// Build tags
	subnetType := "private"
	if config.IsPublic {
		subnetType = "public"
	}

	tags := s.buildTags(config.Tags, config.TenantID, "Subnet", map[string]string{
		"Name":              config.Name,
		"panka-subnet-type": subnetType,
	})

	// Create Subnet
	input := &ec2.CreateSubnetInput{
		VpcId:            aws.String(config.VPCID),
		CidrBlock:        aws.String(config.CidrBlock),
		AvailabilityZone: aws.String(config.AvailabilityZone),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSubnet,
				Tags:         tags,
			},
		},
	}

	result, err := s.ec2Client.CreateSubnet(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet: %w", err)
	}

	subnetID := *result.Subnet.SubnetId
	s.awsProvider.logger.Info("Subnet created", zap.String("subnet_id", subnetID))

	// If public subnet, enable auto-assign public IP
	if config.IsPublic {
		_, err = s.ec2Client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
			SubnetId:            aws.String(subnetID),
			MapPublicIpOnLaunch: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to enable public IP on launch: %w", err)
		}
		s.awsProvider.logger.Debug("Enabled public IP on launch", zap.String("subnet_id", subnetID))
	}

	// Wait for subnet to be available
	if err := s.waitForSubnetAvailable(ctx, subnetID); err != nil {
		return nil, err
	}

	return &SubnetResult{
		SubnetID:         subnetID,
		VPCID:            config.VPCID,
		CidrBlock:        config.CidrBlock,
		AvailabilityZone: config.AvailabilityZone,
		State:            "available",
		IsPublic:         config.IsPublic,
		Tags:             config.Tags,
	}, nil
}

// Get retrieves Subnet information
func (s *SubnetProvider) Get(ctx context.Context, subnetID string) (*SubnetResult, error) {
	s.awsProvider.logger.Debug("Getting Subnet", zap.String("subnet_id", subnetID))

	result, err := s.ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{subnetID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe subnet: %w", err)
	}

	if len(result.Subnets) == 0 {
		return nil, fmt.Errorf("subnet not found: %s", subnetID)
	}

	subnet := result.Subnets[0]
	tags := make(map[string]string)
	isPublic := false
	for _, tag := range subnet.Tags {
		tags[*tag.Key] = *tag.Value
		if *tag.Key == "panka-subnet-type" && *tag.Value == "public" {
			isPublic = true
		}
	}

	return &SubnetResult{
		SubnetID:         *subnet.SubnetId,
		VPCID:            *subnet.VpcId,
		CidrBlock:        *subnet.CidrBlock,
		AvailabilityZone: *subnet.AvailabilityZone,
		State:            string(subnet.State),
		IsPublic:         isPublic,
		Tags:             tags,
	}, nil
}

// Delete deletes a Subnet
func (s *SubnetProvider) Delete(ctx context.Context, subnetID string, opts *provider.Options) error {
	s.awsProvider.logger.Info("Deleting Subnet", zap.String("subnet_id", subnetID))

	if opts != nil && opts.DryRun {
		return nil
	}

	_, err := s.ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
		SubnetId: aws.String(subnetID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete subnet: %w", err)
	}

	s.awsProvider.logger.Info("Subnet deleted", zap.String("subnet_id", subnetID))
	return nil
}

// FindByVPC finds subnets by VPC ID
func (s *SubnetProvider) FindByVPC(ctx context.Context, vpcID string) ([]SubnetResult, error) {
	s.awsProvider.logger.Debug("Finding subnets by VPC", zap.String("vpc_id", vpcID))

	result, err := s.ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
			{
				Name:   aws.String("tag:ManagedBy"),
				Values: []string{"panka"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find subnets: %w", err)
	}

	subnets := make([]SubnetResult, 0, len(result.Subnets))
	for _, subnet := range result.Subnets {
		tags := make(map[string]string)
		isPublic := false
		for _, tag := range subnet.Tags {
			tags[*tag.Key] = *tag.Value
			if *tag.Key == "panka-subnet-type" && *tag.Value == "public" {
				isPublic = true
			}
		}

		subnets = append(subnets, SubnetResult{
			SubnetID:         *subnet.SubnetId,
			VPCID:            *subnet.VpcId,
			CidrBlock:        *subnet.CidrBlock,
			AvailabilityZone: *subnet.AvailabilityZone,
			State:            string(subnet.State),
			IsPublic:         isPublic,
			Tags:             tags,
		})
	}

	return subnets, nil
}

// FindByTenant finds subnets by tenant ID
func (s *SubnetProvider) FindByTenant(ctx context.Context, tenantID string) ([]SubnetResult, error) {
	s.awsProvider.logger.Debug("Finding subnets by tenant", zap.String("tenant_id", tenantID))

	result, err := s.ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
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
		return nil, fmt.Errorf("failed to find subnets: %w", err)
	}

	subnets := make([]SubnetResult, 0, len(result.Subnets))
	for _, subnet := range result.Subnets {
		tags := make(map[string]string)
		isPublic := false
		for _, tag := range subnet.Tags {
			tags[*tag.Key] = *tag.Value
			if *tag.Key == "panka-subnet-type" && *tag.Value == "public" {
				isPublic = true
			}
		}

		subnets = append(subnets, SubnetResult{
			SubnetID:         *subnet.SubnetId,
			VPCID:            *subnet.VpcId,
			CidrBlock:        *subnet.CidrBlock,
			AvailabilityZone: *subnet.AvailabilityZone,
			State:            string(subnet.State),
			IsPublic:         isPublic,
			Tags:             tags,
		})
	}

	return subnets, nil
}

// waitForSubnetAvailable waits for subnet to become available
func (s *SubnetProvider) waitForSubnetAvailable(ctx context.Context, subnetID string) error {
	s.awsProvider.logger.Debug("Waiting for subnet to be available", zap.String("subnet_id", subnetID))

	waiter := ec2.NewSubnetAvailableWaiter(s.ec2Client)
	err := waiter.Wait(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{subnetID},
	}, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("subnet did not become available: %w", err)
	}

	return nil
}

// buildTags builds tags for subnet resources
func (s *SubnetProvider) buildTags(customTags map[string]string, tenantID, resourceType string, extraTags map[string]string) []types.Tag {
	tags := []types.Tag{
		{Key: aws.String("ManagedBy"), Value: aws.String("panka")},
		{Key: aws.String("panka-resource-type"), Value: aws.String(resourceType)},
	}

	if tenantID != "" {
		tags = append(tags, types.Tag{Key: aws.String("panka-tenant"), Value: aws.String(tenantID)})
	}

	// Add default tags from provider
	if s.awsProvider.tagHelper != nil {
		for k, val := range s.awsProvider.tagHelper.DefaultTags {
			tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
		}
	}

	// Add extra tags
	for k, val := range extraTags {
		tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
	}

	// Add custom tags
	for k, val := range customTags {
		tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
	}

	return tags
}

