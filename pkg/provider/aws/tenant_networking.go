package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourusername/panka/pkg/provider"
	"github.com/yourusername/panka/pkg/tenant"
	"go.uber.org/zap"
)

// TenantNetworkingOrchestrator orchestrates the creation of tenant networking
type TenantNetworkingOrchestrator struct {
	awsProvider *Provider
	vpc         *VPCProvider
	subnet      *SubnetProvider
	igw         *InternetGatewayProvider
	natGw       *NATGatewayProvider
	sg          *SecurityGroupProvider
	routeTable  *RouteTableProvider
}

// NewTenantNetworkingOrchestrator creates a new networking orchestrator
func NewTenantNetworkingOrchestrator(p *Provider) *TenantNetworkingOrchestrator {
	return &TenantNetworkingOrchestrator{
		awsProvider: p,
		vpc:         NewVPCProvider(p),
		subnet:      NewSubnetProvider(p),
		igw:         NewInternetGatewayProvider(p),
		natGw:       NewNATGatewayProvider(p),
		sg:          NewSecurityGroupProvider(p),
		routeTable:  NewRouteTableProvider(p),
	}
}

// NetworkingResult represents the result of networking creation
type NetworkingResult struct {
	VPCID               string
	PublicSubnetIDs     []string
	PrivateSubnetIDs    []string
	InternetGatewayID   string
	NATGatewayIDs       []string
	DefaultSecurityGroupID string
	PublicRouteTableID  string
	PrivateRouteTableIDs []string
}

