package graph

import (
	"fmt"
	"sort"

	"github.com/yourusername/panka/internal/logger"
	"go.uber.org/zap"
)

// Sorter performs topological sorting on the graph
type Sorter struct {
	logger *logger.Logger
}

// NewSorter creates a new graph sorter
func NewSorter() *Sorter {
	return &Sorter{
		logger: logger.Global(),
	}
}

// TopologicalSort performs a topological sort on the graph
// Returns nodes in deployment order (dependencies first)
func (s *Sorter) TopologicalSort(g *Graph) ([]*Node, error) {
	if g.IsEmpty() {
		return []*Node{}, nil
	}
	
	// Check for cycles
	if g.HasCycle() {
		cycle := g.GetCycle()
		return nil, fmt.Errorf("cannot sort graph with cycles: %v", cycle)
	}
	
	s.logger.Info("Performing topological sort", zap.Int("nodes", g.NodeCount()))
	
	// Use Kahn's algorithm for topological sorting
	result := make([]*Node, 0, g.NodeCount())
	
	// Clone the graph to avoid modifying the original
	workGraph := g.Clone()
	
	// Queue of nodes with no incoming edges
	queue := make([]*Node, 0)
	
	// Initialize queue with root nodes (in-degree = 0)
	for _, node := range workGraph.Nodes {
		if node.InDegree == 0 {
			queue = append(queue, node)
		}
	}
	
	// Sort queue by node ID for deterministic output
	sort.Slice(queue, func(i, j int) bool {
		return queue[i].ID < queue[j].ID
	})
	
	// Process nodes
	for len(queue) > 0 {
		// Dequeue
		node := queue[0]
		queue = queue[1:]
		
		// Add to result
		result = append(result, node)
		
		s.logger.Debug("Sorted node",
			zap.String("id", node.ID),
			zap.String("kind", string(node.Kind)),
			zap.Int("level", node.Level),
		)
		
		// Get dependents (nodes that depend on this one)
		dependents, _ := workGraph.GetDependents(node.ID)
		
		// Reduce in-degree for dependents
		for _, dependent := range dependents {
			dependent.InDegree--
			
			// If in-degree becomes 0, add to queue
			if dependent.InDegree == 0 {
				queue = append(queue, dependent)
			}
		}
		
		// Sort queue for deterministic output
		sort.Slice(queue, func(i, j int) bool {
			return queue[i].ID < queue[j].ID
		})
	}
	
	// Check if all nodes were processed
	if len(result) != g.NodeCount() {
		return nil, fmt.Errorf("topological sort failed: only sorted %d of %d nodes", 
			len(result), g.NodeCount())
	}
	
	s.logger.Info("Topological sort complete", zap.Int("nodes", len(result)))
	
	return result, nil
}

// SortByLevel returns nodes grouped by deployment level
// All nodes in level N can be deployed in parallel
func (s *Sorter) SortByLevel(g *Graph) ([][]*Node, error) {
	if g.IsEmpty() {
		return [][]*Node{}, nil
	}
	
	// Check for cycles
	if g.HasCycle() {
		cycle := g.GetCycle()
		return nil, fmt.Errorf("cannot sort graph with cycles: %v", cycle)
	}
	
	s.logger.Info("Sorting by deployment level", zap.Int("nodes", g.NodeCount()))
	
	// Find max level
	maxLevel := 0
	for _, node := range g.Nodes {
		if node.Level > maxLevel {
			maxLevel = node.Level
		}
	}
	
	// Create buckets for each level
	levels := make([][]*Node, maxLevel+1)
	for i := range levels {
		levels[i] = make([]*Node, 0)
	}
	
	// Group nodes by level
	for _, node := range g.Nodes {
		levels[node.Level] = append(levels[node.Level], node)
	}
	
	// Sort nodes within each level by ID for deterministic output
	for i := range levels {
		sort.Slice(levels[i], func(a, b int) bool {
			return levels[i][a].ID < levels[i][b].ID
		})
		
		s.logger.Debug("Level",
			zap.Int("level", i),
			zap.Int("nodes", len(levels[i])),
		)
	}
	
	s.logger.Info("Sorting by level complete", zap.Int("levels", len(levels)))
	
	return levels, nil
}

