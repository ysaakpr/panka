package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/yourusername/panka/pkg/provider"
	"go.uber.org/zap"
)

// SecurityGroupProvider handles AWS Security Group operations
type SecurityGroupProvider struct {
	awsProvider *Provider
	ec2Client   *ec2.Client
}

// NewSecurityGroupProvider creates a new Security Group provider
func NewSecurityGroupProvider(p *Provider) *SecurityGroupProvider {
	return &SecurityGroupProvider{
		awsProvider: p,
		ec2Client:   ec2.NewFromConfig(p.GetConfig()),
	}
}

// SecurityGroupConfig represents Security Group configuration
type SecurityGroupConfig struct {
	Name        string
	Description string
	VPCID       string
	Ingress     []SecurityGroupRule
	Egress      []SecurityGroupRule
	Tags        map[string]string
	TenantID    string
}

// SecurityGroupRule represents an ingress or egress rule
type SecurityGroupRule struct {
	Port        int32
	FromPort    int32
	ToPort      int32
	Protocol    string // tcp, udp, icmp, -1 (all)
	CidrBlocks  []string
	SourceSGID  string // Reference to another security group
	Description string
}

// SecurityGroupResult represents the result of a Security Group operation
type SecurityGroupResult struct {
	SecurityGroupID string
	Name            string
	Description     string
	VPCID           string
	Ingress         []SecurityGroupRule
	Egress          []SecurityGroupRule
	Tags            map[string]string
}

// Create creates a new Security Group
func (s *SecurityGroupProvider) Create(ctx context.Context, config *SecurityGroupConfig, opts *provider.Options) (*SecurityGroupResult, error) {
	s.awsProvider.logger.Info("Creating Security Group",
		zap.String("name", config.Name),
		zap.String("vpc_id", config.VPCID),
		zap.String("tenant_id", config.TenantID),
	)

	if opts != nil && opts.DryRun {
		return &SecurityGroupResult{
			SecurityGroupID: "sg-dry-run",
			Name:            config.Name,
			Description:     config.Description,
			VPCID:           config.VPCID,
		}, nil
	}

	// Build tags
	tags := s.buildTags(config.Tags, config.TenantID, "SecurityGroup", config.Name)

	// Create Security Group
	input := &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(config.Name),
		Description: aws.String(config.Description),
		VpcId:       aws.String(config.VPCID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroup,
				Tags:         tags,
			},
		},
	}

	result, err := s.ec2Client.CreateSecurityGroup(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create security group: %w", err)
	}

	sgID := *result.GroupId
	s.awsProvider.logger.Info("Security Group created", zap.String("sg_id", sgID))

	// Add ingress rules
	if len(config.Ingress) > 0 {
		if err := s.addIngressRules(ctx, sgID, config.Ingress); err != nil {
			return nil, fmt.Errorf("failed to add ingress rules: %w", err)
		}
	}

	// Add egress rules (note: SGs have a default allow all egress rule)
	if len(config.Egress) > 0 {
		// First revoke the default egress rule
		_, _ = s.ec2Client.RevokeSecurityGroupEgress(ctx, &ec2.RevokeSecurityGroupEgressInput{
			GroupId: aws.String(sgID),
			IpPermissions: []types.IpPermission{
				{
					IpProtocol: aws.String("-1"),
					IpRanges:   []types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
				},
			},
		})

		if err := s.addEgressRules(ctx, sgID, config.Egress); err != nil {
			return nil, fmt.Errorf("failed to add egress rules: %w", err)
		}
	}

	return &SecurityGroupResult{
		SecurityGroupID: sgID,
		Name:            config.Name,
		Description:     config.Description,
		VPCID:           config.VPCID,
		Ingress:         config.Ingress,
		Egress:          config.Egress,
		Tags:            config.Tags,
	}, nil
}