// CreateTenantNetworking creates the complete networking infrastructure for a tenant
func (o *TenantNetworkingOrchestrator) CreateTenantNetworking(
	ctx context.Context,
	tenantID string,
	networkingConfig *tenant.NetworkingConfig,
	opts *provider.Options,
) (*NetworkingResult, error) {
	o.awsProvider.logger.Info("Creating tenant networking",
		zap.String("tenant_id", tenantID),
		zap.String("vpc_cidr", networkingConfig.VPC.CidrBlock),
	)

	result := &NetworkingResult{}

	// Step 1: Create VPC
	o.awsProvider.logger.Info("Step 1/6: Creating VPC")
	vpcResult, err := o.vpc.Create(ctx, &VPCConfig{
		CidrBlock:          networkingConfig.VPC.CidrBlock,
		EnableDNSHostnames: networkingConfig.VPC.EnableDNSHostnames,
		EnableDNSSupport:   networkingConfig.VPC.EnableDNSSupport,
		TenantID:           tenantID,
		Tags: map[string]string{
			"Name": fmt.Sprintf("panka-%s-vpc", tenantID),
		},
	}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create VPC: %w", err)
	}
	result.VPCID = vpcResult.VPCID
	o.awsProvider.logger.Info("VPC created", zap.String("vpc_id", result.VPCID))

	// Step 2: Create Internet Gateway (if enabled)
	if networkingConfig.InternetGateway.Enabled {
		o.awsProvider.logger.Info("Step 2/6: Creating Internet Gateway")
		igwResult, err := o.igw.Create(ctx, &InternetGatewayConfig{
			VPCID:    result.VPCID,
			TenantID: tenantID,
			Tags: map[string]string{
				"Name": fmt.Sprintf("panka-%s-igw", tenantID),
			},
		}, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create internet gateway: %w", err)
		}
		result.InternetGatewayID = igwResult.InternetGatewayID
		o.awsProvider.logger.Info("Internet Gateway created", zap.String("igw_id", result.InternetGatewayID))
	} else {
		o.awsProvider.logger.Info("Step 2/6: Skipping Internet Gateway (disabled)")
	}

	// Step 3: Create Subnets
	o.awsProvider.logger.Info("Step 3/6: Creating Subnets")
	
	// Create public subnets
	result.PublicSubnetIDs = make([]string, 0, len(networkingConfig.Subnets.Public))
	for i, subnetCfg := range networkingConfig.Subnets.Public {
		subnetResult, err := o.subnet.Create(ctx, &SubnetConfig{
			VPCID:            result.VPCID,
			CidrBlock:        subnetCfg.CidrBlock,
			AvailabilityZone: subnetCfg.AvailabilityZone,
			IsPublic:         true,
			TenantID:         tenantID,
			Name:             fmt.Sprintf("panka-%s-public-%d", tenantID, i+1),
		}, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create public subnet: %w", err)
		}
		result.PublicSubnetIDs = append(result.PublicSubnetIDs, subnetResult.SubnetID)
		o.awsProvider.logger.Debug("Public subnet created",
			zap.String("subnet_id", subnetResult.SubnetID),
			zap.String("az", subnetCfg.AvailabilityZone),
		)
	}

	// Create private subnets
	result.PrivateSubnetIDs = make([]string, 0, len(networkingConfig.Subnets.Private))
	for i, subnetCfg := range networkingConfig.Subnets.Private {
		subnetResult, err := o.subnet.Create(ctx, &SubnetConfig{
			VPCID:            result.VPCID,
			CidrBlock:        subnetCfg.CidrBlock,
			AvailabilityZone: subnetCfg.AvailabilityZone,
			IsPublic:         false,
			TenantID:         tenantID,
			Name:             fmt.Sprintf("panka-%s-private-%d", tenantID, i+1),
		}, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create private subnet: %w", err)
		}
		result.PrivateSubnetIDs = append(result.PrivateSubnetIDs, subnetResult.SubnetID)
		o.awsProvider.logger.Debug("Private subnet created",
			zap.String("subnet_id", subnetResult.SubnetID),
			zap.String("az", subnetCfg.AvailabilityZone),
		)
	}

	o.awsProvider.logger.Info("Subnets created",
		zap.Int("public", len(result.PublicSubnetIDs)),
		zap.Int("private", len(result.PrivateSubnetIDs)),
	)

	// Step 4: Create Route Tables and Routes
	o.awsProvider.logger.Info("Step 4/6: Creating Route Tables")

	// Create public route table
	publicRTB, err := o.routeTable.Create(ctx, &RouteTableConfig{
		VPCID:    result.VPCID,
		Name:     fmt.Sprintf("panka-%s-public-rtb", tenantID),
		TenantID: tenantID,
		IsPublic: true,
	}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create public route table: %w", err)
	}
	result.PublicRouteTableID = publicRTB.RouteTableID

	// Add route to Internet Gateway for public route table
	if result.InternetGatewayID != "" && !(opts != nil && opts.DryRun) {
		err = o.routeTable.AddRoute(ctx, publicRTB.RouteTableID, RouteConfig{
			DestinationCidrBlock: "0.0.0.0/0",
			GatewayID:            result.InternetGatewayID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add internet route: %w", err)
		}
	}

	// Associate public subnets with public route table
	if !(opts != nil && opts.DryRun) {
		for _, subnetID := range result.PublicSubnetIDs {
			_, err = o.routeTable.AssociateSubnet(ctx, publicRTB.RouteTableID, subnetID)
			if err != nil {
				return nil, fmt.Errorf("failed to associate public subnet: %w", err)
			}
		}
	}

	// Step 5: Create NAT Gateway (if enabled)
	result.NATGatewayIDs = make([]string, 0)
	result.PrivateRouteTableIDs = make([]string, 0)

	if networkingConfig.NATGateway.Enabled && len(result.PublicSubnetIDs) > 0 {
		o.awsProvider.logger.Info("Step 5/6: Creating NAT Gateway(s)")

		var natSubnets []string
		if networkingConfig.NATGateway.Type == "per-az" {
			natSubnets = result.PublicSubnetIDs
		} else {
			// Single NAT Gateway
			natSubnets = []string{result.PublicSubnetIDs[0]}
		}

		for i, subnetID := range natSubnets {
			natResult, err := o.natGw.Create(ctx, &NATGatewayConfig{
				SubnetID:         subnetID,
				ConnectivityType: "public",
				TenantID:         tenantID,
				Name:             fmt.Sprintf("panka-%s-nat-%d", tenantID, i+1),
			}, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to create NAT gateway: %w", err)
			}
			result.NATGatewayIDs = append(result.NATGatewayIDs, natResult.NATGatewayID)
			o.awsProvider.logger.Debug("NAT Gateway created", zap.String("nat_id", natResult.NATGatewayID))

			// Create private route table for this NAT Gateway
			privateRTB, err := o.routeTable.Create(ctx, &RouteTableConfig{
				VPCID:    result.VPCID,
				Name:     fmt.Sprintf("panka-%s-private-rtb-%d", tenantID, i+1),
				TenantID: tenantID,
				IsPublic: false,
			}, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to create private route table: %w", err)
			}
			result.PrivateRouteTableIDs = append(result.PrivateRouteTableIDs, privateRTB.RouteTableID)

			// Add route to NAT Gateway
			if !(opts != nil && opts.DryRun) {
				err = o.routeTable.AddRoute(ctx, privateRTB.RouteTableID, RouteConfig{
					DestinationCidrBlock: "0.0.0.0/0",
					NATGatewayID:         natResult.NATGatewayID,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to add NAT route: %w", err)
				}
			}
		}

		// Associate private subnets with private route tables
		if !(opts != nil && opts.DryRun) && len(result.PrivateRouteTableIDs) > 0 {
			for i, subnetID := range result.PrivateSubnetIDs {
				// Use modulo to distribute subnets across route tables
				rtbIndex := i % len(result.PrivateRouteTableIDs)
				_, err = o.routeTable.AssociateSubnet(ctx, result.PrivateRouteTableIDs[rtbIndex], subnetID)
				if err != nil {
					return nil, fmt.Errorf("failed to associate private subnet: %w", err)
				}
			}
		}

		o.awsProvider.logger.Info("NAT Gateway(s) created", zap.Int("count", len(result.NATGatewayIDs)))
	} else {
		o.awsProvider.logger.Info("Step 5/6: Skipping NAT Gateway (disabled)")
	}

	// Step 6: Create Default Security Group
	o.awsProvider.logger.Info("Step 6/6: Creating Default Security Group")

	sgConfig := &SecurityGroupConfig{
		Name:        fmt.Sprintf("panka-%s-default-sg", tenantID),
		Description: fmt.Sprintf("Default security group for tenant %s", tenantID),
		VPCID:       result.VPCID,
		TenantID:    tenantID,
		Ingress:     []SecurityGroupRule{},
		Egress:      []SecurityGroupRule{},
	}

	// Allow internal traffic if configured
	if networkingConfig.DefaultSecurityGroup.AllowInternalTraffic {
		sgConfig.Ingress = append(sgConfig.Ingress, SecurityGroupRule{
			Protocol:    "-1", // All protocols
			CidrBlocks:  []string{networkingConfig.VPC.CidrBlock},
			Description: "Allow all internal traffic",
		})
	}

	// Add custom ingress rules
	for _, rule := range networkingConfig.DefaultSecurityGroup.Ingress {
		sgConfig.Ingress = append(sgConfig.Ingress, SecurityGroupRule{
			Port:        int32(rule.Port),
			FromPort:    int32(rule.FromPort),
			ToPort:      int32(rule.ToPort),
			Protocol:    rule.Protocol,
			CidrBlocks:  rule.CidrBlocks,
			Description: rule.Description,
		})
	}

	// Add custom egress rules or allow all outbound
	if len(networkingConfig.DefaultSecurityGroup.Egress) > 0 {
		for _, rule := range networkingConfig.DefaultSecurityGroup.Egress {
			sgConfig.Egress = append(sgConfig.Egress, SecurityGroupRule{
				Port:        int32(rule.Port),
				FromPort:    int32(rule.FromPort),
				ToPort:      int32(rule.ToPort),
				Protocol:    rule.Protocol,
				CidrBlocks:  rule.CidrBlocks,
				Description: rule.Description,
			})
		}
	} else {
		// Default: allow all outbound
		sgConfig.Egress = append(sgConfig.Egress, SecurityGroupRule{
			Protocol:    "-1",
			CidrBlocks:  []string{"0.0.0.0/0"},
			Description: "Allow all outbound traffic",
		})
	}

	sgResult, err := o.sg.Create(ctx, sgConfig, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create security group: %w", err)
	}
	result.DefaultSecurityGroupID = sgResult.SecurityGroupID
	o.awsProvider.logger.Info("Default Security Group created", zap.String("sg_id", result.DefaultSecurityGroupID))

	o.awsProvider.logger.Info("Tenant networking creation complete",
		zap.String("tenant_id", tenantID),
		zap.String("vpc_id", result.VPCID),
	)

	return result, nil
}

// DeleteTenantNetworking deletes all networking resources for a tenant
func (o *TenantNetworkingOrchestrator) DeleteTenantNetworking(
	ctx context.Context,
	tenantID string,
	opts *provider.Options,
) error {
	o.awsProvider.logger.Info("Deleting tenant networking", zap.String("tenant_id", tenantID))

	// Find VPCs for this tenant
	vpcs, err := o.vpc.FindByTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to find tenant VPCs: %w", err)
	}

	for _, vpc := range vpcs {
		o.awsProvider.logger.Info("Deleting VPC resources", zap.String("vpc_id", vpc.VPCID))

		// 1. Delete NAT Gateways
		natGws, _ := o.natGw.FindByVPC(ctx, vpc.VPCID)
		for _, natGw := range natGws {
			if err := o.natGw.Delete(ctx, natGw.NATGatewayID, opts); err != nil {
				o.awsProvider.logger.Warn("Failed to delete NAT Gateway",
					zap.String("nat_id", natGw.NATGatewayID),
					zap.Error(err),
				)
			}
		}

		// 2. Delete Security Groups (except default)
		sgs, _ := o.sg.FindByVPC(ctx, vpc.VPCID)
		for _, sg := range sgs {
			if !strings.Contains(sg.Name, "default") {
				if err := o.sg.Delete(ctx, sg.SecurityGroupID, opts); err != nil {
					o.awsProvider.logger.Warn("Failed to delete Security Group",
						zap.String("sg_id", sg.SecurityGroupID),
						zap.Error(err),
					)
				}
			}
		}

		// 3. Delete Route Tables (except main)
		rtbs, _ := o.routeTable.FindPankaManaged(ctx, vpc.VPCID)
		for _, rtb := range rtbs {
			if !rtb.IsMain {
				if err := o.routeTable.Delete(ctx, rtb.RouteTableID, opts); err != nil {
					o.awsProvider.logger.Warn("Failed to delete Route Table",
						zap.String("rtb_id", rtb.RouteTableID),
						zap.Error(err),
					)
				}
			}
		}

		// 4. Delete Subnets
		subnets, _ := o.subnet.FindByVPC(ctx, vpc.VPCID)
		for _, subnet := range subnets {
			if err := o.subnet.Delete(ctx, subnet.SubnetID, opts); err != nil {
				o.awsProvider.logger.Warn("Failed to delete Subnet",
					zap.String("subnet_id", subnet.SubnetID),
					zap.Error(err),
				)
			}
		}

		// 5. Delete Internet Gateway
		igws, _ := o.igw.FindByVPC(ctx, vpc.VPCID)
		for _, igw := range igws {
			if err := o.igw.Delete(ctx, igw.InternetGatewayID, vpc.VPCID, opts); err != nil {
				o.awsProvider.logger.Warn("Failed to delete Internet Gateway",
					zap.String("igw_id", igw.InternetGatewayID),
					zap.Error(err),
				)
			}
		}

		// 6. Delete VPC
		if err := o.vpc.Delete(ctx, vpc.VPCID, opts); err != nil {
			return fmt.Errorf("failed to delete VPC: %w", err)
		}
	}

	o.awsProvider.logger.Info("Tenant networking deleted", zap.String("tenant_id", tenantID))
	return nil
}

// GetTenantNetworking gets existing networking for a tenant
func (o *TenantNetworkingOrchestrator) GetTenantNetworking(
	ctx context.Context,
	tenantID string,
) (*NetworkingResult, error) {
	o.awsProvider.logger.Debug("Getting tenant networking", zap.String("tenant_id", tenantID))

	// Find VPC
	vpcs, err := o.vpc.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if len(vpcs) == 0 {
		return nil, fmt.Errorf("no networking found for tenant: %s", tenantID)
	}

	vpc := vpcs[0]
	result := &NetworkingResult{
		VPCID: vpc.VPCID,
	}

	// Get subnets
	subnets, _ := o.subnet.FindByTenant(ctx, tenantID)
	for _, subnet := range subnets {
		if subnet.IsPublic {
			result.PublicSubnetIDs = append(result.PublicSubnetIDs, subnet.SubnetID)
		} else {
			result.PrivateSubnetIDs = append(result.PrivateSubnetIDs, subnet.SubnetID)
		}
	}

	// Get Internet Gateway
	igws, _ := o.igw.FindByVPC(ctx, vpc.VPCID)
	if len(igws) > 0 {
		result.InternetGatewayID = igws[0].InternetGatewayID
	}

	// Get NAT Gateways
	natGws, _ := o.natGw.FindByVPC(ctx, vpc.VPCID)
	for _, natGw := range natGws {
		result.NATGatewayIDs = append(result.NATGatewayIDs, natGw.NATGatewayID)
	}

	// Get Security Groups
	sgs, _ := o.sg.FindByTenant(ctx, tenantID)
	for _, sg := range sgs {
		if strings.Contains(sg.Name, "default") {
			result.DefaultSecurityGroupID = sg.SecurityGroupID
			break
		}
	}

	// Get Route Tables
	rtbs, _ := o.routeTable.FindPankaManaged(ctx, vpc.VPCID)
	for _, rtb := range rtbs {
		if strings.Contains(rtb.Name, "public") {
			result.PublicRouteTableID = rtb.RouteTableID
		} else if strings.Contains(rtb.Name, "private") {
			result.PrivateRouteTableIDs = append(result.PrivateRouteTableIDs, rtb.RouteTableID)
		}
	}

	return result, nil
}

