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

// RouteTableProvider handles AWS Route Table operations
type RouteTableProvider struct {
	awsProvider *Provider
	ec2Client   *ec2.Client
}

// NewRouteTableProvider creates a new Route Table provider
func NewRouteTableProvider(p *Provider) *RouteTableProvider {
	return &RouteTableProvider{
		awsProvider: p,
		ec2Client:   ec2.NewFromConfig(p.GetConfig()),
	}
}

// RouteTableConfig represents Route Table configuration
type RouteTableConfig struct {
	VPCID    string
	Name     string
	Tags     map[string]string
	TenantID string
	IsPublic bool // Indicates if this is a public route table
}

// RouteConfig represents a route configuration
type RouteConfig struct {
	DestinationCidrBlock string
	GatewayID            string // For internet gateway
	NATGatewayID         string // For NAT gateway
	TransitGatewayID     string // For transit gateway
}

// RouteTableResult represents the result of a Route Table operation
type RouteTableResult struct {
	RouteTableID string
	VPCID        string
	Name         string
	Routes       []RouteConfig
	Associations []RouteTableAssociation
	Tags         map[string]string
	IsMain       bool
}

// RouteTableAssociation represents a route table association
type RouteTableAssociation struct {
	AssociationID string
	SubnetID      string
	IsMain        bool
}

// Create creates a new Route Table
func (r *RouteTableProvider) Create(ctx context.Context, config *RouteTableConfig, opts *provider.Options) (*RouteTableResult, error) {
	r.awsProvider.logger.Info("Creating Route Table",
		zap.String("name", config.Name),
		zap.String("vpc_id", config.VPCID),
		zap.String("tenant_id", config.TenantID),
	)

	if opts != nil && opts.DryRun {
		return &RouteTableResult{
			RouteTableID: "rtb-dry-run",
			VPCID:        config.VPCID,
			Name:         config.Name,
		}, nil
	}

	// Build tags
	routeTableType := "private"
	if config.IsPublic {
		routeTableType = "public"
	}

	tags := r.buildTags(config.Tags, config.TenantID, "RouteTable", config.Name, map[string]string{
		"panka-route-table-type": routeTableType,
	})

	// Create Route Table
	input := &ec2.CreateRouteTableInput{
		VpcId: aws.String(config.VPCID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeRouteTable,
				Tags:         tags,
			},
		},
	}

	result, err := r.ec2Client.CreateRouteTable(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create route table: %w", err)
	}

	rtbID := *result.RouteTable.RouteTableId
	r.awsProvider.logger.Info("Route Table created", zap.String("rtb_id", rtbID))

	return &RouteTableResult{
		RouteTableID: rtbID,
		VPCID:        config.VPCID,
		Name:         config.Name,
		Routes:       []RouteConfig{},
		Associations: []RouteTableAssociation{},
		Tags:         config.Tags,
	}, nil
}

// Get retrieves Route Table information
func (r *RouteTableProvider) Get(ctx context.Context, rtbID string) (*RouteTableResult, error) {
	r.awsProvider.logger.Debug("Getting Route Table", zap.String("rtb_id", rtbID))

	result, err := r.ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		RouteTableIds: []string{rtbID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe route table: %w", err)
	}

	if len(result.RouteTables) == 0 {
		return nil, fmt.Errorf("route table not found: %s", rtbID)
	}

	rtb := result.RouteTables[0]
	return r.convertRouteTable(&rtb), nil
}

// Delete deletes a Route Table
func (r *RouteTableProvider) Delete(ctx context.Context, rtbID string, opts *provider.Options) error {
	r.awsProvider.logger.Info("Deleting Route Table", zap.String("rtb_id", rtbID))

	if opts != nil && opts.DryRun {
		return nil
	}

	// First, disassociate all subnets
	rtb, err := r.Get(ctx, rtbID)
	if err != nil {
		return err
	}

	for _, assoc := range rtb.Associations {
		if !assoc.IsMain && assoc.AssociationID != "" {
			_, err := r.ec2Client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
				AssociationId: aws.String(assoc.AssociationID),
			})
			if err != nil {
				r.awsProvider.logger.Warn("Failed to disassociate route table",
					zap.String("association_id", assoc.AssociationID),
					zap.Error(err),
				)
			}
		}
	}

	// Delete the route table
	_, err = r.ec2Client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
		RouteTableId: aws.String(rtbID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete route table: %w", err)
	}

	r.awsProvider.logger.Info("Route Table deleted", zap.String("rtb_id", rtbID))
	return nil
}

// AddRoute adds a route to a route table
func (r *RouteTableProvider) AddRoute(ctx context.Context, rtbID string, route RouteConfig) error {
	r.awsProvider.logger.Debug("Adding route to route table",
		zap.String("rtb_id", rtbID),
		zap.String("destination", route.DestinationCidrBlock),
	)

	input := &ec2.CreateRouteInput{
		RouteTableId:         aws.String(rtbID),
		DestinationCidrBlock: aws.String(route.DestinationCidrBlock),
	}

	if route.GatewayID != "" {
		input.GatewayId = aws.String(route.GatewayID)
	}
	if route.NATGatewayID != "" {
		input.NatGatewayId = aws.String(route.NATGatewayID)
	}
	if route.TransitGatewayID != "" {
		input.TransitGatewayId = aws.String(route.TransitGatewayID)
	}

	_, err := r.ec2Client.CreateRoute(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to add route: %w", err)
	}

	r.awsProvider.logger.Debug("Route added",
		zap.String("rtb_id", rtbID),
		zap.String("destination", route.DestinationCidrBlock),
	)

	return nil
}

