package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yourusername/panka/pkg/parser/schema"
)

// Visualizer provides graph visualization and debugging utilities
type Visualizer struct{}

// NewVisualizer creates a new visualizer
func NewVisualizer() *Visualizer {
	return &Visualizer{}
}

// ToASCII generates an ASCII representation of the graph
func (v *Visualizer) ToASCII(g *Graph) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("Graph: %s\n", g.StackName))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	stats := g.GetStats()
	sb.WriteString(fmt.Sprintf("Nodes: %d\n", stats.NodeCount))
	sb.WriteString(fmt.Sprintf("Edges: %d\n", stats.EdgeCount))
	sb.WriteString(fmt.Sprintf("Levels: %d\n", stats.MaxDepth+1))
	sb.WriteString(fmt.Sprintf("Has Cycle: %v\n\n", stats.HasCycle))
	
	// Group nodes by level
	levels := make(map[int][]*Node)
	for _, node := range g.Nodes {
		levels[node.Level] = append(levels[node.Level], node)
	}
	
	// Sort levels
	levelKeys := make([]int, 0, len(levels))
	for level := range levels {
		levelKeys = append(levelKeys, level)
	}
	sort.Ints(levelKeys)
	
	// Print each level
	for _, level := range levelKeys {
		nodes := levels[level]
		
		// Sort nodes within level
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].ID < nodes[j].ID
		})
		
		sb.WriteString(fmt.Sprintf("Level %d:\n", level))
		for _, node := range nodes {
			sb.WriteString(fmt.Sprintf("  [%s] %s\n", node.Kind, node.ID))
			
			if len(node.DependsOn) > 0 {
				sb.WriteString(fmt.Sprintf("    depends on: %v\n", node.DependsOn))
			}
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}

// ToDOT generates a GraphViz DOT representation
func (v *Visualizer) ToDOT(g *Graph) string {
	var sb strings.Builder
	
	sb.WriteString("digraph deployment {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=rounded];\n\n")
	
	// Define nodes with colors by type
	for _, node := range g.Nodes {
		color := v.getColorForKind(node.Kind)
		label := fmt.Sprintf("%s\\n(%s)", node.ID, node.Kind)
		sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\", fillcolor=\"%s\", style=\"filled,rounded\"];\n",
			node.ID, label, color))
	}
	
	sb.WriteString("\n")
	
	// Define edges
	for from, edges := range g.Edges {
		for _, edge := range edges {
			style := "solid"
			if edge.Type == EdgeTypeImplicit {
				style = "dashed"
			}
			sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [style=%s];\n", 
				from, edge.To, style))
		}
	}
	
	sb.WriteString("}\n")
	
	return sb.String()
}

// getColorForKind returns a color for a resource kind
func (v *Visualizer) getColorForKind(kind schema.Kind) string {
	colors := map[schema.Kind]string{
		schema.KindMicroService: "lightblue",
		schema.KindRDS:         "lightgreen",
		schema.KindDynamoDB:    "lightgreen",
		schema.KindS3:          "lightyellow",
		schema.KindSQS:         "lightpink",
		schema.KindSNS:         "lightpink",
	}
	
	if color, exists := colors[kind]; exists {
		return color
	}
	return "lightgray"
}

// ToMermaid generates a Mermaid diagram representation
func (v *Visualizer) ToMermaid(g *Graph) string {
	var sb strings.Builder
	
	sb.WriteString("graph LR\n")
	
	// Define nodes
	nodeLabels := make(map[string]string)
	for _, node := range g.Nodes {
		label := fmt.Sprintf("%s[%s<br/>%s]", 
			v.sanitizeID(node.ID), node.ID, node.Kind)
		nodeLabels[node.ID] = label
		sb.WriteString(fmt.Sprintf("  %s\n", label))
	}
	
	sb.WriteString("\n")
	
	// Define edges
	for from, edges := range g.Edges {
		for _, edge := range edges {
			arrow := "-->"
			if edge.Type == EdgeTypeImplicit {
				arrow = "-.->"
			}
			sb.WriteString(fmt.Sprintf("  %s %s %s\n",
				v.sanitizeID(from), arrow, v.sanitizeID(edge.To)))
		}
	}
	
	// Add styling
	sb.WriteString("\n  classDef compute fill:#aed6f1\n")
	sb.WriteString("  classDef database fill:#a9dfbf\n")
	sb.WriteString("  classDef storage fill:#fef9e7\n")
	sb.WriteString("  classDef messaging fill:#f5b7b1\n")
	
	return sb.String()
}

// sanitizeID sanitizes an ID for use in Mermaid
func (v *Visualizer) sanitizeID(id string) string {
	return strings.ReplaceAll(id, "-", "_")
}

// PrintDependencyTree prints a tree view of dependencies
func (v *Visualizer) PrintDependencyTree(g *Graph, rootID string) string {
	var sb strings.Builder
	
	root, exists := g.GetNode(rootID)
	if !exists {
		return fmt.Sprintf("Node %s not found\n", rootID)
	}
	
	sb.WriteString(fmt.Sprintf("Dependency tree for: %s\n\n", rootID))
	
	visited := make(map[string]bool)
	v.printTreeNode(&sb, g, root, 0, visited)
	
	return sb.String()
}

