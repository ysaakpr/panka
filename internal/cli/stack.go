package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/panka/pkg/state"
	"github.com/yourusername/panka/pkg/tenant"
	"go.uber.org/zap"
)

var (
	stackEnvironment string
	stackListAll     bool
)

// stackCmd represents the stack command
var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Manage infrastructure stacks",
	Long: `Manage infrastructure stacks within your tenant.

Stack commands allow you to:
  • List all stacks in your tenant
  • View detailed stack information
  • Create new stacks
  • Manage stack lifecycle`,
}

// stackListCmd lists all stacks
var stackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stacks",
	Long: `List all infrastructure stacks in your tenant.

This command shows:
  • Stack name
  • Environment
  • Number of resources
  • Last deployment time
  • Status

Examples:
  # List all stacks
  panka stack list

  # List stacks for specific environment
  panka stack list --environment production`,
	RunE: runStackList,
}

// stackInfoCmd shows detailed stack information
var stackInfoCmd = &cobra.Command{
	Use:   "info <stack-name> [environment]",
	Short: "Show detailed stack information",
	Long: `Show detailed information about a specific stack.

This command displays:
  • Stack metadata
  • All deployed resources
  • Resource dependencies
  • Deployment history
  • Outputs

Examples:
  # Show stack info for production
  panka stack info my-stack production

  # Show stack info (defaults to production)
  panka stack info my-stack`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runStackInfo,
}

// stackCreateCmd creates a new stack
var stackCreateCmd = &cobra.Command{
	Use:   "create <stack-name> <environment>",
	Short: "Create a new stack",
	Long: `Create a new infrastructure stack with initial configuration.

This command:
  • Validates stack name
  • Creates stack metadata
  • Initializes empty state
  • Sets up tracking

Examples:
  # Create production stack
  panka stack create my-app production

  # Create staging stack
  panka stack create my-app staging`,
	Args: cobra.ExactArgs(2),
	RunE: runStackCreate,
}

func init() {
	rootCmd.AddCommand(stackCmd)
	stackCmd.AddCommand(stackListCmd)
	stackCmd.AddCommand(stackInfoCmd)
	stackCmd.AddCommand(stackCreateCmd)

	// Flags
	stackListCmd.Flags().StringVar(&stackEnvironment, "environment", "", "Filter by environment")
	stackListCmd.Flags().BoolVar(&stackListAll, "all", false, "Show all environments")
}

// getBackendConfig reads backend configuration from viper (which reads .panka.yaml)
func getBackendConfig() (bucket, region, prefix string, err error) {
	bucket = viper.GetString("backend.bucket")
	region = viper.GetString("backend.region")
	prefix = viper.GetString("backend.prefix")

	if bucket == "" {
		return "", "", "", fmt.Errorf("backend.bucket is required in .panka.yaml")
	}
	if region == "" {
		region = "us-east-1" // Default region
	}

	return bucket, region, prefix, nil
}

func runStackList(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)

	// Get backend config from .panka.yaml via Viper
	bucket, region, prefix, err := getBackendConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load tenant context
	tenantCtx, err := tenant.LoadTenantContext()
	if err != nil {
		yellow.Println("⚠️  Not logged in - showing non-tenant stacks")
		tenantCtx = &tenant.TenantContext{Enabled: false}
	}

	ctx := tenant.WithTenant(context.Background(), tenantCtx)

	// Create state backend
	backend, err := createStackBackend(bucket, region, prefix)
	if err != nil {
		return fmt.Errorf("failed to create state backend: %w", err)
	}

	// Wrap with tenant-aware backend
	tenantBackend := state.NewTenantAwareBackend(backend)

	cyan.Println("\n📦 Infrastructure Stacks")
	cyan.Println(strings.Repeat("─", 80))

	if tenantCtx.Enabled {
		fmt.Printf("\nTenant: %s\n", tenantCtx.TenantID)
	}

	fmt.Printf("S3 Bucket: %s\n", bucket)

	// List all state files
	searchPrefix := "stacks/"
	keys, err := tenantBackend.List(ctx, searchPrefix)
	if err != nil {
		return fmt.Errorf("failed to list stacks: %w", err)
	}

	if len(keys) == 0 {
		yellow.Println("\nNo stacks found")
		fmt.Println("\nCreate a stack:")
		fmt.Println("  panka stack create <name> <environment>")
		return nil
	}

	// Parse stack information from keys
	stacks := parseStackKeys(keys)

	// Filter by environment if specified
	if stackEnvironment != "" {
		stacks = filterByEnvironment(stacks, stackEnvironment)
	}

	if len(stacks) == 0 {
		yellow.Printf("\nNo stacks found for environment: %s\n", stackEnvironment)
		return nil
	}

	// Display stacks
	fmt.Println()
	fmt.Printf("%-30s %-15s %-10s %-20s %s\n", "STACK", "ENVIRONMENT", "RESOURCES", "LAST UPDATED", "STATUS")
	cyan.Println(strings.Repeat("─", 80))

	for _, stack := range stacks {
		// Load state for each stack
		stateKey := fmt.Sprintf("stacks/%s/%s/state.json", stack.Name, stack.Environment)
		stackState, err := tenantBackend.Load(ctx, stateKey)

		if err != nil {
			// Stack exists but no state yet
			fmt.Printf("%-30s %-15s %-10s %-20s %s\n",
				stack.Name,
				stack.Environment,
				"0",
				"-",
				yellow.Sprint("empty"))
			continue
		}

		// Display stack info
		resourceCount := len(stackState.Resources)
		lastUpdate := stackState.LastUpdate.Format("2006-01-02 15:04")
		status := green.Sprint("✓ deployed")

		fmt.Printf("%-30s %-15s %-10d %-20s %s\n",
			stack.Name,
			stack.Environment,
			resourceCount,
			lastUpdate,
			status)
	}

	cyan.Println(strings.Repeat("─", 80))
	fmt.Printf("\nTotal: %d stacks\n", len(stacks))

	return nil
}

