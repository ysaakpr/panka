package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/diff"
	"github.com/yourusername/panka/pkg/provider"
	"github.com/yourusername/panka/pkg/provider/aws"
	"github.com/yourusername/panka/pkg/state"
	"github.com/yourusername/panka/pkg/tenant"
	"go.uber.org/zap"
)

var (
	driftOutput string
)

// driftCmd represents the drift command
var driftCmd = &cobra.Command{
	Use:   "drift <path>",
	Short: "Detect configuration drift",
	Long: `Detect configuration drift by comparing stored state with actual AWS resources.

This command checks if any resources have been modified or deleted outside of Panka.
It compares the state stored in S3 with the actual resources in your AWS account.

Drift can occur when:
  • Resources are modified manually in the AWS console
  • Changes are made via the AWS CLI or SDK
  • Another tool modifies the same resources
  • Resources are deleted outside of Panka

Examples:
  panka drift ./my-stack
  panka drift ./my-stack --output json
  panka drift ./my-stack --output table`,
	Args: cobra.ExactArgs(1),
	RunE: runDrift,
}

func init() {
	rootCmd.AddCommand(driftCmd)

	driftCmd.Flags().StringVarP(&driftOutput, "output", "o", "table", "Output format: table, json")
}

func runDrift(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)
	magenta := color.New(color.FgMagenta)

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
		return fmt.Errorf("drift requires a stack folder, not a file: %s", absPath)
	}

	cyan.Println("\n🔍 Panka Drift Detection")
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

	// Step 2: Get stack name from folder
	fmt.Print("⏳ Reading stack configuration... ")
	stackYAMLPath := filepath.Join(absPath, "stack.yaml")
	if _, err := os.Stat(stackYAMLPath); os.IsNotExist(err) {
		red.Println("✗")
		return fmt.Errorf("stack.yaml not found in %s", absPath)
	}

	stackName := filepath.Base(absPath)
	green.Println("✓")
	fmt.Printf("   Stack: %s\n", stackName)

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

	environment := "default"
	stateKey := fmt.Sprintf("%s/%s/state.json", stackName, environment)
	currentState, err := stateBackend.Load(ctx, stateKey)
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("no state found for stack '%s'. Nothing to check for drift", stackName)
	}
	green.Println("✓")

	resourceCount := currentState.ResourceCount()
	if resourceCount == 0 {
		yellow.Println("\n⚠️  No resources found in state. Nothing to check for drift.")
		return nil
	}

	fmt.Printf("   Resources in state: %d\n", resourceCount)

	// Step 5: Initialize AWS provider
	fmt.Print("⏳ Initializing AWS provider... ")
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
	})
	if err != nil {
		red.Println("✗")
		return fmt.Errorf("failed to initialize AWS provider: %w", err)
	}
	defer awsProvider.Close()
	green.Println("✓")

	// Step 6: Run drift detection
	cyan.Println("\n🔎 Checking for Drift")
	cyan.Println(strings.Repeat("─", 60))

	detector := diff.NewDriftDetector(awsProvider, nil)
	report, err := detector.DetectDrift(ctx, currentState)
	if err != nil {
		return fmt.Errorf("drift detection failed: %w", err)
	}

	// Display results
	displayDriftReport(report, log)

	// Summary
	cyan.Println("\n" + strings.Repeat("─", 60))
	cyan.Println("📊 Drift Detection Summary")
	cyan.Println(strings.Repeat("─", 60))

	fmt.Printf("Stack:      %s\n", stackName)
	fmt.Printf("Duration:   %s\n", report.Duration.Round(1000000))
	fmt.Printf("Resources:  %d\n", report.Summary.Total)

	if report.Summary.Clean > 0 {
		green.Printf("  Clean:    %d\n", report.Summary.Clean)
	}
	if report.Summary.Modified > 0 {
		yellow.Printf("  Modified: %d\n", report.Summary.Modified)
	}
	if report.Summary.Deleted > 0 {
		red.Printf("  Deleted:  %d\n", report.Summary.Deleted)
	}
	if report.Summary.Unknown > 0 {
		magenta.Printf("  Unknown:  %d\n", report.Summary.Unknown)
	}

	// Final status
	if report.HasDrift() {
		yellow.Println("\n⚠️  Drift detected! Run 'panka apply' to reconcile.")
		return nil
	}

	green.Println("\n✨ No drift detected. Infrastructure is in sync!")
	return nil
}

func displayDriftReport(report *diff.DriftReport, log *logger.Logger) {
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)
	magenta := color.New(color.FgMagenta)
	dim := color.New(color.Faint)

	for _, result := range report.Results {
		switch result.Type {
		case diff.DriftNone:
			green.Printf("  ✓ [%s] %s\n", result.ResourceKind, result.ResourceName)
			dim.Printf("      ID: %s\n", result.ResourceID)

		case diff.DriftModified:
			yellow.Printf("  ~ [%s] %s (modified)\n", result.ResourceKind, result.ResourceName)
			fmt.Printf("      ID: %s\n", result.ResourceID)

			// Show diffs
			if len(result.Diffs) > 0 {
				fmt.Println("      Changes detected:")
				for _, d := range result.Diffs {
					if d.Sensitive {
						dim.Printf("        %s: (sensitive value changed)\n", d.Attribute)
					} else {
						red.Printf("        - %s: %v\n", d.Attribute, d.StoredValue)
						green.Printf("        + %s: %v\n", d.Attribute, d.ActualValue)
					}
				}
			}

			log.Info("Drift detected",
				zap.String("resource", result.ResourceName),
				zap.String("type", string(result.Type)),
			)

		case diff.DriftDeleted:
			red.Printf("  ✗ [%s] %s (DELETED from AWS)\n", result.ResourceKind, result.ResourceName)
			dim.Printf("      ID: %s\n", result.ResourceID)
			dim.Printf("      Resource exists in Panka state but not in AWS\n")

			log.Warn("Resource deleted outside Panka",
				zap.String("resource", result.ResourceName),
				zap.String("id", result.ResourceID),
			)

		case diff.DriftUnknown:
			magenta.Printf("  ? [%s] %s (unknown)\n", result.ResourceKind, result.ResourceName)
			dim.Printf("      ID: %s\n", result.ResourceID)
			if result.Error != "" {
				dim.Printf("      Error: %s\n", result.Error)
			}

			log.Warn("Could not determine drift status",
				zap.String("resource", result.ResourceName),
				zap.String("error", result.Error),
			)
		}
	}
}