// DeleteRoute deletes a route from a route table
func (r *RouteTableProvider) DeleteRoute(ctx context.Context, rtbID string, destinationCidr string) error {
	r.awsProvider.logger.Debug("Deleting route from route table",
		zap.String("rtb_id", rtbID),
		zap.String("destination", destinationCidr),
	)

	_, err := r.ec2Client.DeleteRoute(ctx, &ec2.DeleteRouteInput{
		RouteTableId:         aws.String(rtbID),
		DestinationCidrBlock: aws.String(destinationCidr),
	})
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}

	return nil
}

// AssociateSubnet associates a subnet with a route table
func (r *RouteTableProvider) AssociateSubnet(ctx context.Context, rtbID string, subnetID string) (string, error) {
	r.awsProvider.logger.Debug("Associating subnet with route table",
		zap.String("rtb_id", rtbID),
		zap.String("subnet_id", subnetID),
	)

	result, err := r.ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(rtbID),
		SubnetId:     aws.String(subnetID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to associate subnet: %w", err)
	}

	associationID := *result.AssociationId
	r.awsProvider.logger.Debug("Subnet associated",
		zap.String("rtb_id", rtbID),
		zap.String("subnet_id", subnetID),
		zap.String("association_id", associationID),
	)

	return associationID, nil
}

// DisassociateSubnet disassociates a subnet from a route table
func (r *RouteTableProvider) DisassociateSubnet(ctx context.Context, associationID string) error {
	r.awsProvider.logger.Debug("Disassociating subnet from route table",
		zap.String("association_id", associationID),
	)

	_, err := r.ec2Client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
		AssociationId: aws.String(associationID),
	})
	if err != nil {
		return fmt.Errorf("failed to disassociate subnet: %w", err)
	}

	return nil
}

// FindByVPC finds Route Tables by VPC ID
func (r *RouteTableProvider) FindByVPC(ctx context.Context, vpcID string) ([]RouteTableResult, error) {
	r.awsProvider.logger.Debug("Finding Route Tables by VPC", zap.String("vpc_id", vpcID))

	result, err := r.ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find route tables: %w", err)
	}

	rtbs := make([]RouteTableResult, 0, len(result.RouteTables))
	for _, rtb := range result.RouteTables {
		rtbs = append(rtbs, *r.convertRouteTable(&rtb))
	}

	return rtbs, nil
}

// FindPankaManaged finds Route Tables managed by Panka
func (r *RouteTableProvider) FindPankaManaged(ctx context.Context, vpcID string) ([]RouteTableResult, error) {
	r.awsProvider.logger.Debug("Finding Panka-managed Route Tables", zap.String("vpc_id", vpcID))

	result, err := r.ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
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
		return nil, fmt.Errorf("failed to find route tables: %w", err)
	}

	rtbs := make([]RouteTableResult, 0, len(result.RouteTables))
	for _, rtb := range result.RouteTables {
		rtbs = append(rtbs, *r.convertRouteTable(&rtb))
	}

	return rtbs, nil
}

// GetMainRouteTable gets the main route table for a VPC
func (r *RouteTableProvider) GetMainRouteTable(ctx context.Context, vpcID string) (*RouteTableResult, error) {
	r.awsProvider.logger.Debug("Getting main Route Table", zap.String("vpc_id", vpcID))

	result, err := r.ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
			{
				Name:   aws.String("association.main"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find main route table: %w", err)
	}

	if len(result.RouteTables) == 0 {
		return nil, fmt.Errorf("main route table not found for VPC: %s", vpcID)
	}

	return r.convertRouteTable(&result.RouteTables[0]), nil
}

// convertRouteTable converts AWS RouteTable to RouteTableResult
func (r *RouteTableProvider) convertRouteTable(rtb *types.RouteTable) *RouteTableResult {
	tags := make(map[string]string)
	var name string
	for _, tag := range rtb.Tags {
		tags[*tag.Key] = *tag.Value
		if *tag.Key == "Name" {
			name = *tag.Value
		}
	}

	// Convert routes
	routes := make([]RouteConfig, 0)
	for _, route := range rtb.Routes {
		if route.DestinationCidrBlock == nil {
			continue
		}

		rc := RouteConfig{
			DestinationCidrBlock: *route.DestinationCidrBlock,
		}
		if route.GatewayId != nil {
			rc.GatewayID = *route.GatewayId
		}
		if route.NatGatewayId != nil {
			rc.NATGatewayID = *route.NatGatewayId
		}
		if route.TransitGatewayId != nil {
			rc.TransitGatewayID = *route.TransitGatewayId
		}
		routes = append(routes, rc)
	}

	// Convert associations
	associations := make([]RouteTableAssociation, 0)
	isMain := false
	for _, assoc := range rtb.Associations {
		rta := RouteTableAssociation{}
		if assoc.RouteTableAssociationId != nil {
			rta.AssociationID = *assoc.RouteTableAssociationId
		}
		if assoc.SubnetId != nil {
			rta.SubnetID = *assoc.SubnetId
		}
		if assoc.Main != nil && *assoc.Main {
			rta.IsMain = true
			isMain = true
		}
		associations = append(associations, rta)
	}

	return &RouteTableResult{
		RouteTableID: *rtb.RouteTableId,
		VPCID:        *rtb.VpcId,
		Name:         name,
		Routes:       routes,
		Associations: associations,
		Tags:         tags,
		IsMain:       isMain,
	}
}

// buildTags builds tags for Route Table resources
func (r *RouteTableProvider) buildTags(customTags map[string]string, tenantID, resourceType, name string, extraTags map[string]string) []types.Tag {
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

	if r.awsProvider.tagHelper != nil {
		for k, val := range r.awsProvider.tagHelper.DefaultTags {
			tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
		}
	}

	for k, val := range extraTags {
		tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
	}

	for k, val := range customTags {
		tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(val)})
	}

	return tags
}