func runStackInfo(cmd *cobra.Command, args []string) error {
	stackName := args[0]
	environment := "production"
	if len(args) > 1 {
		environment = args[1]
	}

	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)

	// Get backend config
	bucket, region, prefix, err := getBackendConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load tenant context
	tenantCtx, err := tenant.LoadTenantContext()
	if err != nil {
		yellow.Println("⚠️  Not logged in - accessing non-tenant stacks")
		tenantCtx = &tenant.TenantContext{Enabled: false}
	}

	ctx := tenant.WithTenant(context.Background(), tenantCtx)

	// Create state backend
	backend, err := createStackBackend(bucket, region, prefix)
	if err != nil {
		return fmt.Errorf("failed to create state backend: %w", err)
	}

	tenantBackend := state.NewTenantAwareBackend(backend)

	// Load stack state
	stateKey := fmt.Sprintf("stacks/%s/%s/state.json", stackName, environment)
	stackState, err := tenantBackend.Load(ctx, stateKey)
	if err != nil {
		return fmt.Errorf("stack not found: %s/%s", stackName, environment)
	}

	// Display stack information
	cyan.Printf("\n📦 Stack: %s\n", stackName)
	cyan.Println(strings.Repeat("─", 80))

	fmt.Println("\nMetadata:")
	fmt.Printf("  Environment:   %s\n", stackState.Metadata.Environment)
	fmt.Printf("  Version:       %s\n", stackState.Version)
	if stackState.Metadata.Tenant != "" {
		fmt.Printf("  Tenant:        %s\n", stackState.Metadata.Tenant)
	}
	fmt.Printf("  Created:       %s\n", stackState.Metadata.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Last Updated:  %s\n", stackState.Metadata.UpdatedAt.Format("2006-01-02 15:04:05"))
	if stackState.Metadata.DeployedBy != "" {
		fmt.Printf("  Deployed By:   %s\n", stackState.Metadata.DeployedBy)
	}

	// Display resources
	fmt.Printf("\nResources: %d\n", len(stackState.Resources))

	if len(stackState.Resources) > 0 {
		fmt.Println()
		fmt.Printf("%-30s %-25s %-15s\n", "ID", "TYPE", "STATUS")
		cyan.Println(strings.Repeat("─", 80))

		for id, resource := range stackState.Resources {
			statusColor := green
			statusSymbol := "✓"

			switch resource.Status {
			case state.ResourceStatusReady:
				statusColor = green
				statusSymbol = "✓"
			case state.ResourceStatusCreating:
				statusColor = yellow
				statusSymbol = "⟳"
			case state.ResourceStatusFailed:
				statusColor = color.New(color.FgRed)
				statusSymbol = "✗"
			}

			fmt.Printf("%-30s %-25s %s\n",
				id,
				resource.Type,
				statusColor.Sprintf("%s %s", statusSymbol, resource.Status))
		}
	}

	// Display outputs
	if len(stackState.Outputs) > 0 {
		fmt.Println("\nOutputs:")
		for key, value := range stackState.Outputs {
			fmt.Printf("  %s = %v\n", key, value)
		}
	}

	cyan.Println(strings.Repeat("─", 80))

	return nil
}

