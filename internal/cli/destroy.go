package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
	"github.com/yourusername/panka/pkg/provider/aws"
	"github.com/yourusername/panka/pkg/state"
	"github.com/yourusername/panka/pkg/tenant"
	"go.uber.org/zap"
)

var (
	destroyForce  bool
	destroyDryRun bool
	destroyAuto   bool
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy <path>",
	Short: "Destroy infrastructure resources",
	Long: `Destroy all resources defined in a stack.

Resources are destroyed in reverse dependency order to ensure clean teardown.
The command reads the current state from S3 and deletes resources tracked there.

⚠️  WARNING: This action is destructive and cannot be undone!

The destroy command will:
  • Load the current state from S3
  • Generate a destruction plan (reverse dependency order)
  • Prompt for confirmation (unless --auto-approve)
  • Destroy resources one by one
  • Update state as resources are deleted

Examples:
  panka destroy ./my-stack
  panka destroy ./my-stack --dry-run
  panka destroy ./my-stack --auto-approve
  panka destroy ./my-stack --force

Flags:
  --dry-run       Show what would be destroyed without doing it
  --force         Force destruction even if some resources fail
  --auto-approve  Skip confirmation prompt`,
	Args: cobra.ExactArgs(1),
	RunE: runDestroy,
}

func init() {
	rootCmd.AddCommand(destroyCmd)

	destroyCmd.Flags().BoolVar(&destroyForce, "force", false, "Force destruction even if some resources fail")
	destroyCmd.Flags().BoolVar(&destroyDryRun, "dry-run", false, "Show what would be destroyed")
	destroyCmd.Flags().BoolVar(&destroyAuto, "auto-approve", false, "Skip confirmation prompt")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)

	path := args[0]

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if path exists and is a directory
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("path not found: %s", absPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("destroy requires a stack folder, not a file: %s", absPath)
	}

	red.Println("\n🗑️  Panka Destroy")
	cyan.Println(strings.Repeat("─", 60))
	fmt.Printf("Stack Path: %s\n", absPath)

	if destroyDryRun {
		yellow.Println("\n⚠️  DRY-RUN MODE - No resources will be destroyed")
	}

	log := logger.Global()
	ctx := context.Background()

	// Step 1: Check authentication
	fmt.Print("\n⏳ Checking authentication... ")
	sessionMgr := tenant.NewSessionManager()
	session, err := sessionMgr.LoadSession()
	if err != nil || session.Mode != tenant.ModeTenant || session.Tenant == nil {
		red.Println("✗")
		return fmt.Errorf("not logged in as tenant. Run 'panka login' first")
	}
	green.Println("✓")
	fmt.Printf("   Tenant: %s\n", session.Tenant.ID)

	// Step 2: Get stack name from folder
	fmt.Print("⏳ Reading stack configuration... ")
	stackYAMLPath := filepath.Join(absPath, "stack.yaml")
	if _, err := os.Stat(stackYAMLPath); os.IsNotExist(err) {
		red.Println("✗")
		return fmt.Errorf("stack.yaml not found in %s", absPath)
	}

	// Read just the stack name from stack.yaml
	stackNameFromFolder := filepath.Base(absPath) // Fallback to folder name
	green.Println("✓")
	fmt.Printf("   Stack: %s\n", stackNameFromFolder)

	// Step 3: Load backend config
	bucket := viper.GetString("backend.bucket")
	region := viper.GetString("backend.region")

	if bucket == "" || region == "" {
		return fmt.Errorf("backend.bucket and backend.region must be configured in .panka.yaml")
	}

	// Step 4: Load current state from S3
	fmt.Print("⏳ Loading current state... ")

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	s3Client := s3.NewFromConfig(awsCfg)

	stateBackend, err := state.NewS3Backend(&state.S3BackendConfig{
		Client: s3Client,
		Bucket: bucket,
		Prefix: fmt.Sprintf("tenants/%s/v1/stacks", session.Tenant.ID),
	})
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("failed to create state backend: %w", err)
	}

	stackName := stackNameFromFolder
	environment := "default"

	stateKey := fmt.Sprintf("%s/%s/state.json", stackName, environment)
	currentState, err := stateBackend.Load(ctx, stateKey)
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("no state found for stack '%s'. Nothing to destroy", stackName)
	}
	green.Println("✓")

	resourceCount := currentState.ResourceCount()
	if resourceCount == 0 {
		yellow.Println("\n⚠️  No resources found in state. Nothing to destroy.")
		return nil
	}

	fmt.Printf("   Resources in state: %d\n", resourceCount)
	fmt.Printf("   Last deployed: %s\n", currentState.Metadata.UpdatedAt.Format(time.RFC3339))
	fmt.Printf("   Deployed by: %s\n", currentState.Metadata.DeployedBy)

	// Step 5: Generate destruction plan from state
	fmt.Print("⏳ Generating destruction plan... ")

	// Get resources from state and build a destruction order
	// We reverse the order based on dependencies stored in state
	resources := currentState.ListResources()
	
	// Build destruction plan (reverse dependency order)
	destructionPlan := buildDestructionPlan(resources)
	green.Println("✓")

	// Step 6: Display destruction plan
	displayDestructionPlan(destructionPlan, resourceCount)

	// Show warning
	red.Println("\n" + strings.Repeat("!", 60))
	red.Printf("   ⚠️  WARNING: %d resources will be PERMANENTLY DELETED!\n", resourceCount)
	red.Println("   This action CANNOT be undone!")
	red.Println(strings.Repeat("!", 60))

	// Get confirmation unless auto-approved or dry-run
	if !destroyDryRun && !destroyAuto {
		if !confirmDestruction(stackName) {
			yellow.Println("\n✋ Destruction cancelled by user")
			return nil
		}
	}

	// If dry-run, stop here
	if destroyDryRun {
		yellow.Println("\n🔍 DRY-RUN: No resources were actually destroyed")
		green.Println("✨ Dry-run complete!")
		return nil
	}

	// Step 7: Initialize AWS provider
	fmt.Print("\n⏳ Initializing AWS provider... ")
	awsProvider := aws.NewProvider()

	// Load tenant config for region
	tenantBackend, err := tenant.NewS3RegistryBackend(bucket, region)
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("failed to create tenant backend: %w", err)
	}

	tenantConfig, err := tenantBackend.LoadTenantConfig(ctx, session.Tenant.ID)
	providerRegion := region
	if err == nil && tenantConfig.AWS.Region != "" {
		providerRegion = tenantConfig.AWS.Region
	}

	err = awsProvider.Initialize(ctx, &provider.Config{
		Name:   "aws",
		Region: providerRegion,
		DefaultTags: map[string]string{
			"tenant":     session.Tenant.ID,
			"stack":      stackName,
			"managed-by": "panka",
		},
	})
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("failed to initialize AWS provider: %w", err)
	}
	defer awsProvider.Close()
	green.Println("✓")

	// Step 8: Execute destruction
	red.Println("\n🔥 Destroying Resources")
	cyan.Println(strings.Repeat("─", 60))

	startTime := time.Now()
	successCount := 0
	failCount := 0
	skippedCount := 0

	for _, stage := range destructionPlan {
		fmt.Printf("\n📦 Stage %d: %d resource(s)\n", stage.Number, len(stage.Resources))

		for _, res := range stage.Resources {
			fmt.Printf("   Deleting [%s] %s... ", res.Type, res.Name)

			// Get resource provider
			resourceProvider, err := awsProvider.GetResourceProvider(schema.Kind(res.Type))
			if err != nil {
				yellow.Printf("⚠️  Skipped (no provider)\n")
				log.Warn("No provider for resource kind",
					zap.String("kind", res.Type),
					zap.String("name", res.Name),
				)
				skippedCount++
				continue
			}

			// Delete resource
			opts := &provider.ResourceOptions{
				TenantID:  session.Tenant.ID,
				StackName: stackName,
				DryRun:    false,
			}

			result, err := resourceProvider.Delete(ctx, res.ID, opts)
			if err != nil {
				if destroyForce {
					yellow.Printf("⚠️  Failed (continuing due to --force)\n")
					log.Warn("Failed to delete resource, continuing",
						zap.String("name", res.Name),
						zap.Error(err),
					)
					failCount++
				} else {
					red.Println("✗")
					log.Error("Failed to delete resource",
						zap.String("name", res.Name),
						zap.Error(err),
					)
					failCount++

					// Save partial state before returning error
					currentState.Metadata.UpdatedAt = time.Now()
					if saveErr := stateBackend.Save(ctx, stateKey, currentState); saveErr != nil {
						yellow.Printf("⚠️  Warning: Failed to save state: %v\n", saveErr)
					}

					return fmt.Errorf("failed to delete %s: %w. Use --force to continue on errors", res.Name, err)
				}
				continue
			}

			green.Println("✓")
			successCount++

			// Remove from state
			currentState.RemoveResource(res.Name)

			log.Info("Resource deleted",
				zap.String("name", res.Name),
				zap.String("id", res.ID),
				zap.String("status", string(result.Status)),
			)
		}
	}

	// Step 9: Save final state
	fmt.Print("\n⏳ Saving state... ")
	currentState.Metadata.UpdatedAt = time.Now()
	currentState.Metadata.DeployedBy = "panka-cli"

	if currentState.ResourceCount() == 0 {
		// All resources deleted, remove state file
		if err := stateBackend.Delete(ctx, stateKey); err != nil {
			yellow.Printf("⚠️  Warning: Failed to delete state file: %v\n", err)
		} else {
			green.Println("✓ (state file deleted)")
		}
	} else {
		if err := stateBackend.Save(ctx, stateKey, currentState); err != nil {
			yellow.Printf("⚠️  Warning: Failed to save state: %v\n", err)
		} else {
			green.Println("✓")
		}
	}

	// Summary
	duration := time.Since(startTime)
	cyan.Println("\n" + strings.Repeat("─", 60))
	red.Println("🗑️  Destroy Summary")
	cyan.Println(strings.Repeat("─", 60))

	fmt.Printf("Stack:      %s\n", stackName)
	fmt.Printf("Duration:   %s\n", duration.Round(time.Second))
	green.Printf("Destroyed:  %d\n", successCount)
	if failCount > 0 {
		red.Printf("Failed:     %d\n", failCount)
	}
	if skippedCount > 0 {
		yellow.Printf("Skipped:    %d\n", skippedCount)
	}

	if failCount > 0 && !destroyForce {
		yellow.Println("\n⚠️  Some resources failed to delete")
		return fmt.Errorf("%d resource(s) failed to delete", failCount)
	}

	if successCount == resourceCount {
		green.Println("\n✨ All resources destroyed successfully!")
	} else if successCount > 0 {
		yellow.Println("\n⚠️  Partial destruction complete")
	}

	return nil
}

