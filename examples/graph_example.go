// +build ignore

package main

import (
	"fmt"

	"github.com/yourusername/panka/pkg/graph"
	"github.com/yourusername/panka/pkg/parser"
)

func main() {
	// Parse the stack configuration
	p := parser.NewParser()
	result, err := p.ParseFile("examples/simple-stack.yaml")
	if err != nil {
		panic(fmt.Sprintf("Failed to parse: %v", err))
	}

	// Build dependency graph
	builder := graph.NewBuilder()
	g, err := builder.Build(result)
	if err != nil {
		panic(fmt.Sprintf("Failed to build graph: %v", err))
	}

	// Print graph statistics
	vis := graph.NewVisualizer()
	fmt.Println("=== GRAPH STATISTICS ===")
	fmt.Print(vis.PrintStats(g))
	fmt.Println()

	// Print ASCII representation
	fmt.Println("=== GRAPH VISUALIZATION ===")
	fmt.Print(vis.ToASCII(g))
	fmt.Println()

	// Perform topological sort
	sorter := graph.NewSorter()
	sorted, err := sorter.TopologicalSort(g)
	if err != nil {
		panic(fmt.Sprintf("Failed to sort: %v", err))
	}

	fmt.Println("=== DEPLOYMENT ORDER ===")
	for i, node := range sorted {
		fmt.Printf("%d. %s (%s)\n", i+1, node.ID, node.Kind)
	}
	fmt.Println()

	// Get deployment levels
	levels, err := sorter.SortByLevel(g)
	if err != nil {
		panic(fmt.Sprintf("Failed to sort by level: %v", err))
	}

	fmt.Println("=== DEPLOYMENT LEVELS ===")
	for i, level := range levels {
		fmt.Printf("Level %d (parallel deployment):\n", i)
		for _, node := range level {
			fmt.Printf("  - %s (%s)\n", node.ID, node.Kind)
		}
	}
	fmt.Println()

	// Create deployment plan
	planner := graph.NewPlanner()
	plan, err := planner.CreateDeploymentPlan(g, graph.ActionCreate)
	if err != nil {
		panic(fmt.Sprintf("Failed to create plan: %v", err))
	}

	fmt.Println("=== DEPLOYMENT PLAN ===")
	fmt.Print(vis.PrintPlan(plan))

	// Show critical path
	criticalPath, err := sorter.GetCriticalPath(g)
	if err != nil {
		panic(fmt.Sprintf("Failed to get critical path: %v", err))
	}

	fmt.Println("=== CRITICAL PATH ===")
	fmt.Println("(Longest dependency chain - determines minimum deployment time)")
	for i, node := range criticalPath {
		if i > 0 {
			fmt.Print(" -> ")
		}
		fmt.Printf("%s", node.ID)
	}
	fmt.Println("\n")

	// Generate GraphViz DOT format (for visualization tools)
	fmt.Println("=== GRAPHVIZ DOT FORMAT ===")
	fmt.Println("(Copy to https://dreampuf.github.io/GraphvizOnline/ to visualize)")
	fmt.Println(vis.ToDOT(g))
}

