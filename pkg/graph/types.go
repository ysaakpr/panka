package graph

import (
	"fmt"
	"time"

	"github.com/yourusername/panka/pkg/parser/schema"
)

// Node represents a node in the dependency graph
type Node struct {
	// Resource information
	ID       string          // Unique identifier (metadata.name)
	Kind     schema.Kind     // Resource kind
	Resource schema.Resource // The actual resource
	
	// Dependency information
	DependsOn []string // List of dependency IDs
	
	// Graph metadata
	Level      int      // Deployment level (0 = no dependencies)
	InDegree   int      // Number of incoming edges
	Visited    bool     // For graph traversal
	Processing bool     // For cycle detection
}

// Edge represents a dependency edge in the graph
type Edge struct {
	From string // Source node ID
	To   string // Target node ID
	Type EdgeType
}

// EdgeType defines the type of dependency relationship
type EdgeType string

const (
	// EdgeTypeExplicit represents an explicit dependency (dependsOn)
	EdgeTypeExplicit EdgeType = "explicit"
	
	// EdgeTypeImplicit represents an implicit dependency (e.g., valueFrom)
	EdgeTypeImplicit EdgeType = "implicit"
	
	// EdgeTypeOrder represents an ordering constraint
	EdgeTypeOrder EdgeType = "order"
)

// Graph represents the dependency graph of resources
type Graph struct {
	// Nodes indexed by resource ID
	Nodes map[string]*Node
	
	// Adjacency list (from -> to)
	Edges map[string][]*Edge
	
	// Reverse adjacency list (to -> from)
	ReverseEdges map[string][]*Edge
	
	// Metadata
	StackName   string
	ServiceName string
	BuildTime   time.Time
}

