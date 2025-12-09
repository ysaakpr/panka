package cli

import (
	"fmt"
	"os"
	"path/filepath"

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
	Use:   "graph <path>",
	Short: "Visualize dependency graph",
	Long: `Generate and visualize the resource dependency graph.

Supports both:
  • Stack folders (new structure with services/)
  • Single YAML files (legacy)

Output formats:
  • ascii   - ASCII art (default, prints to console)
  • dot     - Graphviz DOT format
  • mermaid - Mermaid diagram format

Examples:
  panka graph ./my-stack
  panka graph ./my-stack --output dot --file graph.dot
  panka graph infrastructure.yaml --output mermaid`,
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

	path := args[0]

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	cyan.Printf("\n📊 Generating dependency graph for: %s\n\n", absPath)

	log := logger.Global()

	// Check if path is a folder or file
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("path not found: %s", absPath)
	}

	var parseResult *parser.ParseResult

	if info.IsDir() {
		// Parse as stack folder
		fp := parser.NewFolderParser()
		folderResult, err := fp.ParseStackFolder(absPath)
		if err != nil {
			return fmt.Errorf("failed to parse stack folder: %w", err)
		}

		// Convert to ParseResult
		parseResult = &parser.ParseResult{
			Stack:      folderResult.Stack,
			Components: folderResult.AllComponents,
		}
		for _, svc := range folderResult.Services {
			if svc.Service != nil {
				parseResult.Services = append(parseResult.Services, svc.Service)
			}
		}
	} else {
		// Parse as single file (legacy)
		p := parser.NewParser()
		parseResult, err = p.ParseFile(absPath)
		if err != nil {
			return fmt.Errorf("failed to parse file: %w", err)
		}
	}

	log.Info("Resources parsed", zap.Int("count", len(parseResult.Components)))

	// Build graph
	builder := graph.NewBuilder()
	g, err := builder.Build(parseResult)
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