// ReverseTopologicalSort returns nodes in reverse deployment order
// Useful for deletion/teardown operations
func (s *Sorter) ReverseTopologicalSort(g *Graph) ([]*Node, error) {
	sorted, err := s.TopologicalSort(g)
	if err != nil {
		return nil, err
	}
	
	// Reverse the slice
	reversed := make([]*Node, len(sorted))
	for i, node := range sorted {
		reversed[len(sorted)-1-i] = node
	}
	
	s.logger.Info("Reverse topological sort complete", zap.Int("nodes", len(reversed)))
	
	return reversed, nil
}

// GetDeploymentBatches returns batches of nodes that can be deployed in parallel
// Each batch contains nodes with no dependencies on other nodes in later batches
func (s *Sorter) GetDeploymentBatches(g *Graph, maxBatchSize int) ([][]*Node, error) {
	levels, err := s.SortByLevel(g)
	if err != nil {
		return nil, err
	}
	
	// If maxBatchSize is 0, don't limit batch size
	if maxBatchSize <= 0 {
		return levels, nil
	}
	
	// Split large levels into smaller batches
	batches := make([][]*Node, 0)
	
	for _, level := range levels {
		// Split level into batches of maxBatchSize
		for i := 0; i < len(level); i += maxBatchSize {
			end := i + maxBatchSize
			if end > len(level) {
				end = len(level)
			}
			batches = append(batches, level[i:end])
		}
	}
	
	s.logger.Info("Created deployment batches",
		zap.Int("batches", len(batches)),
		zap.Int("max_batch_size", maxBatchSize),
	)
	
	return batches, nil
}

// ValidateOrder verifies that the given order respects dependencies
func (s *Sorter) ValidateOrder(g *Graph, order []*Node) error {
	// Build a map of node positions in the order
	positions := make(map[string]int)
	for i, node := range order {
		positions[node.ID] = i
	}
	
	// Check each node's dependencies
	for _, node := range order {
		deps, err := g.GetDependencies(node.ID)
		if err != nil {
			return err
		}
		
		nodePos := positions[node.ID]
		
		// Verify all dependencies come before this node
		for _, dep := range deps {
			depPos, exists := positions[dep.ID]
			if !exists {
				return fmt.Errorf("dependency %s not found in order", dep.ID)
			}
			
			if depPos >= nodePos {
				return fmt.Errorf("invalid order: %s (pos %d) depends on %s (pos %d)",
					node.ID, nodePos, dep.ID, depPos)
			}
		}
	}
	
	s.logger.Info("Order validation successful")
	return nil
}

// GetCriticalPath finds the longest path through the graph
// This represents the minimum deployment time if all parallel operations are maximized
func (s *Sorter) GetCriticalPath(g *Graph) ([]*Node, error) {
	if g.IsEmpty() {
		return []*Node{}, nil
	}
	
	// Find the node with the highest level
	var deepestNode *Node
	maxLevel := -1
	
	for _, node := range g.Nodes {
		if node.Level > maxLevel {
			maxLevel = node.Level
			deepestNode = node
		}
	}
	
	if deepestNode == nil {
		return []*Node{}, nil
	}
	
	// Trace back the critical path
	path := make([]*Node, 0)
	current := deepestNode
	
	for current != nil {
		path = append([]*Node{current}, path...) // Prepend
		
		// Find the dependency with the highest level
		deps, err := g.GetDependencies(current.ID)
		if err != nil {
			return nil, err
		}
		
		var nextNode *Node
		maxDepLevel := -1
		
		for _, dep := range deps {
			if dep.Level > maxDepLevel {
				maxDepLevel = dep.Level
				nextNode = dep
			}
		}
		
		current = nextNode
	}
	
	s.logger.Info("Critical path found",
		zap.Int("length", len(path)),
		zap.Int("depth", maxLevel),
	)
	
	return path, nil
}