// NewGraph creates a new empty graph
func NewGraph(stackName string) *Graph {
	return &Graph{
		Nodes:        make(map[string]*Node),
		Edges:        make(map[string][]*Edge),
		ReverseEdges: make(map[string][]*Edge),
		StackName:    stackName,
		BuildTime:    time.Now(),
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(node *Node) error {
	if node.ID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}
	
	if _, exists := g.Nodes[node.ID]; exists {
		return fmt.Errorf("node %s already exists in graph", node.ID)
	}
	
	g.Nodes[node.ID] = node
	return nil
}

// AddEdge adds an edge to the graph
// from depends on to (to must be deployed before from)
func (g *Graph) AddEdge(from, to string, edgeType EdgeType) error {
	// Verify nodes exist
	if _, exists := g.Nodes[from]; !exists {
		return fmt.Errorf("source node %s does not exist", from)
	}
	if _, exists := g.Nodes[to]; !exists {
		return fmt.Errorf("target node %s does not exist", to)
	}
	
	// Create edge
	edge := &Edge{
		From: from,
		To:   to,
		Type: edgeType,
	}
	
	// Add to adjacency list (from -> to means "from depends on to")
	g.Edges[from] = append(g.Edges[from], edge)
	
	// Add to reverse adjacency list
	g.ReverseEdges[to] = append(g.ReverseEdges[to], edge)
	
	// Update in-degree: "from" has one more dependency
	// InDegree represents the number of dependencies a node has
	g.Nodes[from].InDegree++
	
	return nil
}

// GetNode retrieves a node by ID
func (g *Graph) GetNode(id string) (*Node, bool) {
	node, exists := g.Nodes[id]
	return node, exists
}

// GetDependencies returns all direct dependencies of a node
func (g *Graph) GetDependencies(id string) ([]*Node, error) {
	edges, exists := g.Edges[id]
	if !exists {
		return []*Node{}, nil
	}
	
	deps := make([]*Node, 0, len(edges))
	for _, edge := range edges {
		if node, exists := g.Nodes[edge.To]; exists {
			deps = append(deps, node)
		}
	}
	
	return deps, nil
}

// GetDependents returns all nodes that depend on this node
func (g *Graph) GetDependents(id string) ([]*Node, error) {
	edges, exists := g.ReverseEdges[id]
	if !exists {
		return []*Node{}, nil
	}
	
	dependents := make([]*Node, 0, len(edges))
	for _, edge := range edges {
		if node, exists := g.Nodes[edge.From]; exists {
			dependents = append(dependents, node)
		}
	}
	
	return dependents, nil
}

// GetRootNodes returns all nodes with no dependencies
func (g *Graph) GetRootNodes() []*Node {
	roots := make([]*Node, 0)
	for _, node := range g.Nodes {
		if node.InDegree == 0 {
			roots = append(roots, node)
		}
	}
	return roots
}

// GetLeafNodes returns all nodes with no dependents
func (g *Graph) GetLeafNodes() []*Node {
	leaves := make([]*Node, 0)
	for id, node := range g.Nodes {
		if len(g.Edges[id]) == 0 {
			leaves = append(leaves, node)
		}
	}
	return leaves
}

// Clone creates a deep copy of the graph for safe manipulation
func (g *Graph) Clone() *Graph {
	clone := &Graph{
		Nodes:        make(map[string]*Node),
		Edges:        make(map[string][]*Edge),
		ReverseEdges: make(map[string][]*Edge),
		StackName:    g.StackName,
		ServiceName:  g.ServiceName,
		BuildTime:    g.BuildTime,
	}
	
	// Clone nodes
	for id, node := range g.Nodes {
		nodeCopy := *node
		nodeCopy.DependsOn = make([]string, len(node.DependsOn))
		copy(nodeCopy.DependsOn, node.DependsOn)
		clone.Nodes[id] = &nodeCopy
	}
	
	// Clone edges
	for from, edges := range g.Edges {
		edgesCopy := make([]*Edge, len(edges))
		for i, edge := range edges {
			edgeCopy := *edge
			edgesCopy[i] = &edgeCopy
		}
		clone.Edges[from] = edgesCopy
	}
	
	// Clone reverse edges
	for to, edges := range g.ReverseEdges {
		edgesCopy := make([]*Edge, len(edges))
		for i, edge := range edges {
			edgeCopy := *edge
			edgesCopy[i] = &edgeCopy
		}
		clone.ReverseEdges[to] = edgesCopy
	}
	
	return clone
}

// Reset resets traversal state for all nodes
func (g *Graph) Reset() {
	for _, node := range g.Nodes {
		node.Visited = false
		node.Processing = false
	}
}

// NodeCount returns the number of nodes in the graph
func (g *Graph) NodeCount() int {
	return len(g.Nodes)
}

// EdgeCount returns the total number of edges in the graph
func (g *Graph) EdgeCount() int {
	count := 0
	for _, edges := range g.Edges {
		count += len(edges)
	}
	return count
}

// IsEmpty returns true if the graph has no nodes
func (g *Graph) IsEmpty() bool {
	return len(g.Nodes) == 0
}

// HasCycle returns true if the graph contains a cycle
func (g *Graph) HasCycle() bool {
	g.Reset()
	
	var hasCycle func(id string) bool
	hasCycle = func(id string) bool {
		node := g.Nodes[id]
		
		if node.Processing {
			return true // Back edge found, cycle detected
		}
		
		if node.Visited {
			return false // Already processed
		}
		
		node.Processing = true
		
		// Check all dependencies
		for _, edge := range g.Edges[id] {
			if hasCycle(edge.To) {
				return true
			}
		}
		
		node.Processing = false
		node.Visited = true
		return false
	}
	
	// Check from each unvisited node
	for id := range g.Nodes {
		if !g.Nodes[id].Visited {
			if hasCycle(id) {
				return true
			}
		}
	}
	
	return false
}

// GetCycle returns the nodes involved in a cycle, if one exists
func (g *Graph) GetCycle() []string {
	g.Reset()
	path := make([]string, 0)
	cycleStart := -1
	
	var dfs func(id string) bool
	dfs = func(id string) bool {
		node := g.Nodes[id]
		
		if node.Processing {
			// Found cycle, mark where it starts
			for i, nodeID := range path {
				if nodeID == id {
					cycleStart = i
					return true
				}
			}
			return true
		}
		
		if node.Visited {
			return false
		}
		
		node.Processing = true
		path = append(path, id)
		
		// Check all dependencies
		for _, edge := range g.Edges[id] {
			if dfs(edge.To) {
				return true
			}
		}
		
		path = path[:len(path)-1]
		node.Processing = false
		node.Visited = true
		return false
	}
	
	// Check from each unvisited node
	for id := range g.Nodes {
		if !g.Nodes[id].Visited {
			if dfs(id) {
				if cycleStart >= 0 {
					return path[cycleStart:]
				}
				return path
			}
		}
	}
	
	return nil
}

// Stats returns statistics about the graph
type GraphStats struct {
	NodeCount      int
	EdgeCount      int
	RootCount      int
	LeafCount      int
	MaxDepth       int
	HasCycle       bool
	AverageDegree  float64
}

// GetStats returns statistics about the graph
func (g *Graph) GetStats() GraphStats {
	stats := GraphStats{
		NodeCount: g.NodeCount(),
		EdgeCount: g.EdgeCount(),
		RootCount: len(g.GetRootNodes()),
		LeafCount: len(g.GetLeafNodes()),
		HasCycle:  g.HasCycle(),
	}
	
	if stats.NodeCount > 0 {
		stats.AverageDegree = float64(stats.EdgeCount) / float64(stats.NodeCount)
	}
	
	// Calculate max depth
	maxDepth := 0
	for _, node := range g.Nodes {
		if node.Level > maxDepth {
			maxDepth = node.Level
		}
	}
	stats.MaxDepth = maxDepth
	
	return stats
}

