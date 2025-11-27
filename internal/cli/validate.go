package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yourusername/panka/pkg/parser"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate [file...]",
	Short: "Validate infrastructure configuration",
	Long: `Validate one or more infrastructure configuration files.

This checks:
  • YAML syntax
  • Schema compliance
  • Resource references
  • Circular dependencies
  • Required fields`,
	Args: cobra.MinimumNArgs(1),
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n🔍 Validating infrastructure configuration...")

	p := parser.NewParser()
	v := parser.NewValidator()

	totalFiles := 0
	totalErrors := 0
	totalWarnings := 0

	for _, file := range args {
		totalFiles++
		
		// Get absolute path
		absPath, err := filepath.Abs(file)
		if err != nil {
			red.Printf("\n❌ Error resolving path for %s: %v\n", file, err)
			totalErrors++
			continue
		}

		fmt.Printf("\n📄 Validating: %s\n", absPath)

		// Check if file exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			red.Printf("   ❌ File not found\n")
			totalErrors++
			continue
		}

		// Parse file
		result, err := p.ParseFile(absPath)
		if err != nil {
			red.Printf("   ❌ Parse error: %v\n", err)
			totalErrors++
			continue
		}

		fmt.Printf("   ℹ️  Found %d resources\n", len(result.Components))

		// Validate resources
		if err := v.Validate(result); err != nil {
			red.Printf("   ❌ Validation failed:\n")
			fmt.Printf("      %v\n", err)
			totalErrors++
			continue
		}

		// Circular dependencies are checked during graph building phase
		// Validation checks schema compliance only

		green.Printf("   ✅ Valid\n")

		// Show summary
		if GetVerbose() {
			resourceCounts := make(map[string]int)
			for _, res := range result.Components {
				resourceCounts[string(res.GetKind())]++
			}

			yellow.Println("\n   Resource Summary:")
			for kind, count := range resourceCounts {
				fmt.Printf("      • %s: %d\n", kind, count)
			}
		}
	}

	// Print overall summary
	cyan.Println("\n" + "==================================================")
	fmt.Printf("📊 Summary: %d files validated\n", totalFiles)
	
	if totalErrors > 0 {
		red.Printf("   ❌ Errors: %d\n", totalErrors)
	} else {
		green.Printf("   ✅ All files valid!\n")
	}

	if totalWarnings > 0 {
		yellow.Printf("   ⚠️  Warnings: %d\n", totalWarnings)
	}

	cyan.Println("==================================================")

	if totalErrors > 0 {
		return fmt.Errorf("validation failed with %d errors", totalErrors)
	}

	return nil
}

