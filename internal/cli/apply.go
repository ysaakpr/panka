package cli

import (
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
	"github.com/yourusername/panka/pkg/diff"
	"github.com/yourusername/panka/pkg/graph"
	"github.com/yourusername/panka/pkg/parser"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
	"github.com/yourusername/panka/pkg/provider/aws"
	"github.com/yourusername/panka/pkg/rollback"
	"github.com/yourusername/panka/pkg/state"
	"github.com/yourusername/panka/pkg/tenant"
	"go.uber.org/zap"
)

var (
	applyDryRun        bool
	applyAutoApprove   bool
	applyTarget        string
	applyNoRollback    bool
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply <path>",
	Short: "Apply infrastructure changes",
	Long: `Apply infrastructure changes defined in a stack folder.

This command:
  1. Parses the stack configuration
  2. Loads tenant networking (VPC, subnets, security groups)
  3. Builds the dependency graph
  4. Creates/updates resources in dependency order
  5. Saves state to S3

The stack will use the tenant's networking configuration automatically.

Examples:
  panka apply ./my-stack
  panka apply ./my-stack --dry-run
  panka apply ./my-stack --auto-approve
  panka apply ./my-stack --target api-server`,
	Args: cobra.ExactArgs(1),
	RunE: runApply,
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Preview changes without applying")
	applyCmd.Flags().BoolVarP(&applyAutoApprove, "auto-approve", "y", false, "Skip confirmation prompt")
	applyCmd.Flags().StringVar(&applyTarget, "target", "", "Target a specific resource")
	applyCmd.Flags().BoolVar(&applyNoRollback, "no-rollback", false, "Disable automatic rollback on failure")
}

