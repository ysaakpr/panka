package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yourusername/panka/pkg/parser"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate <path>",
	Short: "Validate infrastructure configuration",
	Long: `Validate a stack folder or single configuration file.

For stack folders (new structure):
  panka validate ./my-stack

  Expected folder structure:
    my-stack/
    ├── stack.yaml
    └── services/
        ├── api/
        │   ├── service.yaml
        │   └── *.yaml
        └── worker/
            └── *.yaml

For single files (legacy):
  panka validate infrastructure.yaml

Validation checks:
  • YAML syntax
  • Schema compliance
  • Resource references
  • Circular dependencies
  • Required fields`,
	Args: cobra.ExactArgs(1),
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("path not found: %s", absPath)
	}

	// Determine if it's a folder or file
	if info.IsDir() {
		return validateStackFolder(absPath)
	}
	return validateSingleFile(absPath)
}

// validateStackFolder validates a stack folder structure
func validateStackFolder(stackPath string) error {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n📦 Validating Stack Folder")
	cyan.Println(strings.Repeat("─", 60))
	fmt.Printf("Path: %s\n\n", stackPath)

	// Parse stack folder
	fp := parser.NewFolderParser()
	result, err := fp.ParseStackFolder(stackPath)
	if err != nil {
		red.Printf("❌ Parse Error: %v\n", err)
		return err
	}

	// Display stack info
	green.Printf("✓ Stack: %s\n", result.Stack.Metadata.Name)

	if result.Stack.Metadata.Tenant != "" {
		fmt.Printf("  Tenant: %s\n", result.Stack.Metadata.Tenant)
	}

	// Display networking info if available
	if result.TenantNetworking != nil && result.TenantNetworking.VPC.CidrBlock != "" {
		fmt.Printf("  VPC: %s (from tenant)\n", result.TenantNetworking.VPC.CidrBlock)
	}

	// Display services
	cyan.Printf("\n📋 Services: %d\n", len(result.Services))
	for serviceName, svc := range result.Services {
		fmt.Printf("  ├── %s\n", serviceName)
		fmt.Printf("  │   Components: %d\n", len(svc.Components))
		for _, comp := range svc.Components {
			fmt.Printf("  │     • %s (%s)\n", comp.GetMetadata().Name, comp.GetKind())
		}
	}

	// Display total components
	cyan.Printf("\n📦 Total Components: %d\n", len(result.AllComponents))

	// Resource type summary
	resourceCounts := make(map[string]int)
	for _, comp := range result.AllComponents {
		resourceCounts[string(comp.GetKind())]++
	}

	if len(resourceCounts) > 0 {
		fmt.Println("  By Type:")
		for kind, count := range resourceCounts {
			fmt.Printf("    • %s: %d\n", kind, count)
		}
	}

	// Validate with standard validator
	v := parser.NewValidator()
	parseResult := &parser.ParseResult{
		Stack:      result.Stack,
		Components: result.AllComponents,
	}
	for _, svc := range result.Services {
		if svc.Service != nil {
			parseResult.Services = append(parseResult.Services, svc.Service)
		}
	}

	if err := v.Validate(parseResult); err != nil {
		red.Printf("\n❌ Validation Errors:\n")
		fmt.Printf("   %v\n", err)
		return err
	}

	// Display warnings
	if len(result.Warnings) > 0 {
		yellow.Println("\n⚠️  Warnings:")
		for _, w := range result.Warnings {
			fmt.Printf("   • %s\n", w)
		}
	}

	// Success
	cyan.Println("\n" + strings.Repeat("─", 60))
	green.Println("✓ Stack validation successful!")

	fmt.Println("\nNext steps:")
	fmt.Printf("  • Generate plan: panka plan %s\n", stackPath)
	fmt.Printf("  • Visualize:     panka graph %s\n", stackPath)

	return nil
}

// validateSingleFile validates a single YAML file (legacy mode)
func validateSingleFile(filePath string) error {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n🔍 Validating single file (legacy mode)...")
	fmt.Printf("File: %s\n\n", filePath)

	p := parser.NewParser()
	v := parser.NewValidator()

	// Parse file
	result, err := p.ParseFile(filePath)
	if err != nil {
		red.Printf("❌ Parse error: %v\n", err)
		return err
	}

	fmt.Printf("ℹ️  Found %d resources\n", len(result.Components))

	// Validate resources
	if err := v.Validate(result); err != nil {
		red.Printf("❌ Validation failed:\n")
		fmt.Printf("   %v\n", err)
		return err
	}

	green.Printf("✅ Valid\n")

	// Show summary
	if GetVerbose() {
		resourceCounts := make(map[string]int)
		for _, res := range result.Components {
			resourceCounts[string(res.GetKind())]++
		}

		yellow.Println("\nResource Summary:")
		for kind, count := range resourceCounts {
			fmt.Printf("  • %s: %d\n", kind, count)
		}
	}

	return nil
}

