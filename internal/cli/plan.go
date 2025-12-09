package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/graph"
	"github.com/yourusername/panka/pkg/parser"
	"go.uber.org/zap"
)

var (
	planDetailed bool
	planFile     string
)

// planCmd represents the plan command
var planCmd = &cobra.Command{
	Use:   "plan <path>",
	Short: "Generate and show deployment plan",
	Long: `Generate a deployment plan showing what resources will be created,
updated, or deleted.

Supports both:
  • Stack folders (new structure with services/)
  • Single YAML files (legacy)

The plan command:
  • Parses your infrastructure configuration
  • Builds the dependency graph
  • Determines deployment order
  • Shows estimated changes

No actual changes are made - this is a dry-run to preview actions.

Examples:
  panka plan ./my-stack
  panka plan ./my-stack --detailed
  panka plan infrastructure.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: runPlan,
}

func init() {
	rootCmd.AddCommand(planCmd)

	planCmd.Flags().BoolVarP(&planDetailed, "detailed", "d", false, "show detailed resource information")
	planCmd.Flags().StringVarP(&planFile, "out", "o", "", "write plan to file")
}

func runPlan(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow)

	path := args[0]

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	cyan.Printf("\n📋 Generating deployment plan for: %s\n\n", absPath)

	log := logger.Global()

	// Check if path is a folder or file
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("path not found: %s", absPath)
	}

	var result *parser.ParseResult

	// Step 1: Parse configuration
	fmt.Print("🔍 Parsing configuration... ")
	if info.IsDir() {
		// Parse as stack folder
		fp := parser.NewFolderParser()
		folderResult, err := fp.ParseStackFolder(absPath)
		if err != nil {
			return fmt.Errorf("failed to parse stack folder: %w", err)
		}

		// Convert to ParseResult
		result = &parser.ParseResult{
			Stack:      folderResult.Stack,
			Components: folderResult.AllComponents,
		}
		for _, svc := range folderResult.Services {
			if svc.Service != nil {
				result.Services = append(result.Services, svc.Service)
			}
		}
	} else {
		// Parse as single file (legacy)
		p := parser.NewParser()
		result, err = p.ParseFile(absPath)
		if err != nil {
			return fmt.Errorf("failed to parse file: %w", err)
		}
	}
	green.Println("✓")

	if result.Stack == nil {
		return fmt.Errorf("no Stack definition found in configuration")
	}

	log.Info("Resources parsed",
		zap.String("stack", result.Stack.Metadata.Name),
		zap.Int("components", len(result.Components)),
	)

	// Step 2: Validate configuration
	fmt.Print("✅ Validating resources... ")
	v := parser.NewValidator()
	if err := v.Validate(result); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	green.Println("✓")

	// Step 3: Build dependency graph
	fmt.Print("🔗 Building dependency graph... ")
	builder := graph.NewBuilder()
	g, err := builder.Build(result)
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}
	green.Println("✓")

	log.Info("Graph built",
		zap.Int("nodes", g.NodeCount()),
		zap.Int("edges", g.EdgeCount()),
	)

	// Check for cycles
	if g.HasCycle() {
		return fmt.Errorf("circular dependency detected - cannot generate plan")
	}

	// Step 4: Generate deployment plan
	fmt.Print("📊 Generating deployment plan... ")
	planner := graph.NewPlanner()
	plan, err := planner.CreateDeploymentPlan(g, graph.ActionCreate)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}
	green.Println("✓\n")

	// Display plan
	displayPlan(plan, result, planDetailed)

	// Show summary
	cyan.Println("\n==================================================")
	fmt.Println("📊 Deployment Plan Summary")
	cyan.Println("==================================================")

	fmt.Printf("\nStack:      %s\n", result.Stack.Metadata.Name)
	fmt.Printf("Resources:  %d\n", len(result.Components))
	fmt.Printf("Stages:     %d\n", plan.TotalStages)

	// Estimate duration
	estimatedMinutes := len(plan.Stages) * 2 // Rough estimate: 2 min per stage
	fmt.Printf("Estimated Duration: ~%d minutes\n", estimatedMinutes)

	cyan.Println("\n==================================================")
	yellow.Println("\n⚠️  This is a plan preview. No resources will be created.")
	fmt.Println("   Run 'panka apply' to execute this plan.")
	green.Println("\n✨ Plan generation complete!")

	return nil
}

func displayPlan(plan *graph.DeploymentPlan, result *parser.ParseResult, detailed bool) {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	magenta := color.New(color.FgMagenta)

	cyan.Println("🚀 Deployment Plan:")
	cyan.Println(strings.Repeat("─", 50))

	for _, stage := range plan.Stages {
		fmt.Printf("\n")
		magenta.Printf("Stage %d", stage.Number)
		if len(stage.Resources) > 1 {
			fmt.Printf(" (parallel execution - %d resources)", len(stage.Resources))
		} else {
			fmt.Printf(" (1 resource)")
		}
		fmt.Println()

		for _, res := range stage.Resources {
			// Resource will be created
			green.Printf("  + Create ")
			fmt.Printf("[%s] %s\n", res.Kind, res.ID)

			if detailed {
				// Show dependencies
				if len(res.Dependencies) > 0 {
					fmt.Printf("      Dependencies: %v\n", res.Dependencies)
				}

				// Show resource metadata
				if res.Resource != nil {
					metadata := res.Resource.GetMetadata()
					if len(metadata.Labels) > 0 {
						fmt.Printf("      Labels: %v\n", metadata.Labels)
					}
				}
			}
		}
	}
}

