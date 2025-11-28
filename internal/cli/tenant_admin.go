package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yourusername/panka/pkg/tenant"
)

var (
	tenantName         string
	tenantDisplayName  string
	tenantEmail        string
	tenantAWSAccount   string
	tenantVersion      string
	tenantCostTracking bool
	tenantCostLimit    int
	tenantMaxStacks    int
	tenantMaxServices  int
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
	Use:   "init",
	Short: "Create a new tenant",
	Long:  `Create a new tenant with isolated state and credentials.`,
	RunE:  runTenantInit,
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
	
	// Init flags
	tenantInitCmd.Flags().StringVar(&tenantName, "name", "", "Tenant name (lowercase, alphanumeric, hyphens)")
	tenantInitCmd.Flags().StringVar(&tenantDisplayName, "display-name", "", "Display name")
	tenantInitCmd.Flags().StringVar(&tenantEmail, "email", "", "Contact email")
	tenantInitCmd.Flags().StringVar(&tenantAWSAccount, "aws-account", "", "AWS account ID")
	tenantInitCmd.Flags().StringVar(&tenantVersion, "version", "v1", "State version")
	tenantInitCmd.Flags().BoolVar(&tenantCostTracking, "cost-tracking", true, "Enable cost tracking")
	tenantInitCmd.Flags().IntVar(&tenantCostLimit, "cost-limit", 5000, "Monthly cost limit (USD, 0 for unlimited)")
	tenantInitCmd.Flags().IntVar(&tenantMaxStacks, "max-stacks", 100, "Maximum number of stacks")
	tenantInitCmd.Flags().IntVar(&tenantMaxServices, "max-services", 500, "Maximum number of services")
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

	cyan.Println("\n🏢 Create New Tenant")
	cyan.Println(strings.Repeat("─", 50))

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
		Version:          tenantVersion,
		CostTracking:     tenantCostTracking,
		MonthlyCostLimit: tenantCostLimit,
		MaxStacks:        tenantMaxStacks,
		MaxServices:      tenantMaxServices,
		Metadata:         make(map[string]string),
	}

	// Create tenant
	fmt.Println("\nCreating tenant...")
	fmt.Println("├── Validating tenant name...")
	fmt.Println("├── Checking for conflicts...")
	fmt.Println("├── Generating tenant ID...")
	fmt.Println("├── Generating secure credentials...")
	
	newTenant, creds, err := manager.CreateTenant(ctx, req)
	if err != nil {
		red.Printf("✗ Failed to create tenant: %v\n", err)
		return err
	}
	
	fmt.Println("├── Creating S3 directory structure...")
	if err := backend.CreateTenantDirectory(ctx, newTenant); err != nil {
		yellow.Printf("⚠️  Warning: Failed to create tenant directory: %v\n", err)
	}
	
	fmt.Println("└── Tenant created successfully ✓")

	// Display credentials
	cyan.Println("\n" + strings.Repeat("─", 50))
	green.Println("✓ Tenant Created")
	
	fmt.Println("\nTenant ID:    ", creds.TenantID)
	yellow.Println("Tenant Secret:", creds.Secret)
	yellow.Println("               ", strings.Repeat("^", len(creds.Secret)))
	yellow.Println("               SAVE THIS - IT CANNOT BE RECOVERED")
	
	fmt.Printf("\nS3 Path:      s3://%s/%s\n", session.Backend.Bucket, newTenant.Storage.Path)
	fmt.Printf("Lock Prefix:  %s\n", newTenant.Locks.Prefix)
	
	cyan.Println("\nShare with team:")
	fmt.Printf("  Tenant: %s\n", creds.TenantID)
	fmt.Printf("  Secret: %s\n", creds.Secret)
	fmt.Printf("  Bucket: %s\n", session.Backend.Bucket)
	fmt.Printf("  Region: %s\n", session.Backend.Region)
	
	cyan.Println(strings.Repeat("─", 50))
	
	yellow.Println("\n⚠️  IMPORTANT: Store the tenant secret securely.")
	yellow.Println("   It cannot be retrieved later. If lost, use 'panka tenant rotate'.")
	
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
	fmt.Printf("  Cost Tracking: %v\n", t.Limits.CostTracking)
	if t.Limits.MonthlyCostLimit > 0 {
		fmt.Printf("  Cost Limit:    $%d/month\n", t.Limits.MonthlyCostLimit)
	} else {
		fmt.Println("  Cost Limit:    Unlimited")
	}
	fmt.Printf("  Max Stacks:    %d\n", t.Limits.MaxStacks)
	fmt.Printf("  Max Services:  %d\n", t.Limits.MaxServices)

	if t.AWS.AccountID != "" {
		cyan.Println("\nAWS:")
		fmt.Printf("  Account ID:  %s\n", t.AWS.AccountID)
		fmt.Printf("  Region:      %s\n", t.AWS.Region)
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