// DestructionStage represents a stage in the destruction plan
type DestructionStage struct {
	Number    int
	Resources []*state.Resource
}

// buildDestructionPlan creates a destruction plan from state resources
// Resources are destroyed in reverse order of their type priority
func buildDestructionPlan(resources []*state.Resource) []*DestructionStage {
	// Define destruction priority (higher number = destroy first)
	// This is the reverse of creation order
	typePriority := map[string]int{
		// Destroy compute resources first
		"AWS::Lambda::Function":   100,
		"AWS::ECS::Service":       100,
		"AWS::ECS::TaskDefinition": 95,
		"AWS::ECS::Cluster":       90,
		
		// Then messaging
		"AWS::SNS::Subscription":  85,
		"AWS::SNS::Topic":         80,
		"AWS::SQS::Queue":         80,
		
		// Then databases and storage
		"AWS::DynamoDB::Table":    70,
		"AWS::RDS::DBInstance":    70,
		"AWS::S3::Bucket":         60,
		
		// Then networking (last, as other resources depend on them)
		"AWS::EC2::SecurityGroup": 30,
		"AWS::EC2::NatGateway":    25,
		"AWS::EC2::RouteTable":    20,
		"AWS::EC2::Subnet":        15,
		"AWS::EC2::InternetGateway": 10,
		"AWS::EC2::VPC":           5,
	}

	// Group resources by priority
	priorityGroups := make(map[int][]*state.Resource)
	for _, res := range resources {
		priority, ok := typePriority[res.Type]
		if !ok {
			priority = 50 // Default priority
		}
		priorityGroups[priority] = append(priorityGroups[priority], res)
	}

	// Get sorted priorities (descending)
	var priorities []int
	for p := range priorityGroups {
		priorities = append(priorities, p)
	}
	// Sort descending
	for i := 0; i < len(priorities); i++ {
		for j := i + 1; j < len(priorities); j++ {
			if priorities[j] > priorities[i] {
				priorities[i], priorities[j] = priorities[j], priorities[i]
			}
		}
	}

	// Build stages
	var stages []*DestructionStage
	stageNum := 1
	for _, priority := range priorities {
		resources := priorityGroups[priority]
		if len(resources) > 0 {
			stages = append(stages, &DestructionStage{
				Number:    stageNum,
				Resources: resources,
			})
			stageNum++
		}
	}

	return stages
}

func displayDestructionPlan(stages []*DestructionStage, totalCount int) {
	cyan := color.New(color.FgCyan, color.Bold)
	red := color.New(color.FgRed)

	cyan.Println("\n🗑️  Destruction Plan (Reverse Dependency Order)")
	cyan.Println(strings.Repeat("─", 60))

	for _, stage := range stages {
		fmt.Printf("\nStage %d", stage.Number)
		if len(stage.Resources) > 1 {
			fmt.Printf(" (%d resources)", len(stage.Resources))
		} else {
			fmt.Printf(" (1 resource)")
		}
		fmt.Println()

		for _, res := range stage.Resources {
			red.Printf("  - Delete ")
			fmt.Printf("[%s] %s\n", res.Type, res.Name)
			if res.ID != "" {
				fmt.Printf("      ID: %s\n", res.ID)
			}
		}
	}

	fmt.Printf("\nPlan: %d resources to destroy in %d stages\n", totalCount, len(stages))
}

func confirmDestruction(stackName string) bool {
	reader := bufio.NewReader(os.Stdin)

	yellow := color.New(color.FgYellow, color.Bold)
	yellow.Printf("\nType the stack name '%s' to confirm destruction: ", stackName)

	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(input)
	return input == stackName
}
