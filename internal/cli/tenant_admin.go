package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yourusername/panka/pkg/provider"
	"github.com/yourusername/panka/pkg/provider/aws"
	"github.com/yourusername/panka/pkg/tenant"
)

var (
	tenantName         string
	tenantDisplayName  string
	tenantEmail        string
	tenantAWSAccount   string
	tenantAWSRegion    string
	tenantVersion      string
	tenantCostTracking bool
	tenantCostLimit    int
	tenantMaxStacks    int
	tenantMaxServicesPerStack int
	tenantMaxResourcesPerService int

	// Networking flags
	tenantVPCCidr            string
	tenantEnableNATGateway   bool
	tenantNATGatewayType     string
	tenantAZs                []string
	tenantOutputFile         string
	tenantCreateNetworking   bool
	tenantNetworkingDryRun   bool
)

// tenantAdminCmd represents the tenant command for admin operations
var tenantAdminCmd = &cobra.Command{
	Use:   "tenant",
	Short: "Manage tenants (admin only)",
	Long: `Manage tenants in the multi-tenant system.

Admin commands for tenant lifecycle management:
  • Create new tenants
  • List all tenants
  • Show tenant details
  • Rotate credentials
  • Suspend/activate tenants`,
}

// tenantInitCmd creates a new tenant
var tenantInitCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Create a new tenant with networking",
	Long: `Create a new tenant with isolated state, credentials, and networking.

The tenant's networking configuration (VPC, subnets, NAT gateway, security groups)
is shared by all stacks within the tenant. This provides:

  • Cost efficiency (shared NAT gateway)
  • Easy service communication (same VPC)
  • Security isolation (tenant-specific VPC)

Examples:
  # Create tenant with default networking (10.0.0.0/16)
  panka admin tenant init notifications-team \
    --aws-account 123456789012 \
    --region us-east-1 \
    --vpc-cidr 10.0.0.0/16 \
    --nat-gateway

  # Create tenant with specific AZs
  panka admin tenant init payments-team \
    --aws-account 123456789012 \
    --region us-east-1 \
    --vpc-cidr 10.1.0.0/16 \
    --azs us-east-1a,us-east-1b,us-east-1c \
    --nat-gateway \
    --nat-type per-az

  # Save credentials to file
  panka admin tenant init my-team \
    --vpc-cidr 10.0.0.0/16 \
    --output credentials.txt`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTenantInit,
}

// tenantListCmd lists all tenants
var tenantListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tenants",
	Long:  `List all tenants in the system.`,
	RunE:  runTenantList,
}

// tenantShowCmd shows tenant details
var tenantShowCmd = &cobra.Command{
	Use:   "show <tenant-id>",
	Short: "Show tenant details",
	Long:  `Show detailed information about a specific tenant.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTenantShow,
}

// tenantRotateCmd rotates tenant credentials
var tenantRotateCmd = &cobra.Command{
	Use:   "rotate <tenant-id>",
	Short: "Rotate tenant credentials",
	Long:  `Generate new credentials for a tenant and invalidate the old ones.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTenantRotate,
}

// tenantSuspendCmd suspends a tenant
var tenantSuspendCmd = &cobra.Command{
	Use:   "suspend <tenant-id>",
	Short: "Suspend a tenant",
	Long:  `Suspend a tenant to prevent new logins (existing sessions remain valid).`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTenantSuspend,
}

