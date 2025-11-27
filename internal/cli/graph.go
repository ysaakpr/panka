package cli

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/graph"
	"github.com/yourusername/panka/pkg/parser"
	"go.uber.org/zap"
)

var (
	graphOutput string
	graphFile   string
)

// graphCmd represents the graph command
var graphCmd = &cobra.Command{
	Use:   "graph [file]",
	Short: "Visualize dependency graph",
	Long: `Generate and visualize the resource dependency graph.

Output formats:
  • ascii   - ASCII art (default, prints to console)
  • dot     - Graphviz DOT format
  • mermaid - Mermaid diagram format

Examples:
  panka graph infrastructure.yaml
  panka graph infrastructure.yaml --output dot --file graph.dot
  panka graph infrastructure.yaml --output mermaid --file graph.mmd`,
	Args: cobra.ExactArgs(1),
	RunE: runGraph,
}

func init() {
	rootCmd.AddCommand(graphCmd)
	
	graphCmd.Flags().StringVarP(&graphOutput, "output", "o", "ascii", "output format (ascii, dot, mermaid)")
	graphCmd.Flags().StringVarP(&graphFile, "file", "f", "", "output file (default: stdout)")
}

func runGraph(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan)
	red := color.New(color.FgRed, color.Bold)

	file := args[0]
	
	cyan.Printf("\n📊 Generating dependency graph for: %s\n\n", file)

	log := logger.Global()

	// Parse file
	p := parser.NewParser()
	result, err := p.ParseFile(file)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	log.Info("Resources parsed", zap.Int("count", len(result.Components)))

	// Build graph
	builder := graph.NewBuilder()
	g, err := builder.Build(result)
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}

	log.Info("Graph built",
		zap.Int("nodes", g.NodeCount()),
		zap.Int("edges", g.EdgeCount()),
	)

	// Generate visualization
	visualizer := graph.NewVisualizer()
	var output string

	switch graphOutput {
	case "ascii":
		output = visualizer.ToASCII(g)
	case "dot":
		output = visualizer.ToDOT(g)
	case "mermaid":
		output = visualizer.ToMermaid(g)
	default:
		return fmt.Errorf("unsupported output format: %s (use: ascii, dot, mermaid)", graphOutput)
	}

	// Write output
	if graphFile != "" {
		if err := os.WriteFile(graphFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		green.Printf("✅ Graph written to: %s\n", graphFile)
	} else {
		fmt.Println(output)
	}

	// Print stats
	stats := g.GetStats()
	cyan.Println("\n📈 Graph Statistics:")
	fmt.Printf("   • Total nodes:    %d\n", stats.NodeCount)
	fmt.Printf("   • Total edges:    %d\n", stats.EdgeCount)
	fmt.Printf("   • Root nodes:     %d\n", stats.RootCount)
	fmt.Printf("   • Leaf nodes:     %d\n", stats.LeafCount)
	fmt.Printf("   • Max depth:      %d\n", stats.MaxDepth)
	fmt.Printf("   • Avg degree:     %.2f\n", stats.AverageDegree)

	// Check for cycles
	if stats.HasCycle {
		red.Println("\n⚠️  Warning: Circular dependencies detected!")
	} else {
		green.Println("\n✅ No circular dependencies")
	}

	return nil
}