// Get retrieves Security Group information
func (s *SecurityGroupProvider) Get(ctx context.Context, sgID string) (*SecurityGroupResult, error) {
	s.awsProvider.logger.Debug("Getting Security Group", zap.String("sg_id", sgID))

	result, err := s.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{sgID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe security group: %w", err)
	}

	if len(result.SecurityGroups) == 0 {
		return nil, fmt.Errorf("security group not found: %s", sgID)
	}

	sg := result.SecurityGroups[0]
	tags := make(map[string]string)
	for _, tag := range sg.Tags {
		tags[*tag.Key] = *tag.Value
	}

	// Convert ingress rules
	ingress := s.convertIpPermissions(sg.IpPermissions)

	// Convert egress rules
	egress := s.convertIpPermissions(sg.IpPermissionsEgress)

	return &SecurityGroupResult{
		SecurityGroupID: *sg.GroupId,
		Name:            *sg.GroupName,
		Description:     *sg.Description,
		VPCID:           *sg.VpcId,
		Ingress:         ingress,
		Egress:          egress,
		Tags:            tags,
	}, nil
}

// Delete deletes a Security Group
func (s *SecurityGroupProvider) Delete(ctx context.Context, sgID string, opts *provider.Options) error {
	s.awsProvider.logger.Info("Deleting Security Group", zap.String("sg_id", sgID))

	if opts != nil && opts.DryRun {
		return nil
	}

	_, err := s.ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(sgID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete security group: %w", err)
	}

	s.awsProvider.logger.Info("Security Group deleted", zap.String("sg_id", sgID))
	return nil
}

// AddIngressRule adds an ingress rule to a security group
func (s *SecurityGroupProvider) AddIngressRule(ctx context.Context, sgID string, rule SecurityGroupRule) error {
	return s.addIngressRules(ctx, sgID, []SecurityGroupRule{rule})
}

// AddEgressRule adds an egress rule to a security group
func (s *SecurityGroupProvider) AddEgressRule(ctx context.Context, sgID string, rule SecurityGroupRule) error {
	return s.addEgressRules(ctx, sgID, []SecurityGroupRule{rule})
}

// FindByVPC finds Security Groups by VPC ID
func (s *SecurityGroupProvider) FindByVPC(ctx context.Context, vpcID string) ([]SecurityGroupResult, error) {
	s.awsProvider.logger.Debug("Finding Security Groups by VPC", zap.String("vpc_id", vpcID))

	result, err := s.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
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
		return nil, fmt.Errorf("failed to find security groups: %w", err)
	}

	sgs := make([]SecurityGroupResult, 0, len(result.SecurityGroups))
	for _, sg := range result.SecurityGroups {
		tags := make(map[string]string)
		for _, tag := range sg.Tags {
			tags[*tag.Key] = *tag.Value
		}

		sgs = append(sgs, SecurityGroupResult{
			SecurityGroupID: *sg.GroupId,
			Name:            *sg.GroupName,
			Description:     *sg.Description,
			VPCID:           *sg.VpcId,
			Ingress:         s.convertIpPermissions(sg.IpPermissions),
			Egress:          s.convertIpPermissions(sg.IpPermissionsEgress),
			Tags:            tags,
		})
	}

	return sgs, nil
}

// FindByTenant finds Security Groups by tenant ID
func (s *SecurityGroupProvider) FindByTenant(ctx context.Context, tenantID string) ([]SecurityGroupResult, error) {
	s.awsProvider.logger.Debug("Finding Security Groups by tenant", zap.String("tenant_id", tenantID))

	result, err := s.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
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
		return nil, fmt.Errorf("failed to find security groups: %w", err)
	}

	sgs := make([]SecurityGroupResult, 0, len(result.SecurityGroups))
	for _, sg := range result.SecurityGroups {
		tags := make(map[string]string)
		for _, tag := range sg.Tags {
			tags[*tag.Key] = *tag.Value
		}

		sgs = append(sgs, SecurityGroupResult{
			SecurityGroupID: *sg.GroupId,
			Name:            *sg.GroupName,
			Description:     *sg.Description,
			VPCID:           *sg.VpcId,
			Ingress:         s.convertIpPermissions(sg.IpPermissions),
			Egress:          s.convertIpPermissions(sg.IpPermissionsEgress),
			Tags:            tags,
		})
	}

	return sgs, nil
}

// addIngressRules adds ingress rules to a security group
func (s *SecurityGroupProvider) addIngressRules(ctx context.Context, sgID string, rules []SecurityGroupRule) error {
	permissions := s.buildIpPermissions(rules)

	_, err := s.ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:       aws.String(sgID),
		IpPermissions: permissions,
	})
	if err != nil {
		return err
	}

	s.awsProvider.logger.Debug("Added ingress rules",
		zap.String("sg_id", sgID),
		zap.Int("rule_count", len(rules)),
	)

	return nil
}

