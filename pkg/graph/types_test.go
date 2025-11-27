package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/panka/pkg/parser/schema"
)

func TestNewGraph(t *testing.T) {
	g := NewGraph("test-stack")
	
	assert.Equal(t, "test-stack", g.StackName)
	assert.NotNil(t, g.Nodes)
	assert.NotNil(t, g.Edges)
	assert.NotNil(t, g.ReverseEdges)
	assert.True(t, g.IsEmpty())
}

func TestGraph_AddNode(t *testing.T) {
	g := NewGraph("test")
	
	node := &Node{
		ID:   "node1",
		Kind: schema.KindS3,
	}
	
	err := g.AddNode(node)
	assert.NoError(t, err)
	assert.Equal(t, 1, g.NodeCount())
	
	// Try adding duplicate
	err = g.AddNode(node)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestGraph_AddEdge(t *testing.T) {
	g := NewGraph("test")
	
	node1 := &Node{ID: "node1", Kind: schema.KindS3}
	node2 := &Node{ID: "node2", Kind: schema.KindMicroService}
	
	g.AddNode(node1)
	g.AddNode(node2)
	
	// node2 depends on node1
	err := g.AddEdge("node2", "node1", EdgeTypeExplicit)
	assert.NoError(t, err)
	assert.Equal(t, 1, g.EdgeCount())
	assert.Equal(t, 1, node2.InDegree) // node2 has 1 dependency
	assert.Equal(t, 0, node1.InDegree) // node1 has no dependencies
	
	// Try adding edge with non-existent node
	err = g.AddEdge("node3", "node1", EdgeTypeExplicit)
	assert.Error(t, err)
}

func TestGraph_GetNode(t *testing.T) {
	g := NewGraph("test")
	
	node := &Node{ID: "node1", Kind: schema.KindS3}
	g.AddNode(node)
	
	retrieved, exists := g.GetNode("node1")
	assert.True(t, exists)
	assert.Equal(t, "node1", retrieved.ID)
	
	_, exists = g.GetNode("non-existent")
	assert.False(t, exists)
}

func TestGraph_GetDependencies(t *testing.T) {
	g := NewGraph("test")
	
	node1 := &Node{ID: "db", Kind: schema.KindRDS}
	node2 := &Node{ID: "cache", Kind: schema.KindS3}
	node3 := &Node{ID: "api", Kind: schema.KindMicroService}
	
	g.AddNode(node1)
	g.AddNode(node2)
	g.AddNode(node3)
	
	// api depends on db and cache
	g.AddEdge("api", "db", EdgeTypeExplicit)
	g.AddEdge("api", "cache", EdgeTypeExplicit)
	
	deps, err := g.GetDependencies("api")
	require.NoError(t, err)
	assert.Len(t, deps, 2)
	
	// db has no dependencies
	deps, err = g.GetDependencies("db")
	require.NoError(t, err)
	assert.Len(t, deps, 0)
}

func TestGraph_GetDependents(t *testing.T) {
	g := NewGraph("test")
	
	node1 := &Node{ID: "db", Kind: schema.KindRDS}
	node2 := &Node{ID: "api", Kind: schema.KindMicroService}
	node3 := &Node{ID: "worker", Kind: schema.KindMicroService}
	
	g.AddNode(node1)
	g.AddNode(node2)
	g.AddNode(node3)
	
	// Both api and worker depend on db
	g.AddEdge("api", "db", EdgeTypeExplicit)
	g.AddEdge("worker", "db", EdgeTypeExplicit)
	
	dependents, err := g.GetDependents("db")
	require.NoError(t, err)
	assert.Len(t, dependents, 2)
}

func TestGraph_GetRootNodes(t *testing.T) {
	g := NewGraph("test")
	
	root1 := &Node{ID: "s3", Kind: schema.KindS3}
	root2 := &Node{ID: "db", Kind: schema.KindRDS}
	dependent := &Node{ID: "api", Kind: schema.KindMicroService}
	
	g.AddNode(root1)
	g.AddNode(root2)
	g.AddNode(dependent)
	
	g.AddEdge("api", "db", EdgeTypeExplicit)
	
	roots := g.GetRootNodes()
	assert.Len(t, roots, 2)
}

func TestGraph_GetLeafNodes(t *testing.T) {
	g := NewGraph("test")
	
	root := &Node{ID: "db", Kind: schema.KindRDS}
	leaf1 := &Node{ID: "api", Kind: schema.KindMicroService}
	leaf2 := &Node{ID: "worker", Kind: schema.KindMicroService}
	
	g.AddNode(root)
	g.AddNode(leaf1)
	g.AddNode(leaf2)
	
	// api and worker depend on db
	// So api and worker are leaf nodes (nothing depends on them)
	g.AddEdge("api", "db", EdgeTypeExplicit)
	g.AddEdge("worker", "db", EdgeTypeExplicit)
	
	leaves := g.GetLeafNodes()
	// db is the root (no outgoing edges = no dependencies)
	// api and worker are leaves (nothing depends on them = no outgoing edges from them in reverse)
	// GetLeafNodes checks for nodes with no outgoing edges (nothing depends on them)
	assert.Len(t, leaves, 1) // Only db has no outgoing edges
}

func TestGraph_Clone(t *testing.T) {
	g := NewGraph("test")
	
	node1 := &Node{ID: "node1", Kind: schema.KindS3}
	node2 := &Node{ID: "node2", Kind: schema.KindMicroService}
	
	g.AddNode(node1)
	g.AddNode(node2)
	g.AddEdge("node2", "node1", EdgeTypeExplicit)
	
	clone := g.Clone()
	
	assert.Equal(t, g.NodeCount(), clone.NodeCount())
	assert.Equal(t, g.EdgeCount(), clone.EdgeCount())
	assert.Equal(t, g.StackName, clone.StackName)
	
	// Modify clone shouldn't affect original
	node3 := &Node{ID: "node3", Kind: schema.KindRDS}
	clone.AddNode(node3)
	
	assert.Equal(t, 2, g.NodeCount())
	assert.Equal(t, 3, clone.NodeCount())
}

func TestGraph_HasCycle_NoCycle(t *testing.T) {
	g := NewGraph("test")
	
	node1 := &Node{ID: "db", Kind: schema.KindRDS}
	node2 := &Node{ID: "cache", Kind: schema.KindS3}
	node3 := &Node{ID: "api", Kind: schema.KindMicroService}
	
	g.AddNode(node1)
	g.AddNode(node2)
	g.AddNode(node3)
	
	// Linear dependency: api -> cache -> db
	g.AddEdge("api", "cache", EdgeTypeExplicit)
	g.AddEdge("cache", "db", EdgeTypeExplicit)
	
	assert.False(t, g.HasCycle())
}

func TestGraph_HasCycle_WithCycle(t *testing.T) {
	g := NewGraph("test")
	
	node1 := &Node{ID: "a", Kind: schema.KindMicroService}
	node2 := &Node{ID: "b", Kind: schema.KindMicroService}
	node3 := &Node{ID: "c", Kind: schema.KindMicroService}
	
	g.AddNode(node1)
	g.AddNode(node2)
	g.AddNode(node3)
	
	// Create cycle: a -> b -> c -> a
	g.AddEdge("a", "b", EdgeTypeExplicit)
	g.AddEdge("b", "c", EdgeTypeExplicit)
	g.AddEdge("c", "a", EdgeTypeExplicit)
	
	assert.True(t, g.HasCycle())
	
	cycle := g.GetCycle()
	assert.NotNil(t, cycle)
	assert.Greater(t, len(cycle), 0)
}

func TestGraph_GetStats(t *testing.T) {
	g := NewGraph("test")
	
	node1 := &Node{ID: "db", Kind: schema.KindRDS, Level: 0}
	node2 := &Node{ID: "cache", Kind: schema.KindS3, Level: 0}
	node3 := &Node{ID: "api", Kind: schema.KindMicroService, Level: 1}
	
	g.AddNode(node1)
	g.AddNode(node2)
	g.AddNode(node3)
	
	// api depends on db and cache
	g.AddEdge("api", "db", EdgeTypeExplicit)
	g.AddEdge("api", "cache", EdgeTypeExplicit)
	
	stats := g.GetStats()
	
	assert.Equal(t, 3, stats.NodeCount)
	assert.Equal(t, 2, stats.EdgeCount)
	assert.Equal(t, 2, stats.RootCount) // db and cache have no dependencies (InDegree 0)
	assert.Equal(t, 2, stats.LeafCount) // db and cache have no outgoing edges
	assert.Equal(t, 1, stats.MaxDepth)
	assert.False(t, stats.HasCycle)
	assert.Greater(t, stats.AverageDegree, 0.0)
}

