package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/graph"
	"github.com/yourusername/panka/pkg/parser"
	"go.uber.org/zap"
)

var (
	destroyForce  bool
	destroyDryRun bool
	destroyAuto   bool
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy [file]",
	Short: "Destroy infrastructure resources",
	Long: `Destroy all resources defined in the infrastructure configuration.

Resources are destroyed in reverse dependency order to ensure clean teardown.

⚠️  WARNING: This action is destructive and cannot be undone!

The destroy command will:
  • Parse your infrastructure configuration
  • Build the dependency graph
  • Generate a destruction plan (reverse order)
  • Prompt for confirmation (unless --auto-approve)
  • Destroy resources one by one

Flags:
  --dry-run       Show what would be destroyed without doing it
  --force         Skip dependency checks (dangerous!)
  --auto-approve  Skip confirmation prompt`,
	Args: cobra.ExactArgs(1),
	RunE: runDestroy,
}

func init() {
	rootCmd.AddCommand(destroyCmd)
	
	destroyCmd.Flags().BoolVar(&destroyForce, "force", false, "force destruction, skip dependency checks")
	destroyCmd.Flags().BoolVar(&destroyDryRun, "dry-run", false, "show what would be destroyed")
	destroyCmd.Flags().BoolVar(&destroyAuto, "auto-approve", false, "skip confirmation prompt")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow, color.Bold)

	file := args[0]
	
	red.Printf("\n🗑️  Preparing to DESTROY infrastructure: %s\n\n", file)

	if destroyDryRun {
		cyan.Println("🔍 DRY-RUN MODE - No resources will be destroyed")
	}

	log := logger.Global()

	// Step 1: Parse configuration
	fmt.Print("🔍 Parsing configuration... ")
	p := parser.NewParser()
	result, err := p.ParseFile(file)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}
	green.Println("✓")

	if result.Stack == nil {
		return fmt.Errorf("no Stack definition found in configuration")
	}

	log.Info("Resources parsed",
		zap.String("stack", result.Stack.Metadata.Name),
		zap.Int("components", len(result.Components)),
	)

	// Step 2: Build dependency graph
	fmt.Print("🔗 Building dependency graph... ")
	builder := graph.NewBuilder()
	g, err := builder.Build(result)
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}
	green.Println("✓")

	// Step 3: Generate destruction plan (reverse order)
	fmt.Print("📊 Generating destruction plan... ")
	planner := graph.NewPlanner()
	plan, err := planner.CreateDeploymentPlan(g, graph.ActionDelete)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}
	green.Println("✓\n")

	// Reverse the stages for destruction
	reversedStages := make([]*graph.DeploymentStage, len(plan.Stages))
	for i, stage := range plan.Stages {
		reversedStages[len(plan.Stages)-1-i] = stage
	}
	plan.Stages = reversedStages

	// Display destruction plan
	displayDestructionPlan(plan)

	// Show warning
	yellow.Println("\n⚠️  WARNING: This action is DESTRUCTIVE and CANNOT be undone!")
	red.Printf("   %d resources will be PERMANENTLY DELETED\n\n", len(result.Components))

	// Get confirmation unless auto-approved or dry-run
	if !destroyDryRun && !destroyAuto {
		if !confirmDestruction(result.Stack.Metadata.Name) {
			cyan.Println("\n✋ Destruction cancelled by user")
			return nil
		}
	}

	// Execute destruction
	if destroyDryRun {
		cyan.Println("\n🔍 DRY-RUN: No resources were actually destroyed")
		green.Println("✨ Dry-run complete!")
	} else {
		red.Println("\n🔥 Starting destruction...")
		
		// In a real implementation, this would:
		// 1. Acquire lock
		// 2. Load current state
		// 3. For each stage (reversed):
		//    - For each resource in stage:
		//      - Call provider.Delete()
		//      - Update state
		// 4. Release lock
		
		yellow.Println("\n⚠️  Note: Full destruction implementation requires state backend")
		cyan.Println("   This would:")
		
		for i, stage := range plan.Stages {
			fmt.Printf("   Stage %d: Destroy %d resources\n", i+1, len(stage.Resources))
			for _, res := range stage.Resources {
				fmt.Printf("     - %s (%s)\n", res.ID, res.Kind)
			}
		}
		
		green.Println("\n✨ Destruction plan ready (actual execution requires 'apply' implementation)")
	}

	return nil
}

func displayDestructionPlan(plan *graph.DeploymentPlan) {
	cyan := color.New(color.FgCyan, color.Bold)
	red := color.New(color.FgRed)
	magenta := color.New(color.FgMagenta)

	cyan.Println("🗑️  Destruction Plan (Reverse Order):")
	cyan.Println(strings.Repeat("─", 50))

	for _, stage := range plan.Stages {
		fmt.Printf("\n")
		magenta.Printf("Stage %d", stage.Number)
		if len(stage.Resources) > 1 {
			fmt.Printf(" (%d resources)", len(stage.Resources))
		} else {
			fmt.Printf(" (1 resource)")
		}
		fmt.Println()

		for _, res := range stage.Resources {
			red.Printf("  - Delete ")
			fmt.Printf("[%s] %s\n", res.Kind, res.ID)
		}
	}
}

func confirmDestruction(stackName string) bool {
	reader := bufio.NewReader(os.Stdin)
	
	yellow := color.New(color.FgYellow, color.Bold)
	yellow.Printf("Type the stack name '%s' to confirm destruction: ", stackName)
	
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	input = strings.TrimSpace(input)
	return input == stackName
}