func runStackCreate(cmd *cobra.Command, args []string) error {
	stackName := args[0]
	environment := args[1]

	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)

	// Validate stack name
	if err := validateStackName(stackName); err != nil {
		return err
	}

	// Validate environment
	if err := validateEnvironment(environment); err != nil {
		return err
	}

	// Get backend config
	bucket, region, prefix, err := getBackendConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load tenant context
	tenantCtx, err := tenant.LoadTenantContext()
	if err != nil {
		yellow.Println("⚠️  Not logged in - creating in non-tenant mode")
		tenantCtx = &tenant.TenantContext{Enabled: false}
	}

	ctx := tenant.WithTenant(context.Background(), tenantCtx)

	// Create state backend
	backend, err := createStackBackend(bucket, region, prefix)
	if err != nil {
		return fmt.Errorf("failed to create state backend: %w", err)
	}

	tenantBackend := state.NewTenantAwareBackend(backend)

	cyan.Printf("\n📦 Creating Stack: %s\n", stackName)
	cyan.Println(strings.Repeat("─", 50))

	fmt.Printf("\nEnvironment:  %s\n", environment)
	fmt.Printf("S3 Bucket:    %s\n", bucket)
	if tenantCtx.Enabled {
		fmt.Printf("Tenant:       %s\n", tenantCtx.TenantID)
	}

	// Check if stack already exists
	stateKey := fmt.Sprintf("stacks/%s/%s/state.json", stackName, environment)
	_, err = tenantBackend.Load(ctx, stateKey)
	if err == nil {
		red.Printf("\n✗ Stack already exists: %s/%s\n", stackName, environment)
		return fmt.Errorf("stack already exists")
	}

	// Create initial state
	newState := state.NewState(stackName, environment)
	newState.Metadata.DeployedBy = os.Getenv("USER")
	if tenantCtx.Enabled {
		newState.Metadata.Tenant = tenantCtx.TenantID
	}

	// Save initial state
	fmt.Println("\nCreating stack...")
	fmt.Println("├── Initializing state...")

	if err := tenantBackend.Save(ctx, stateKey, newState); err != nil {
		red.Printf("✗ Failed to create stack: %v\n", err)
		return fmt.Errorf("failed to save initial state: %w", err)
	}

	fmt.Println("└── Stack created ✓")

	green.Println("\n✓ Stack created successfully!")

	cyan.Println("\nNext steps:")
	fmt.Printf("  1. Create infrastructure.yaml defining your resources\n")
	fmt.Printf("  2. Validate: panka validate infrastructure.yaml\n")
	fmt.Printf("  3. Plan:     panka plan infrastructure.yaml\n")
	fmt.Printf("  4. Apply:    panka apply infrastructure.yaml\n")

	cyan.Println(strings.Repeat("─", 50))

	return nil
}

// Helper functions

type StackInfo struct {
	Name        string
	Environment string
}

func parseStackKeys(keys []string) []StackInfo {
	stackMap := make(map[string]bool)
	var stacks []StackInfo

	for _, key := range keys {
		// Parse: stacks/<name>/<env>/state.json
		parts := strings.Split(key, "/")
		if len(parts) >= 3 && parts[0] == "stacks" {
			stackKey := parts[1] + "/" + parts[2]
			if !stackMap[stackKey] {
				stackMap[stackKey] = true
				stacks = append(stacks, StackInfo{
					Name:        parts[1],
					Environment: parts[2],
				})
			}
		}
	}

	return stacks
}

func filterByEnvironment(stacks []StackInfo, env string) []StackInfo {
	var filtered []StackInfo
	for _, stack := range stacks {
		if stack.Environment == env {
			filtered = append(filtered, stack)
		}
	}
	return filtered
}

func validateStackName(name string) error {
	if name == "" {
		return fmt.Errorf("stack name cannot be empty")
	}
	if len(name) < 3 {
		return fmt.Errorf("stack name must be at least 3 characters")
	}
	if len(name) > 63 {
		return fmt.Errorf("stack name must be at most 63 characters")
	}
	// Must be lowercase alphanumeric with hyphens
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return fmt.Errorf("stack name must be lowercase alphanumeric with hyphens only")
		}
	}
	return nil
}

func validateEnvironment(env string) error {
	validEnvs := map[string]bool{
		"production":  true,
		"staging":     true,
		"development": true,
		"dev":         true,
		"prod":        true,
		"test":        true,
		"qa":          true,
	}

	if !validEnvs[env] {
		return fmt.Errorf("invalid environment: %s (valid: production, staging, development, test, qa)", env)
	}
	return nil
}

// createStackBackend creates an S3 backend using the provided configuration
func createStackBackend(bucket, region, prefix string) (state.Backend, error) {
	// Create zap logger
	zapLog, _ := zap.NewProduction()

	// Create AWS config
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsCfg)

	// Create S3 backend
	backend, err := state.NewS3Backend(&state.S3BackendConfig{
		Client: s3Client,
		Bucket: bucket,
		Prefix: prefix,
		Logger: zapLog,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 backend: %w", err)
	}

	return backend, nil
}

