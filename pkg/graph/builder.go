package graph

import (
	"fmt"

	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/parser"
	"github.com/yourusername/panka/pkg/parser/schema"
	"go.uber.org/zap"
)

// Builder builds a dependency graph from parsed resources
type Builder struct {
	logger *logger.Logger
}

// NewBuilder creates a new graph builder
func NewBuilder() *Builder {
	return &Builder{
		logger: logger.Global(),
	}
}

// Build builds a dependency graph from parsed resources
func (b *Builder) Build(result *parser.ParseResult) (*Graph, error) {
	if result.Stack == nil {
		return nil, fmt.Errorf("parse result has no stack")
	}
	
	b.logger.Info("Building dependency graph", zap.String("stack", result.Stack.Metadata.Name))
	
	// Create graph
	graph := NewGraph(result.Stack.Metadata.Name)
	
	// Add all nodes first
	for _, component := range result.Components {
		node := b.createNode(component)
		if err := graph.AddNode(node); err != nil {
			return nil, fmt.Errorf("failed to add node %s: %w", node.ID, err)
		}
		
		b.logger.Debug("Added node", 
			zap.String("id", node.ID),
			zap.String("kind", string(node.Kind)),
		)
	}
	
	// Add edges based on dependencies
	for _, component := range result.Components {
		if err := b.addEdges(graph, component); err != nil {
			return nil, fmt.Errorf("failed to add edges for %s: %w", 
				component.GetMetadata().Name, err)
		}
	}
	
	// Detect cycles
	if graph.HasCycle() {
		cycle := graph.GetCycle()
		return nil, fmt.Errorf("circular dependency detected: %v", cycle)
	}
	
	// Calculate deployment levels
	if err := b.calculateLevels(graph); err != nil {
		return nil, fmt.Errorf("failed to calculate deployment levels: %w", err)
	}
	
	stats := graph.GetStats()
	b.logger.Info("Graph built successfully",
		zap.Int("nodes", stats.NodeCount),
		zap.Int("edges", stats.EdgeCount),
		zap.Int("max_depth", stats.MaxDepth),
	)
	
	return graph, nil
}

// createNode creates a graph node from a resource
func (b *Builder) createNode(resource schema.Resource) *Node {
	metadata := resource.GetMetadata()
	
	// Extract dependencies
	dependsOn := b.extractDependencies(resource)
	
	return &Node{
		ID:         metadata.Name,
		Kind:       resource.GetKind(),
		Resource:   resource,
		DependsOn:  dependsOn,
		InDegree:   0,
		Level:      0,
		Visited:    false,
		Processing: false,
	}
}

// extractDependencies extracts dependency names from a resource
func (b *Builder) extractDependencies(resource schema.Resource) []string {
	switch r := resource.(type) {
	case *schema.MicroService:
		deps := make([]string, len(r.Spec.DependsOn))
		copy(deps, r.Spec.DependsOn)
		
		// Also extract implicit dependencies from environment variables
		for _, env := range r.Spec.Environment {
			if env.ValueFrom != nil {
				deps = append(deps, env.ValueFrom.Component)
			}
		}
		
		return deps
		
	case *schema.RDS:
		if r.Spec.DependsOn != nil {
			deps := make([]string, len(r.Spec.DependsOn))
			copy(deps, r.Spec.DependsOn)
			return deps
		}
		
	case *schema.DynamoDB:
		if r.Spec.DependsOn != nil {
			deps := make([]string, len(r.Spec.DependsOn))
			copy(deps, r.Spec.DependsOn)
			return deps
		}
		
	case *schema.S3:
		if r.Spec.DependsOn != nil {
			deps := make([]string, len(r.Spec.DependsOn))
			copy(deps, r.Spec.DependsOn)
			return deps
		}
		
	case *schema.SQS:
		if r.Spec.DependsOn != nil {
			deps := make([]string, len(r.Spec.DependsOn))
			copy(deps, r.Spec.DependsOn)
			return deps
		}
		
	case *schema.SNS:
		if r.Spec.DependsOn != nil {
			deps := make([]string, len(r.Spec.DependsOn))
			copy(deps, r.Spec.DependsOn)
			return deps
		}
	}
	
	return []string{}
}

// addEdges adds edges to the graph based on resource dependencies
func (b *Builder) addEdges(graph *Graph, resource schema.Resource) error {
	metadata := resource.GetMetadata()
	fromID := metadata.Name
	
	// Get dependencies
	deps := b.extractDependencies(resource)
	
	// Add explicit dependency edges
	for _, depID := range deps {
		// Check if dependency exists
		if _, exists := graph.GetNode(depID); !exists {
			// For implicit deps from ValueFrom, we might not have them yet
			// Log a warning but don't fail
			b.logger.Warn("Dependency not found",
				zap.String("from", fromID),
				zap.String("to", depID),
			)
			continue
		}
		
		// Determine edge type
		edgeType := EdgeTypeExplicit
		
		// Check if this is an implicit dependency (from ValueFrom)
		if ms, ok := resource.(*schema.MicroService); ok {
			for _, env := range ms.Spec.Environment {
				if env.ValueFrom != nil && env.ValueFrom.Component == depID {
					edgeType = EdgeTypeImplicit
					break
				}
			}
		}
		
		// Add edge
		if err := graph.AddEdge(fromID, depID, edgeType); err != nil {
			return fmt.Errorf("failed to add edge %s -> %s: %w", fromID, depID, err)
		}
		
		b.logger.Debug("Added edge",
			zap.String("from", fromID),
			zap.String("to", depID),
			zap.String("type", string(edgeType)),
		)
	}
	
	return nil
}

// calculateLevels calculates deployment level for each node
// Level 0 = no dependencies, Level N = max(dependency levels) + 1
func (b *Builder) calculateLevels(graph *Graph) error {
	// Reset all levels
	for _, node := range graph.Nodes {
		node.Level = -1
	}
	
	var calculateLevel func(id string) (int, error)
	calculateLevel = func(id string) (int, error) {
		node := graph.Nodes[id]
		
		// Already calculated
		if node.Level >= 0 {
			return node.Level, nil
		}
		
		// No dependencies, level 0
		if node.InDegree == 0 {
			node.Level = 0
			return 0, nil
		}
		
		// Calculate based on dependencies
		maxDepLevel := 0
		deps, err := graph.GetDependencies(id)
		if err != nil {
			return 0, err
		}
		
		for _, dep := range deps {
			depLevel, err := calculateLevel(dep.ID)
			if err != nil {
				return 0, err
			}
			if depLevel > maxDepLevel {
				maxDepLevel = depLevel
			}
		}
		
		node.Level = maxDepLevel + 1
		return node.Level, nil
	}
	
	// Calculate level for each node
	for id := range graph.Nodes {
		if _, err := calculateLevel(id); err != nil {
			return err
		}
	}
	
	return nil
}

// BuildFromComponents builds a graph from a list of components
// This is useful for testing or when you have components without a full parse result
func (b *Builder) BuildFromComponents(stackName string, components []schema.Resource) (*Graph, error) {
	// Create a minimal parse result
	result := &parser.ParseResult{
		Stack: &schema.Stack{
			ResourceBase: schema.ResourceBase{
				Metadata: schema.Metadata{
					Name: stackName,
				},
			},
		},
		Services:   []*schema.Service{},
		Components: components,
	}
	
	return b.Build(result)
}