// tenantActivateCmd activates a tenant
var tenantActivateCmd = &cobra.Command{
	Use:   "activate <tenant-id>",
	Short: "Activate a suspended tenant",
	Long:  `Activate a previously suspended tenant.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTenantActivate,
}

func init() {
	adminCmd.AddCommand(tenantAdminCmd)
	tenantAdminCmd.AddCommand(tenantInitCmd)
	tenantAdminCmd.AddCommand(tenantListCmd)
	tenantAdminCmd.AddCommand(tenantShowCmd)
	tenantAdminCmd.AddCommand(tenantRotateCmd)
	tenantAdminCmd.AddCommand(tenantSuspendCmd)
	tenantAdminCmd.AddCommand(tenantActivateCmd)
	
	// Basic tenant flags
	tenantInitCmd.Flags().StringVar(&tenantName, "name", "", "Tenant name (can also be provided as argument)")
	tenantInitCmd.Flags().StringVar(&tenantDisplayName, "display-name", "", "Display name")
	tenantInitCmd.Flags().StringVar(&tenantEmail, "email", "", "Contact email")
	tenantInitCmd.Flags().StringVar(&tenantVersion, "version", "v1", "State version")

	// AWS flags
	tenantInitCmd.Flags().StringVar(&tenantAWSAccount, "aws-account", "", "AWS account ID")
	tenantInitCmd.Flags().StringVar(&tenantAWSRegion, "region", "", "AWS region (e.g., us-east-1)")

	// Networking flags
	tenantInitCmd.Flags().StringVar(&tenantVPCCidr, "vpc-cidr", "", "VPC CIDR block (e.g., 10.0.0.0/16)")
	tenantInitCmd.Flags().BoolVar(&tenantEnableNATGateway, "nat-gateway", false, "Enable NAT Gateway for private subnets")
	tenantInitCmd.Flags().StringVar(&tenantNATGatewayType, "nat-type", "single", "NAT Gateway type: single (cost-effective) or per-az (high availability)")
	tenantInitCmd.Flags().StringSliceVar(&tenantAZs, "azs", nil, "Availability zones (e.g., us-east-1a,us-east-1b)")

	// Limits flags
	tenantInitCmd.Flags().BoolVar(&tenantCostTracking, "cost-tracking", true, "Enable cost tracking")
	tenantInitCmd.Flags().IntVar(&tenantCostLimit, "cost-limit", 5000, "Monthly cost limit (USD, 0 for unlimited)")
	tenantInitCmd.Flags().IntVar(&tenantMaxStacks, "max-stacks", 10, "Maximum number of stacks")
	tenantInitCmd.Flags().IntVar(&tenantMaxServicesPerStack, "max-services-per-stack", 20, "Maximum services per stack")
	tenantInitCmd.Flags().IntVar(&tenantMaxResourcesPerService, "max-resources-per-service", 50, "Maximum resources per service")

	// Output flags
	tenantInitCmd.Flags().StringVarP(&tenantOutputFile, "output", "o", "", "Write credentials to file")

	// Networking provisioning flags
	tenantInitCmd.Flags().BoolVar(&tenantCreateNetworking, "create-networking", false, "Create AWS networking resources (VPC, subnets, NAT, etc.)")
	tenantInitCmd.Flags().BoolVar(&tenantNetworkingDryRun, "dry-run", false, "Show what would be created without actually creating")
}

func runTenantInit(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)

	// Require admin session
	sessionMgr := tenant.NewSessionManager()
	session, err := sessionMgr.RequireAdminSession()
	if err != nil {
		return err
	}

	// Get tenant name from args or flag
	if len(args) > 0 {
		tenantName = args[0]
	}

	cyan.Println("\n🏢 Create New Tenant with Networking")
	cyan.Println(strings.Repeat("─", 60))

	reader := bufio.NewReader(os.Stdin)

	// Interactive prompts if not provided via flags
	if tenantName == "" {
		fmt.Print("\n? Tenant Name (lowercase, alphanumeric, hyphens): ")
		input, _ := reader.ReadString('\n')
		tenantName = strings.TrimSpace(input)
	}

	if tenantDisplayName == "" {
		fmt.Printf("? Display Name [%s]: ", tenantName)
		input, _ := reader.ReadString('\n')
		tenantDisplayName = strings.TrimSpace(input)
		if tenantDisplayName == "" {
			tenantDisplayName = tenantName
		}
	}

	if tenantEmail == "" {
		fmt.Printf("? Contact Email: ")
		input, _ := reader.ReadString('\n')
		tenantEmail = strings.TrimSpace(input)
	}

	// Region - prompt if not provided
	if tenantAWSRegion == "" {
		tenantAWSRegion = session.Backend.Region
		fmt.Printf("? AWS Region [%s]: ", tenantAWSRegion)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			tenantAWSRegion = input
		}
	}

	// VPC CIDR - prompt if not provided
	if tenantVPCCidr == "" {
		fmt.Print("? VPC CIDR Block [10.0.0.0/16]: ")
		input, _ := reader.ReadString('\n')
		tenantVPCCidr = strings.TrimSpace(input)
		if tenantVPCCidr == "" {
			tenantVPCCidr = "10.0.0.0/16"
		}
	}

	// NAT Gateway - prompt if not provided via flag
	if !tenantEnableNATGateway {
		fmt.Print("? Enable NAT Gateway for private subnets? (yes/no) [yes]: ")
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))
		tenantEnableNATGateway = input == "" || input == "yes" || input == "y"
	}

	// Create tenant manager
	backend, err := tenant.NewS3RegistryBackend(session.Backend.Bucket, session.Backend.Region)
	if err != nil {
		return fmt.Errorf("failed to create backend: %w", err)
	}

	manager := tenant.NewManager(backend)

	// Load registry
	ctx := context.Background()
	if err := manager.LoadRegistry(ctx); err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Create tenant request
	req := &tenant.CreateTenantRequest{
		Name:             tenantName,
		DisplayName:      tenantDisplayName,
		Email:            tenantEmail,
		AWSAccountID:     tenantAWSAccount,
		AWSRegion:        tenantAWSRegion,
		Version:          tenantVersion,

		// Networking
		VPCCidr:           tenantVPCCidr,
		EnableNATGateway:  tenantEnableNATGateway,
		NATGatewayType:    tenantNATGatewayType,
		AvailabilityZones: tenantAZs,

		// Limits
		CostTracking:           tenantCostTracking,
		MonthlyCostLimit:       tenantCostLimit,
		MaxStacks:              tenantMaxStacks,
		MaxServicesPerStack:    tenantMaxServicesPerStack,
		MaxResourcesPerService: tenantMaxResourcesPerService,

		// Tags
		DefaultTags: map[string]string{
			"tenant":     tenantName,
			"managed-by": "panka",
		},
		Metadata: make(map[string]string),
	}

	// Create tenant
	fmt.Println("\nCreating tenant...")
	fmt.Println("├── Validating tenant name...")
	fmt.Println("├── Checking for conflicts...")
	fmt.Println("├── Generating tenant ID...")
	fmt.Println("├── Generating secure credentials...")
	fmt.Println("├── Configuring networking...")

	newTenant, creds, err := manager.CreateTenant(ctx, req)
	if err != nil {
		red.Printf("✗ Failed to create tenant: %v\n", err)
		return err
	}

	fmt.Println("├── Creating S3 directory structure...")
	if err := backend.CreateTenantDirectory(ctx, newTenant); err != nil {
		yellow.Printf("⚠️  Warning: Failed to create tenant directory: %v\n", err)
	}

	// Provision AWS networking if requested
	if tenantCreateNetworking {
		fmt.Println("├── Provisioning AWS networking resources...")

		networkResult, err := provisionTenantNetworking(ctx, newTenant, tenantNetworkingDryRun)
		if err != nil {
			red.Printf("✗ Failed to provision networking: %v\n", err)
			yellow.Println("   The tenant was created but networking resources were not provisioned.")
			yellow.Println("   You can retry with: panka admin tenant provision-networking " + newTenant.ID)
			return err
		}

		// Store resource IDs in tenant
		newTenant.Networking.ResourceIDs = &tenant.NetworkingResourceIDs{
			VPCID:                networkResult.VPCID,
			InternetGatewayID:    networkResult.InternetGatewayID,
			NATGatewayIDs:        networkResult.NATGatewayIDs,
			PublicSubnetIDs:      networkResult.PublicSubnetIDs,
			PrivateSubnetIDs:     networkResult.PrivateSubnetIDs,
			SecurityGroupID:      networkResult.DefaultSecurityGroupID,
			PublicRouteTableID:   networkResult.PublicRouteTableID,
			PrivateRouteTableIDs: networkResult.PrivateRouteTableIDs,
		}

		// Update tenant in S3
		if !tenantNetworkingDryRun {
			if err := manager.UpdateTenant(ctx, newTenant); err != nil {
				yellow.Printf("⚠️  Warning: Failed to save networking resource IDs: %v\n", err)
			}
		}

		if tenantNetworkingDryRun {
			fmt.Println("│   └── [DRY-RUN] Would create networking resources")
		} else {
			green.Println("│   └── Networking resources created ✓")
		}
	}

	fmt.Println("└── Tenant created successfully ✓")

	// Display summary
	cyan.Println("\n" + strings.Repeat("─", 60))
	green.Println("✓ Tenant Created")

	cyan.Println("\n📋 Tenant Details:")
	fmt.Printf("  Tenant ID:      %s\n", creds.TenantID)
	fmt.Printf("  Display Name:   %s\n", newTenant.DisplayName)
	fmt.Printf("  S3 Path:        s3://%s/%s\n", session.Backend.Bucket, newTenant.Storage.Path)

	cyan.Println("\n🌐 Networking Configuration:")
	fmt.Printf("  VPC CIDR:       %s\n", newTenant.Networking.VPC.CidrBlock)
	fmt.Printf("  Region:         %s\n", newTenant.AWS.Region)

	if len(newTenant.Networking.Subnets.Public) > 0 {
		fmt.Println("  Public Subnets:")
		for _, subnet := range newTenant.Networking.Subnets.Public {
			fmt.Printf("    - %s (%s)\n", subnet.CidrBlock, subnet.AvailabilityZone)
		}
	}

	if len(newTenant.Networking.Subnets.Private) > 0 {
		fmt.Println("  Private Subnets:")
		for _, subnet := range newTenant.Networking.Subnets.Private {
			fmt.Printf("    - %s (%s)\n", subnet.CidrBlock, subnet.AvailabilityZone)
		}
	}

	if newTenant.Networking.NATGateway.Enabled {
		fmt.Printf("  NAT Gateway:    Enabled (%s)\n", newTenant.Networking.NATGateway.Type)
	} else {
		fmt.Println("  NAT Gateway:    Disabled")
	}

	fmt.Printf("  Security Group: Allow internal traffic = %v\n", newTenant.Networking.DefaultSecurityGroup.AllowInternalTraffic)

	// Show resource IDs if networking was provisioned
	if newTenant.Networking.ResourceIDs != nil && newTenant.Networking.ResourceIDs.VPCID != "" {
		cyan.Println("\n🔗 AWS Resource IDs:")
		fmt.Printf("  VPC:              %s\n", newTenant.Networking.ResourceIDs.VPCID)
		if newTenant.Networking.ResourceIDs.InternetGatewayID != "" {
			fmt.Printf("  Internet Gateway: %s\n", newTenant.Networking.ResourceIDs.InternetGatewayID)
		}
		if len(newTenant.Networking.ResourceIDs.PublicSubnetIDs) > 0 {
			fmt.Printf("  Public Subnets:   %v\n", newTenant.Networking.ResourceIDs.PublicSubnetIDs)
		}
		if len(newTenant.Networking.ResourceIDs.PrivateSubnetIDs) > 0 {
			fmt.Printf("  Private Subnets:  %v\n", newTenant.Networking.ResourceIDs.PrivateSubnetIDs)
		}
		if len(newTenant.Networking.ResourceIDs.NATGatewayIDs) > 0 {
			fmt.Printf("  NAT Gateways:     %v\n", newTenant.Networking.ResourceIDs.NATGatewayIDs)
		}
		fmt.Printf("  Security Group:   %s\n", newTenant.Networking.ResourceIDs.SecurityGroupID)
	}

	cyan.Println("\n📊 Limits:")
	fmt.Printf("  Max Stacks:              %d\n", newTenant.Limits.MaxStacks)
	fmt.Printf("  Max Services/Stack:      %d\n", newTenant.Limits.MaxServicesPerStack)
	fmt.Printf("  Max Resources/Service:   %d\n", newTenant.Limits.MaxResourcesPerService)

	cyan.Println("\n🔐 Credentials:")
	yellow.Printf("  Tenant ID:     %s\n", creds.TenantID)
	yellow.Printf("  Tenant Secret: %s\n", creds.Secret)
	yellow.Println("                 " + strings.Repeat("^", len(creds.Secret)))
	yellow.Println("                 SAVE THIS - IT CANNOT BE RECOVERED")

	// Write to output file if specified
	if tenantOutputFile != "" {
		credContent := fmt.Sprintf(`# Panka Tenant Credentials
# Generated: %s
# KEEP THIS FILE SECURE

Tenant ID: %s
Secret: %s

# Share these details with your team:
Bucket: %s
Region: %s

# Login command:
# panka login
`, 
			newTenant.Created.Format("2006-01-02 15:04:05"),
			creds.TenantID,
			creds.Secret,
			session.Backend.Bucket,
			session.Backend.Region,
		)
		if err := os.WriteFile(tenantOutputFile, []byte(credContent), 0600); err != nil {
			yellow.Printf("⚠️  Warning: Failed to write credentials to file: %v\n", err)
		} else {
			green.Printf("\n✓ Credentials saved to: %s\n", tenantOutputFile)
		}
	}

	cyan.Println("\n" + strings.Repeat("─", 60))

	cyan.Println("\n📝 Next Steps:")
	fmt.Println("  1. Share credentials with your team")
	fmt.Println("  2. Team members login:  panka login")
	fmt.Println("  3. Create stack folder: panka stack init my-stack --services api,worker")
	fmt.Println("  4. Deploy:             panka apply ./my-stack")

	yellow.Println("\n⚠️  IMPORTANT: Store the tenant secret securely.")
	yellow.Println("   It cannot be retrieved later. If lost, use 'panka admin tenant rotate'.")

	return nil
}

func runTenantList(cmd *cobra.Command, args []string) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	// Require admin session
	sessionMgr := tenant.NewSessionManager()
	session, err := sessionMgr.RequireAdminSession()
	if err != nil {
		return err
	}

	// Create tenant manager
	backend, err := tenant.NewS3RegistryBackend(session.Backend.Bucket, session.Backend.Region)
	if err != nil {
		return fmt.Errorf("failed to create backend: %w", err)
	}

	manager := tenant.NewManager(backend)
	
	// Load registry
	ctx := context.Background()
	if err := manager.LoadRegistry(ctx); err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	tenants := manager.ListTenants()

	cyan.Println("\n📋 Tenants")
	cyan.Println(strings.Repeat("─", 80))
	
	if len(tenants) == 0 {
		yellow.Println("No tenants found")
		return nil
	}

	// Table header
	fmt.Printf("%-20s %-25s %-12s %-10s\n", "ID", "NAME", "STATUS", "CREATED")
	cyan.Println(strings.Repeat("─", 80))

	// Table rows
	for _, t := range tenants {
		statusColor := green
		statusSymbol := "●"
		
		switch t.Status {
		case tenant.StatusActive:
			statusColor = green
			statusSymbol = "✓"
		case tenant.StatusSuspended:
			statusColor = yellow
			statusSymbol = "⏸"
		case tenant.StatusDeleted:
			statusColor = red
			statusSymbol = "✗"
		}
		
		fmt.Printf("%-20s %-25s ", t.ID, t.DisplayName)
		statusColor.Printf("%-12s", fmt.Sprintf("%s %s", statusSymbol, t.Status))
		fmt.Printf(" %-10s\n", t.Created.Format("2006-01-02"))
	}

	cyan.Println(strings.Repeat("─", 80))
	fmt.Printf("Total: %d tenants\n", len(tenants))
	
	return nil
}

func runTenantShow(cmd *cobra.Command, args []string) error {
	tenantID := args[0]
	
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	// Require admin session
	sessionMgr := tenant.NewSessionManager()
	session, err := sessionMgr.RequireAdminSession()
	if err != nil {
		return err
	}

	// Create tenant manager
	backend, err := tenant.NewS3RegistryBackend(session.Backend.Bucket, session.Backend.Region)
	if err != nil {
		return fmt.Errorf("failed to create backend: %w", err)
	}

	manager := tenant.NewManager(backend)
	
	// Load registry
	ctx := context.Background()
	if err := manager.LoadRegistry(ctx); err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	t := manager.GetTenant(tenantID)
	if t == nil {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}

	cyan.Printf("\n📋 Tenant: %s\n", t.ID)
	cyan.Println(strings.Repeat("─", 50))

	fmt.Printf("\nDisplay Name:  %s\n", t.DisplayName)
	fmt.Printf("Email:         %s\n", t.Email)
	
	// Status with color
	fmt.Print("Status:        ")
	switch t.Status {
	case tenant.StatusActive:
		green.Println("✓ Active")
	case tenant.StatusSuspended:
		yellow.Println("⏸ Suspended")
	case tenant.StatusDeleted:
		red.Println("✗ Deleted")
	}
	
	fmt.Printf("Created:       %s\n", t.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:       %s\n", t.Updated.Format("2006-01-02 15:04:05"))

	cyan.Println("\nStorage:")
	fmt.Printf("  Bucket:      %s\n", session.Backend.Bucket)
	fmt.Printf("  Prefix:      %s\n", t.Storage.Prefix)
	fmt.Printf("  Version:     %s\n", t.Storage.Version)

	cyan.Println("\nCredentials:")
	fmt.Printf("  Rotations:   %d\n", t.Credentials.Rotations)
	if t.Credentials.LastRotated != nil {
		fmt.Printf("  Last Rotated: %s\n", t.Credentials.LastRotated.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("  Last Rotated: Never")
	}

	cyan.Println("\nLimits:")
	fmt.Printf("  Cost Tracking:         %v\n", t.Limits.CostTracking)
	if t.Limits.MonthlyCostLimit > 0 {
		fmt.Printf("  Cost Limit:            $%d/month\n", t.Limits.MonthlyCostLimit)
	} else {
		fmt.Println("  Cost Limit:            Unlimited")
	}
	fmt.Printf("  Max Stacks:            %d\n", t.Limits.MaxStacks)
	fmt.Printf("  Max Services/Stack:    %d\n", t.Limits.MaxServicesPerStack)
	fmt.Printf("  Max Resources/Service: %d\n", t.Limits.MaxResourcesPerService)

	if t.AWS.AccountID != "" || t.AWS.Region != "" {
		cyan.Println("\nAWS:")
		if t.AWS.AccountID != "" {
			fmt.Printf("  Account ID:  %s\n", t.AWS.AccountID)
		}
		if t.AWS.Region != "" {
			fmt.Printf("  Region:      %s\n", t.AWS.Region)
		}
		if t.AWS.AssumeRoleArn != "" {
			fmt.Printf("  Role ARN:    %s\n", t.AWS.AssumeRoleArn)
		}
	}

	// Display networking configuration
	if t.Networking.VPC.CidrBlock != "" {
		cyan.Println("\nNetworking:")
		fmt.Printf("  VPC CIDR:      %s\n", t.Networking.VPC.CidrBlock)

		if len(t.Networking.Subnets.Public) > 0 {
			fmt.Println("  Public Subnets:")
			for _, subnet := range t.Networking.Subnets.Public {
				fmt.Printf("    - %s (%s)\n", subnet.CidrBlock, subnet.AvailabilityZone)
			}
		}

		if len(t.Networking.Subnets.Private) > 0 {
			fmt.Println("  Private Subnets:")
			for _, subnet := range t.Networking.Subnets.Private {
				fmt.Printf("    - %s (%s)\n", subnet.CidrBlock, subnet.AvailabilityZone)
			}
		}

		if t.Networking.NATGateway.Enabled {
			fmt.Printf("  NAT Gateway:   Enabled (%s)\n", t.Networking.NATGateway.Type)
		} else {
			fmt.Println("  NAT Gateway:   Disabled")
		}

		fmt.Printf("  Internal Traffic: %v\n", t.Networking.DefaultSecurityGroup.AllowInternalTraffic)

		// Show resource IDs if created
		if t.Networking.ResourceIDs != nil && t.Networking.ResourceIDs.VPCID != "" {
			cyan.Println("\nAWS Resources (Created):")
			fmt.Printf("  VPC ID:        %s\n", t.Networking.ResourceIDs.VPCID)
			if t.Networking.ResourceIDs.SecurityGroupID != "" {
				fmt.Printf("  Security Group: %s\n", t.Networking.ResourceIDs.SecurityGroupID)
			}
		}
	}

	// Display default tags
	if len(t.DefaultTags) > 0 {
		cyan.Println("\nDefault Tags:")
		for k, v := range t.DefaultTags {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	cyan.Println(strings.Repeat("─", 50))

	return nil
}

func runTenantRotate(cmd *cobra.Command, args []string) error {
	tenantID := args[0]
	
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)

	// Require admin session
	sessionMgr := tenant.NewSessionManager()
	session, err := sessionMgr.RequireAdminSession()
	if err != nil {
		return err
	}

	// Create tenant manager
	backend, err := tenant.NewS3RegistryBackend(session.Backend.Bucket, session.Backend.Region)
	if err != nil {
		return fmt.Errorf("failed to create backend: %w", err)
	}

	manager := tenant.NewManager(backend)
	
	// Load registry
	ctx := context.Background()
	if err := manager.LoadRegistry(ctx); err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	t := manager.GetTenant(tenantID)
	if t == nil {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}

	cyan.Println("\n🔄 Rotate Tenant Credentials")
	cyan.Println(strings.Repeat("─", 50))
	
	fmt.Printf("\nTenant:           %s (%s)\n", t.ID, t.DisplayName)
	fmt.Printf("Current rotations: %d\n", t.Credentials.Rotations)
	if t.Credentials.LastRotated != nil {
		fmt.Printf("Last rotated:      %s\n", t.Credentials.LastRotated.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("Last rotated:      Never")
	}

	yellow.Println("\n⚠️  This will invalidate the current tenant secret.")
	yellow.Println("   All team members will need to re-authenticate.")
	
	fmt.Print("\nContinue? (yes/no): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))
	
	if input != "yes" && input != "y" {
		fmt.Println("Cancelled")
		return nil
	}

	// Rotate credentials
	fmt.Println("\nRotating credentials...")
	creds, err := manager.RotateTenantCredentials(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to rotate credentials: %w", err)
	}

	cyan.Println("\n" + strings.Repeat("─", 50))
	green.Println("✓ Credentials Rotated")
	
	fmt.Println("\nNew Tenant Secret:", creds.Secret)
	yellow.Println("                  ", strings.Repeat("^", len(creds.Secret)))
	yellow.Println("                   SHARE WITH TEAM SECURELY")
	
	fmt.Printf("\nRotations:         %d\n", t.Credentials.Rotations+1)
	
	cyan.Println(strings.Repeat("─", 50))

	return nil
}

func runTenantSuspend(cmd *cobra.Command, args []string) error {
	tenantID := args[0]
	
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow)

	// Require admin session
	sessionMgr := tenant.NewSessionManager()
	session, err := sessionMgr.RequireAdminSession()
	if err != nil {
		return err
	}

	// Create tenant manager
	backend, err := tenant.NewS3RegistryBackend(session.Backend.Bucket, session.Backend.Region)
	if err != nil {
		return fmt.Errorf("failed to create backend: %w", err)
	}

	manager := tenant.NewManager(backend)
	
	// Load registry
	ctx := context.Background()
	if err := manager.LoadRegistry(ctx); err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Suspend tenant
	fmt.Printf("Suspending tenant: %s\n", tenantID)
	if err := manager.SuspendTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to suspend tenant: %w", err)
	}

	green.Printf("✓ Tenant suspended: %s\n", tenantID)
	yellow.Println("  Existing sessions will remain valid until expiry.")
	yellow.Println("  New logins are now blocked.")

	return nil
}

func runTenantActivate(cmd *cobra.Command, args []string) error {
	tenantID := args[0]
	
	green := color.New(color.FgGreen, color.Bold)

	// Require admin session
	sessionMgr := tenant.NewSessionManager()
	session, err := sessionMgr.RequireAdminSession()
	if err != nil {
		return err
	}

	// Create tenant manager
	backend, err := tenant.NewS3RegistryBackend(session.Backend.Bucket, session.Backend.Region)
	if err != nil {
		return fmt.Errorf("failed to create backend: %w", err)
	}

	manager := tenant.NewManager(backend)
	
	// Load registry
	ctx := context.Background()
	if err := manager.LoadRegistry(ctx); err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Activate tenant
	fmt.Printf("Activating tenant: %s\n", tenantID)
	if err := manager.ActivateTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to activate tenant: %w", err)
	}

	green.Printf("✓ Tenant activated: %s\n", tenantID)
	green.Println("  Team members can now login.")

	return nil
}

// provisionTenantNetworking creates AWS networking resources for a tenant
func provisionTenantNetworking(ctx context.Context, t *tenant.Tenant, dryRun bool) (*aws.NetworkingResult, error) {
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)

	// Initialize AWS provider
	awsProvider := aws.NewProvider()
	err := awsProvider.Initialize(ctx, &provider.Config{
		Name:   "aws",
		Region: t.AWS.Region,
		DefaultTags: map[string]string{
			"tenant":     t.ID,
			"managed-by": "panka",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS provider: %w", err)
	}
	defer awsProvider.Close()

	// Create orchestrator
	orchestrator := aws.NewTenantNetworkingOrchestrator(awsProvider)

	// Build networking config
	networkingConfig := &t.Networking

	// Log what we're creating
	cyan.Println("\n   Creating AWS Resources:")
	fmt.Printf("   ├── VPC: %s\n", networkingConfig.VPC.CidrBlock)
	fmt.Printf("   ├── Region: %s\n", t.AWS.Region)

	if len(networkingConfig.Subnets.Public) > 0 {
		fmt.Println("   ├── Public Subnets:")
		for _, s := range networkingConfig.Subnets.Public {
			fmt.Printf("   │   └── %s (%s)\n", s.CidrBlock, s.AvailabilityZone)
		}
	}

	if len(networkingConfig.Subnets.Private) > 0 {
		fmt.Println("   ├── Private Subnets:")
		for _, s := range networkingConfig.Subnets.Private {
			fmt.Printf("   │   └── %s (%s)\n", s.CidrBlock, s.AvailabilityZone)
		}
	}

	if networkingConfig.InternetGateway.Enabled {
		fmt.Println("   ├── Internet Gateway: Yes")
	}

	if networkingConfig.NATGateway.Enabled {
		fmt.Printf("   ├── NAT Gateway: %s\n", networkingConfig.NATGateway.Type)
	}

	fmt.Println("   └── Default Security Group: Yes")

	if dryRun {
		cyan.Println("\n   [DRY-RUN] Simulating resource creation...")
	} else {
		cyan.Println("\n   Creating resources (this may take a few minutes)...")
	}

	// Create networking
	opts := &provider.Options{
		DryRun: dryRun,
	}

	result, err := orchestrator.CreateTenantNetworking(ctx, t.ID, networkingConfig, opts)
	if err != nil {
		return nil, err
	}

	// Display created resources
	green.Println("\n   Resources Created:")
	fmt.Printf("   ├── VPC:             %s\n", result.VPCID)

	if result.InternetGatewayID != "" {
		fmt.Printf("   ├── Internet Gateway: %s\n", result.InternetGatewayID)
	}

	if len(result.PublicSubnetIDs) > 0 {
		fmt.Printf("   ├── Public Subnets:   %v\n", result.PublicSubnetIDs)
	}

	if len(result.PrivateSubnetIDs) > 0 {
		fmt.Printf("   ├── Private Subnets:  %v\n", result.PrivateSubnetIDs)
	}

	if len(result.NATGatewayIDs) > 0 {
		fmt.Printf("   ├── NAT Gateways:     %v\n", result.NATGatewayIDs)
	}

	fmt.Printf("   └── Security Group:   %s\n", result.DefaultSecurityGroupID)

	return result, nil
}