func runApply(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	red := color.New(color.FgRed, color.Bold)
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
		return fmt.Errorf("apply requires a stack folder, not a file: %s", absPath)
	}

	cyan.Println("\n🚀 Panka Apply")
	cyan.Println(strings.Repeat("─", 60))
	fmt.Printf("Stack Path: %s\n", absPath)

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

	// Step 2: Parse stack folder
	fmt.Print("⏳ Parsing stack configuration... ")
	fp := parser.NewFolderParser()
	parseResult, err := fp.ParseStackFolder(absPath)
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("failed to parse stack: %w", err)
	}
	green.Println("✓")
	fmt.Printf("   Stack: %s\n", parseResult.Stack.Metadata.Name)
	fmt.Printf("   Services: %d\n", len(parseResult.Services))
	fmt.Printf("   Components: %d\n", len(parseResult.AllComponents))

	// Step 3: Load tenant configuration (for networking)
	fmt.Print("⏳ Loading tenant configuration... ")
	bucket := viper.GetString("backend.bucket")
	region := viper.GetString("backend.region")

	if bucket == "" || region == "" {
		red.Println("✗")
		return fmt.Errorf("backend.bucket and backend.region must be configured in .panka.yaml")
	}

	tenantBackend, err := tenant.NewS3RegistryBackend(bucket, region)
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("failed to create tenant backend: %w", err)
	}

	tenantConfig, err := tenantBackend.LoadTenantConfig(ctx, session.Tenant.ID)
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("failed to load tenant config: %w", err)
	}
	green.Println("✓")

	// Display networking info
	if tenantConfig.Networking.ResourceIDs != nil && tenantConfig.Networking.ResourceIDs.VPCID != "" {
		fmt.Printf("   VPC: %s\n", tenantConfig.Networking.ResourceIDs.VPCID)
		fmt.Printf("   Security Group: %s\n", tenantConfig.Networking.ResourceIDs.SecurityGroupID)
	} else {
		yellow.Println("   ⚠️  No networking provisioned for this tenant")
		yellow.Println("      Resources will be created without VPC configuration")
	}

	// Step 4: Validate configuration
	fmt.Print("⏳ Validating configuration... ")
	v := parser.NewValidator()
	validationResult := &parser.ParseResult{
		Stack:      parseResult.Stack,
		Components: parseResult.AllComponents,
	}
	for _, svc := range parseResult.Services {
		if svc.Service != nil {
			validationResult.Services = append(validationResult.Services, svc.Service)
		}
	}
	if err := v.Validate(validationResult); err != nil {
		red.Println("✗")
		return fmt.Errorf("validation failed: %w", err)
	}
	green.Println("✓")

	// Step 5: Build dependency graph
	fmt.Print("⏳ Building dependency graph... ")
	builder := graph.NewBuilder()
	depGraph, err := builder.Build(validationResult)
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("failed to build graph: %w", err)
	}

	if depGraph.HasCycle() {
		red.Println("✗")
		return fmt.Errorf("circular dependency detected")
	}
	green.Println("✓")
	fmt.Printf("   Nodes: %d, Edges: %d\n", depGraph.NodeCount(), depGraph.EdgeCount())

	// Step 6: Load current state for comparison
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

	stackName := parseResult.Stack.Metadata.Name
	environment := "default" // TODO: Get from stack or flag

	stateKey := fmt.Sprintf("%s/%s/state.json", stackName, environment)
	currentState, err := stateBackend.Load(ctx, stateKey)
	if err != nil {
		// State might not exist yet
		currentState = state.NewState(stackName, environment)
		currentState.Metadata.Tenant = session.Tenant.ID
	}
	green.Println("✓")

	// Step 7: Compute changes (state vs desired)
	fmt.Print("⏳ Computing changes... ")
	differ := diff.NewDiffer(nil)
	changeSet, err := differ.ComputeChangesFromFolderParse(parseResult, currentState)
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("failed to compute changes: %w", err)
	}
	changeSet.TenantID = session.Tenant.ID
	green.Println("✓")

	// Step 8: Generate deployment plan
	fmt.Print("⏳ Generating deployment plan... ")
	planner := graph.NewPlanner()
	plan, err := planner.CreateDeploymentPlan(depGraph, graph.ActionCreate)
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("failed to create plan: %w", err)
	}
	green.Println("✓")

	// Display changes
	diff.PrintDiff(changeSet)

	// Check if there are any changes
	if !changeSet.HasChanges() {
		green.Println("\n✨ No changes to apply. Infrastructure is up-to-date!")
		return nil
	}

	// If dry-run, stop here
	if applyDryRun {
		yellow.Println("\n⚠️  Dry-run mode - no changes will be made")
		return nil
	}

	// Confirmation
	if !applyAutoApprove {
		fmt.Print("\nDo you want to apply these changes? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "yes" {
			yellow.Println("Apply cancelled")
			return nil
		}
	}

	// Step 9: Initialize AWS provider
	fmt.Print("\n⏳ Initializing AWS provider... ")
	awsProvider := aws.NewProvider()
	providerRegion := tenantConfig.AWS.Region
	if providerRegion == "" {
		providerRegion = region
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

	// Step 10: Initialize rollback manager
	rollbackMgr := rollback.NewManager(awsProvider)
	rollbackMgr.StartTransaction(stackName, session.Tenant.ID, currentState)

	// Step 11: Apply changes
	cyan.Println("\n🔧 Applying Changes")
	cyan.Println(strings.Repeat("─", 60))

	startTime := time.Now()
	successCount := 0
	failCount := 0
	applyFailed := false

	for _, stage := range plan.Stages {
		fmt.Printf("\n📦 Stage %d: %d resource(s)\n", stage.Number, len(stage.Resources))

		for _, res := range stage.Resources {
			resourceName := res.ID
			resourceKind := res.Kind

			// Check if resource already exists in state
			existingResource, existsInState := currentState.GetResource(resourceName)

			// Get resource provider
			resourceProvider, err := awsProvider.GetResourceProvider(schema.Kind(resourceKind))
			if err != nil {
				yellow.Printf("   ⚠️  [%s] %s - Skipped (no provider)\n", resourceKind, resourceName)
				log.Warn("No provider for resource kind",
					zap.String("kind", string(resourceKind)),
					zap.String("name", resourceName),
				)
				continue
			}

			// Build options
			opts := &provider.ResourceOptions{
				TenantID:    session.Tenant.ID,
				StackName:   stackName,
				ServiceName: res.Resource.GetMetadata().Service,
				Tags: map[string]string{
					"stack":   stackName,
					"service": res.Resource.GetMetadata().Service,
				},
				DryRun: false,
			}

			// If resource exists in state, check if it exists in AWS and skip if unchanged
			if existsInState && existingResource.ID != "" {
				// Check if resource still exists in AWS
				exists, _ := resourceProvider.Exists(ctx, existingResource.ID, opts)
				if exists {
					green.Printf("   ✓ [%s] %s - Already exists\n", resourceKind, resourceName)
					log.Info("Resource already exists, skipping",
						zap.String("name", resourceName),
						zap.String("id", existingResource.ID),
					)
					successCount++
					continue
				}
				// Resource in state but not in AWS - recreate it
				yellow.Printf("   ~ [%s] %s - Recreating (missing in AWS)... ", resourceKind, resourceName)
			} else {
				fmt.Printf("   + [%s] %s - Creating... ", resourceKind, resourceName)
			}

			// Create resource
			result, err := resourceProvider.Create(ctx, res.Resource, opts)
			if err != nil {
				red.Println("✗")
				log.Error("Failed to create resource",
					zap.String("name", resourceName),
					zap.Error(err),
				)
				failCount++
				applyFailed = true

				// Record failed action for rollback tracking
				rollbackMgr.RecordCreate(resourceName, "", schema.Kind(resourceKind), nil, false, err)

				// Trigger rollback if enabled
				if !applyNoRollback && rollbackMgr.CanRollback() {
					yellow.Println("\n⚠️  Apply failed. Initiating rollback...")
					rollbackResult, rollbackErr := rollbackMgr.Rollback(ctx)
					if rollbackErr != nil {
						red.Printf("❌ Rollback error: %v\n", rollbackErr)
					} else {
						displayRollbackResult(rollbackResult)
					}
				}

				// Save partial state before returning
				currentState.Metadata.UpdatedAt = time.Now()
				_ = stateBackend.Save(ctx, stateKey, currentState)

				return fmt.Errorf("failed to create %s: %w", resourceName, err)
			}

			green.Println("✓")
			successCount++

			// Create state resource
			stateResource := &state.Resource{
				ID:         result.ResourceID,
				Type:       string(result.Kind),
				Name:       resourceName,
				Provider:   "aws",
				Status:     state.ResourceStatusReady,
				Attributes: convertOutputsToMap(result.Outputs),
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			// Record successful action for rollback
			rollbackMgr.RecordCreate(resourceName, result.ResourceID, schema.Kind(resourceKind), stateResource, true, nil)

			// Update state
			currentState.AddResource(resourceName, stateResource)

			// Log outputs
			if len(result.Outputs) > 0 {
				for k, v := range result.Outputs {
					fmt.Printf("      %s: %s\n", k, v)
				}
			}
		}
	}

	// Step 12: Delete resources that are in state but not in config
	// Build set of desired resource names
	desiredResources := make(map[string]bool)
	for _, comp := range parseResult.AllComponents {
		desiredResources[comp.GetMetadata().Name] = true
	}

	// Find resources to delete (in state but not in desired)
	resourcesToDelete := make([]*state.Resource, 0)
	for _, res := range currentState.ListResources() {
		if !desiredResources[res.Name] {
			resourcesToDelete = append(resourcesToDelete, res)
		}
	}

	// Delete removed resources
	deleteCount := 0
	deleteFailCount := 0
	if len(resourcesToDelete) > 0 {
		red.Printf("\n🗑️  Deleting %d removed resource(s)\n", len(resourcesToDelete))

		for _, res := range resourcesToDelete {
			fmt.Printf("   - [%s] %s... ", res.Type, res.Name)

			// Get resource provider
			resourceProvider, err := awsProvider.GetResourceProvider(schema.Kind(res.Type))
			if err != nil {
				yellow.Printf("⚠️  Skipped (no provider)\n")
				continue
			}

			// Delete resource
			_, err = resourceProvider.Delete(ctx, res.ID, &provider.ResourceOptions{
				TenantID:  session.Tenant.ID,
				StackName: stackName,
			})
			if err != nil {
				red.Println("✗")
				log.Error("Failed to delete resource",
					zap.String("name", res.Name),
					zap.Error(err),
				)
				deleteFailCount++
			} else {
				green.Println("✓")
				deleteCount++
				// Remove from state
				currentState.RemoveResource(res.Name)
			}
		}
	}

	// Clear rollback transaction on success
	if !applyFailed {
		rollbackMgr.ClearTransaction()
	}

	// Step 13: Save state
	fmt.Print("\n⏳ Saving state... ")
	currentState.Metadata.UpdatedAt = time.Now()
	currentState.Metadata.DeployedBy = "panka-cli"

	if err := stateBackend.Save(ctx, stateKey, currentState); err != nil {
		red.Println("✗")
		yellow.Printf("⚠️  Warning: Failed to save state: %v\n", err)
	} else {
		green.Println("✓")
	}

	// Summary
	duration := time.Since(startTime)
	cyan.Println("\n" + strings.Repeat("─", 60))
	cyan.Println("📊 Apply Summary")
	cyan.Println(strings.Repeat("─", 60))

	fmt.Printf("Stack:      %s\n", stackName)
	fmt.Printf("Duration:   %s\n", duration.Round(time.Second))
	green.Printf("Created/Updated: %d\n", successCount)
	if deleteCount > 0 {
		red.Printf("Deleted:    %d\n", deleteCount)
	}
	if failCount > 0 {
		red.Printf("Failed:     %d\n", failCount)
	}
	if deleteFailCount > 0 {
		red.Printf("Delete Failed: %d\n", deleteFailCount)
	}

	totalFailed := failCount + deleteFailCount
	if totalFailed > 0 {
		yellow.Println("\n⚠️  Some operations failed")
		return fmt.Errorf("%d operation(s) failed", totalFailed)
	}

	green.Println("\n✨ Apply complete!")
	return nil
}

// displayRollbackResult displays the result of a rollback operation
func displayRollbackResult(result *rollback.RollbackResult) {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n🔄 Rollback Summary")
	cyan.Println(strings.Repeat("─", 40))

	fmt.Printf("Duration:  %s\n", result.Duration.Round(time.Millisecond))
	if result.SuccessCount > 0 {
		green.Printf("Reversed:  %d\n", result.SuccessCount)
	}
	if result.FailedCount > 0 {
		red.Printf("Failed:    %d\n", result.FailedCount)
	}
	if result.SkippedCount > 0 {
		yellow.Printf("Skipped:   %d\n", result.SkippedCount)
	}

	if len(result.Errors) > 0 {
		red.Println("\nRollback Errors:")
		for _, err := range result.Errors {
			red.Printf("  - %s (%s): %s\n", err.ResourceName, err.Action, err.Error)
		}
	}

	if result.Success {
		green.Println("\n✓ Rollback completed successfully")
	} else {
		yellow.Println("\n⚠️  Rollback completed with errors - manual cleanup may be required")
	}
}

// displayApplyPlan displays the deployment plan
func displayApplyPlan(plan *graph.DeploymentPlan, result *parser.StackParseResult) {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)

	cyan.Println("\n📋 Deployment Plan")
	cyan.Println(strings.Repeat("─", 60))

	for _, stage := range plan.Stages {
		fmt.Printf("\nStage %d", stage.Number)
		if len(stage.Resources) > 1 {
			fmt.Printf(" (parallel - %d resources)", len(stage.Resources))
		}
		fmt.Println()

		for _, res := range stage.Resources {
			green.Printf("  + Create ")
			fmt.Printf("[%s] %s\n", res.Kind, res.ID)

			// Show service info
			if res.Resource != nil {
				service := res.Resource.GetMetadata().Service
				if service != "" {
					fmt.Printf("      Service: %s\n", service)
				}
			}
		}
	}

	// Summary
	fmt.Printf("\nPlan: %d resources to create in %d stages\n", plan.TotalResources, plan.TotalStages)
}

// convertOutputsToMap converts string outputs to interface{} map
func convertOutputsToMap(outputs map[string]string) map[string]interface{} {
	if outputs == nil {
		return nil
	}
	result := make(map[string]interface{}, len(outputs))
	for k, v := range outputs {
		result[k] = v
	}
	return result
}