// addEgressRules adds egress rules to a security group
func (s *SecurityGroupProvider) addEgressRules(ctx context.Context, sgID string, rules []SecurityGroupRule) error {
	permissions := s.buildIpPermissions(rules)

	_, err := s.ec2Client.AuthorizeSecurityGroupEgress(ctx, &ec2.AuthorizeSecurityGroupEgressInput{
		GroupId:       aws.String(sgID),
		IpPermissions: permissions,
	})
	if err != nil {
		return err
	}

	s.awsProvider.logger.Debug("Added egress rules",
		zap.String("sg_id", sgID),
		zap.Int("rule_count", len(rules)),
	)

	return nil
}

// buildIpPermissions builds AWS IpPermission from SecurityGroupRule
func (s *SecurityGroupProvider) buildIpPermissions(rules []SecurityGroupRule) []types.IpPermission {
	permissions := make([]types.IpPermission, 0, len(rules))

	for _, rule := range rules {
		permission := types.IpPermission{
			IpProtocol: aws.String(rule.Protocol),
		}

		// Set port range
		if rule.FromPort != 0 || rule.ToPort != 0 {
			permission.FromPort = aws.Int32(rule.FromPort)
			permission.ToPort = aws.Int32(rule.ToPort)
		} else if rule.Port != 0 {
			permission.FromPort = aws.Int32(rule.Port)
			permission.ToPort = aws.Int32(rule.Port)
		}

		// Set CIDR blocks
		if len(rule.CidrBlocks) > 0 {
			ipRanges := make([]types.IpRange, 0, len(rule.CidrBlocks))
			for _, cidr := range rule.CidrBlocks {
				ipRange := types.IpRange{CidrIp: aws.String(cidr)}
				if rule.Description != "" {
					ipRange.Description = aws.String(rule.Description)
				}
				ipRanges = append(ipRanges, ipRange)
			}
			permission.IpRanges = ipRanges
		}

		// Set source security group
		if rule.SourceSGID != "" {
			permission.UserIdGroupPairs = []types.UserIdGroupPair{
				{
					GroupId:     aws.String(rule.SourceSGID),
					Description: aws.String(rule.Description),
				},
			}
		}

		permissions = append(permissions, permission)
	}

	return permissions
}

// convertIpPermissions converts AWS IpPermission to SecurityGroupRule
func (s *SecurityGroupProvider) convertIpPermissions(permissions []types.IpPermission) []SecurityGroupRule {
	rules := make([]SecurityGroupRule, 0)

	for _, perm := range permissions {
		var fromPort, toPort int32
		if perm.FromPort != nil {
			fromPort = *perm.FromPort
		}
		if perm.ToPort != nil {
			toPort = *perm.ToPort
		}

		protocol := "-1"
		if perm.IpProtocol != nil {
			protocol = *perm.IpProtocol
		}

		// Create rules for CIDR blocks
		for _, ipRange := range perm.IpRanges {
			rule := SecurityGroupRule{
				FromPort:   fromPort,
				ToPort:     toPort,
				Protocol:   protocol,
				CidrBlocks: []string{*ipRange.CidrIp},
			}
			if ipRange.Description != nil {
				rule.Description = *ipRange.Description
			}
			rules = append(rules, rule)
		}

		// Create rules for security group references
		for _, sg := range perm.UserIdGroupPairs {
			rule := SecurityGroupRule{
				FromPort: fromPort,
				ToPort:   toPort,
				Protocol: protocol,
			}
			if sg.GroupId != nil {
				rule.SourceSGID = *sg.GroupId
			}
			if sg.Description != nil {
				rule.Description = *sg.Description
			}
			rules = append(rules, rule)
		}
	}

	return rules
}

// buildTags builds tags for Security Group resources
func (s *SecurityGroupProvider) buildTags(customTags map[string]string, tenantID, resourceType, name string) []types.Tag {
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

	if s.awsProvider.tagHelper != nil {
		for k, val := range s.awsProvider.tagHelper.DefaultTags {
			tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
		}
	}

	for k, val := range customTags {
		tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
	}

	return tags
}

