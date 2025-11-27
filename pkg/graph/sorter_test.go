package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/panka/pkg/parser/schema"
)

func createTestGraph() *Graph {
	g := NewGraph("test")
	
	// Create a simple dependency graph:
	// db (level 0) <- api (level 1) <- frontend (level 2)
	// cache (level 0) <-/
	
	db := &Node{ID: "db", Kind: schema.KindRDS}
	cache := &Node{ID: "cache", Kind: schema.KindS3}
	api := &Node{ID: "api", Kind: schema.KindMicroService, DependsOn: []string{"db", "cache"}}
	frontend := &Node{ID: "frontend", Kind: schema.KindMicroService, DependsOn: []string{"api"}}
	
	g.AddNode(db)
	g.AddNode(cache)
	g.AddNode(api)
	g.AddNode(frontend)
	
	g.AddEdge("api", "db", EdgeTypeExplicit)
	g.AddEdge("api", "cache", EdgeTypeExplicit)
	g.AddEdge("frontend", "api", EdgeTypeExplicit)
	
	// Calculate levels
	builder := NewBuilder()
	builder.calculateLevels(g)
	
	return g
}

func TestSorter_TopologicalSort(t *testing.T) {
	g := createTestGraph()
	sorter := NewSorter()
	
	sorted, err := sorter.TopologicalSort(g)
	require.NoError(t, err)
	assert.Len(t, sorted, 4)
	
	// Verify order: dependencies come first
	positions := make(map[string]int)
	for i, node := range sorted {
		positions[node.ID] = i
	}
	
	// db and cache should come before api
	assert.Less(t, positions["db"], positions["api"])
	assert.Less(t, positions["cache"], positions["api"])
	
	// api should come before frontend
	assert.Less(t, positions["api"], positions["frontend"])
}

func TestSorter_TopologicalSort_WithCycle(t *testing.T) {
	g := NewGraph("test")
	
	// Create a cycle
	a := &Node{ID: "a", Kind: schema.KindMicroService}
	b := &Node{ID: "b", Kind: schema.KindMicroService}
	
	g.AddNode(a)
	g.AddNode(b)
	g.AddEdge("a", "b", EdgeTypeExplicit)
	g.AddEdge("b", "a", EdgeTypeExplicit)
	
	sorter := NewSorter()
	_, err := sorter.TopologicalSort(g)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestSorter_SortByLevel(t *testing.T) {
	g := createTestGraph()
	sorter := NewSorter()
	
	levels, err := sorter.SortByLevel(g)
	require.NoError(t, err)
	assert.Len(t, levels, 3) // 3 levels (0, 1, 2)
	
	// Level 0: db, cache
	assert.Len(t, levels[0], 2)
	
	// Level 1: api
	assert.Len(t, levels[1], 1)
	assert.Equal(t, "api", levels[1][0].ID)
	
	// Level 2: frontend
	assert.Len(t, levels[2], 1)
	assert.Equal(t, "frontend", levels[2][0].ID)
}

func TestSorter_ReverseTopologicalSort(t *testing.T) {
	g := createTestGraph()
	sorter := NewSorter()
	
	forward, _ := sorter.TopologicalSort(g)
	reverse, err := sorter.ReverseTopologicalSort(g)
	
	require.NoError(t, err)
	assert.Len(t, reverse, len(forward))
	
	// Verify reversal
	for i := 0; i < len(forward); i++ {
		assert.Equal(t, forward[i].ID, reverse[len(reverse)-1-i].ID)
	}
}

func TestSorter_GetDeploymentBatches(t *testing.T) {
	g := createTestGraph()
	sorter := NewSorter()
	
	// No batch size limit
	batches, err := sorter.GetDeploymentBatches(g, 0)
	require.NoError(t, err)
	assert.Len(t, batches, 3) // Same as levels
	
	// With batch size limit
	batches, err = sorter.GetDeploymentBatches(g, 1)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(batches), 3)
}

func TestSorter_ValidateOrder(t *testing.T) {
	g := createTestGraph()
	sorter := NewSorter()
	
	// Valid order
	validOrder := []*Node{g.Nodes["db"], g.Nodes["cache"], g.Nodes["api"], g.Nodes["frontend"]}
	err := sorter.ValidateOrder(g, validOrder)
	assert.NoError(t, err)
	
	// Invalid order (api before db)
	invalidOrder := []*Node{g.Nodes["api"], g.Nodes["db"], g.Nodes["cache"], g.Nodes["frontend"]}
	err = sorter.ValidateOrder(g, invalidOrder)
	assert.Error(t, err)
}

func TestSorter_GetCriticalPath(t *testing.T) {
	g := createTestGraph()
	sorter := NewSorter()
	
	path, err := sorter.GetCriticalPath(g)
	require.NoError(t, err)
	
	// Critical path should be: db/cache -> api -> frontend (length 3)
	assert.GreaterOrEqual(t, len(path), 2)
	
	// Last node should be frontend (highest level)
	assert.Equal(t, "frontend", path[len(path)-1].ID)
}