// printTreeNode recursively prints a tree node
func (v *Visualizer) printTreeNode(sb *strings.Builder, g *Graph, node *Node, depth int, visited map[string]bool) {
	indent := strings.Repeat("  ", depth)
	
	marker := "├─"
	if depth == 0 {
		marker = "●"
	}
	
	sb.WriteString(fmt.Sprintf("%s%s %s (%s)\n", indent, marker, node.ID, node.Kind))
	
	if visited[node.ID] {
		sb.WriteString(fmt.Sprintf("%s  (already shown above)\n", indent))
		return
	}
	
	visited[node.ID] = true
	
	deps, _ := g.GetDependencies(node.ID)
	for i, dep := range deps {
		if i == len(deps)-1 {
			// Last dependency
			v.printTreeNode(sb, g, dep, depth+1, visited)
		} else {
			v.printTreeNode(sb, g, dep, depth+1, visited)
		}
	}
}

// PrintStats prints detailed statistics about the graph
func (v *Visualizer) PrintStats(g *Graph) string {
	var sb strings.Builder
	
	stats := g.GetStats()
	
	sb.WriteString(fmt.Sprintf("Graph Statistics for: %s\n", g.StackName))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	sb.WriteString(fmt.Sprintf("Nodes:           %d\n", stats.NodeCount))
	sb.WriteString(fmt.Sprintf("Edges:           %d\n", stats.EdgeCount))
	sb.WriteString(fmt.Sprintf("Root Nodes:      %d\n", stats.RootCount))
	sb.WriteString(fmt.Sprintf("Leaf Nodes:      %d\n", stats.LeafCount))
	sb.WriteString(fmt.Sprintf("Max Depth:       %d\n", stats.MaxDepth))
	sb.WriteString(fmt.Sprintf("Average Degree:  %.2f\n", stats.AverageDegree))
	sb.WriteString(fmt.Sprintf("Has Cycle:       %v\n\n", stats.HasCycle))
	
	// Node type breakdown
	kindCounts := make(map[schema.Kind]int)
	for _, node := range g.Nodes {
		kindCounts[node.Kind]++
	}
	
	sb.WriteString("Node Types:\n")
	for kind, count := range kindCounts {
		sb.WriteString(fmt.Sprintf("  %-20s %d\n", kind, count))
	}
	sb.WriteString("\n")
	
	// Level distribution
	levelCounts := make(map[int]int)
	for _, node := range g.Nodes {
		levelCounts[node.Level]++
	}
	
	sb.WriteString("Level Distribution:\n")
	for i := 0; i <= stats.MaxDepth; i++ {
		count := levelCounts[i]
		bar := strings.Repeat("█", count)
		sb.WriteString(fmt.Sprintf("  Level %d: %s (%d)\n", i, bar, count))
	}
	sb.WriteString("\n")
	
	// Root nodes
	roots := g.GetRootNodes()
	if len(roots) > 0 {
		sb.WriteString("Root Nodes (no dependencies):\n")
		for _, root := range roots {
			sb.WriteString(fmt.Sprintf("  - %s (%s)\n", root.ID, root.Kind))
		}
		sb.WriteString("\n")
	}
	
	// Leaf nodes
	leaves := g.GetLeafNodes()
	if len(leaves) > 0 {
		sb.WriteString("Leaf Nodes (no dependents):\n")
		for _, leaf := range leaves {
			sb.WriteString(fmt.Sprintf("  - %s (%s)\n", leaf.ID, leaf.Kind))
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}

// PrintPlan prints a formatted deployment plan
func (v *Visualizer) PrintPlan(plan *DeploymentPlan) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("╔%s╗\n", strings.Repeat("═", 60)))
	sb.WriteString(fmt.Sprintf("║ Deployment Plan: %-44s║\n", plan.StackName))
	sb.WriteString(fmt.Sprintf("╚%s╝\n\n", strings.Repeat("═", 60)))
	
	sb.WriteString(fmt.Sprintf("Created:         %s\n", plan.CreatedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Total Stages:    %d\n", plan.TotalStages))
	sb.WriteString(fmt.Sprintf("Total Resources: %d\n", plan.TotalResources))
	sb.WriteString(fmt.Sprintf("Estimated Time:  %s\n\n", plan.EstimatedTime))
	
	if plan.IsEmpty() {
		sb.WriteString("  (No resources to deploy)\n\n")
		return sb.String()
	}
	
	for _, stage := range plan.Stages {
		sb.WriteString(fmt.Sprintf("┌─ Stage %d %s\n", stage.Number, strings.Repeat("─", 52)))
		sb.WriteString(fmt.Sprintf("│  Level:             %d\n", stage.Level))
		sb.WriteString(fmt.Sprintf("│  Resources:         %d (parallel deployment)\n", len(stage.Resources)))
		sb.WriteString(fmt.Sprintf("│  Estimated Time:    %s\n", stage.EstimatedDuration))
		sb.WriteString("│\n")
		
		for i, resource := range stage.Resources {
			marker := "├"
			if i == len(stage.Resources)-1 {
				marker = "└"
			}
			
			sb.WriteString(fmt.Sprintf("│  %s─ %-20s [%-15s] %s\n",
				marker, resource.ID, resource.Kind, resource.Action))
			
			if len(resource.Dependencies) > 0 {
				sb.WriteString(fmt.Sprintf("│     depends on: %v\n", resource.Dependencies))
			}
		}
		
		sb.WriteString("│\n")
	}
	
	sb.WriteString(fmt.Sprintf("└%s\n\n", strings.Repeat("─", 62)))
	
	return sb.String()
}

