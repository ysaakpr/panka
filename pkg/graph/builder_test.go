package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/panka/pkg/parser"
	"github.com/yourusername/panka/pkg/parser/schema"
)

func TestBuilder_Build_EmptyStack(t *testing.T) {
	builder := NewBuilder()
	
	result := &parser.ParseResult{
		Stack:      schema.NewStack("test-stack"),
		Services:   []*schema.Service{},
		Components: []schema.Resource{},
	}
	
	g, err := builder.Build(result)
	require.NoError(t, err)
	assert.NotNil(t, g)
	assert.True(t, g.IsEmpty())
	assert.Equal(t, "test-stack", g.StackName)
}

func TestBuilder_Build_SimpleGraph(t *testing.T) {
	builder := NewBuilder()
	
	// Create components
	db := schema.NewRDS("db", "backend", "test-stack")
	api := schema.NewMicroService("api", "backend", "test-stack")
	api.Spec.DependsOn = []string{"db"}
	api.Spec.Image.Repository = "myrepo/api"
	api.Spec.Image.Tag = "v1.0.0"
	
	result := &parser.ParseResult{
		Stack:      schema.NewStack("test-stack"),
		Services:   []*schema.Service{schema.NewService("backend", "test-stack")},
		Components: []schema.Resource{db, api},
	}
	
	g, err := builder.Build(result)
	require.NoError(t, err)
	assert.NotNil(t, g)
	assert.Equal(t, 2, g.NodeCount())
	assert.Equal(t, 1, g.EdgeCount())
	
	// Verify nodes
	dbNode, exists := g.GetNode("db")
	assert.True(t, exists)
	assert.Equal(t, schema.KindRDS, dbNode.Kind)
	assert.Equal(t, 0, dbNode.Level)
	
	apiNode, exists := g.GetNode("api")
	assert.True(t, exists)
	assert.Equal(t, schema.KindMicroService, apiNode.Kind)
	assert.Equal(t, 1, apiNode.Level)
	
	// Verify edge
	deps, _ := g.GetDependencies("api")
	assert.Len(t, deps, 1)
	assert.Equal(t, "db", deps[0].ID)
}

func TestBuilder_Build_ComplexGraph(t *testing.T) {
	builder := NewBuilder()
	
	// Create components
	db := schema.NewRDS("db", "backend", "test-stack")
	cache := schema.NewS3("cache", "backend", "test-stack")
	queue := schema.NewSQS("queue", "backend", "test-stack")
	
	api := schema.NewMicroService("api", "backend", "test-stack")
	api.Spec.DependsOn = []string{"db", "cache"}
	api.Spec.Image.Repository = "myrepo/api"
	api.Spec.Image.Tag = "v1.0.0"
	
	worker := schema.NewMicroService("worker", "backend", "test-stack")
	worker.Spec.DependsOn = []string{"db", "queue"}
	worker.Spec.Image.Repository = "myrepo/worker"
	worker.Spec.Image.Tag = "v1.0.0"
	
	result := &parser.ParseResult{
		Stack:      schema.NewStack("test-stack"),
		Services:   []*schema.Service{schema.NewService("backend", "test-stack")},
		Components: []schema.Resource{db, cache, queue, api, worker},
	}
	
	g, err := builder.Build(result)
	require.NoError(t, err)
	assert.Equal(t, 5, g.NodeCount())
	assert.Equal(t, 4, g.EdgeCount())
	
	// Verify levels
	assert.Equal(t, 0, g.Nodes["db"].Level)
	assert.Equal(t, 0, g.Nodes["cache"].Level)
	assert.Equal(t, 0, g.Nodes["queue"].Level)
	assert.Equal(t, 1, g.Nodes["api"].Level)
	assert.Equal(t, 1, g.Nodes["worker"].Level)
	
	// Verify root nodes
	roots := g.GetRootNodes()
	assert.Len(t, roots, 3)
}

func TestBuilder_Build_WithImplicitDependencies(t *testing.T) {
	builder := NewBuilder()
	
	// Create components
	db := schema.NewRDS("db", "backend", "test-stack")
	
	api := schema.NewMicroService("api", "backend", "test-stack")
	api.Spec.Image.Repository = "myrepo/api"
	api.Spec.Image.Tag = "v1.0.0"
	// Implicit dependency via environment variable
	api.Spec.Environment = []schema.EnvironmentVariable{
		{
			Name: "DB_HOST",
			ValueFrom: &schema.ValueFrom{
				Component: "db",
				Output:    "endpoint",
			},
		},
	}
	
	result := &parser.ParseResult{
		Stack:      schema.NewStack("test-stack"),
		Services:   []*schema.Service{schema.NewService("backend", "test-stack")},
		Components: []schema.Resource{db, api},
	}
	
	g, err := builder.Build(result)
	require.NoError(t, err)
	assert.Equal(t, 2, g.NodeCount())
	assert.Equal(t, 1, g.EdgeCount())
	
	// Verify edge type
	edges := g.Edges["api"]
	require.Len(t, edges, 1)
	assert.Equal(t, EdgeTypeImplicit, edges[0].Type)
}

func TestBuilder_Build_CircularDependency(t *testing.T) {
	builder := NewBuilder()
	
	// Create circular dependency
	api := schema.NewMicroService("api", "backend", "test-stack")
	api.Spec.DependsOn = []string{"worker"}
	api.Spec.Image.Repository = "myrepo/api"
	api.Spec.Image.Tag = "v1.0.0"
	
	worker := schema.NewMicroService("worker", "backend", "test-stack")
	worker.Spec.DependsOn = []string{"api"}
	worker.Spec.Image.Repository = "myrepo/worker"
	worker.Spec.Image.Tag = "v1.0.0"
	
	result := &parser.ParseResult{
		Stack:      schema.NewStack("test-stack"),
		Services:   []*schema.Service{schema.NewService("backend", "test-stack")},
		Components: []schema.Resource{api, worker},
	}
	
	_, err := builder.Build(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestBuilder_Build_MultiLevel(t *testing.T) {
	builder := NewBuilder()
	
	// Create multi-level dependencies
	db := schema.NewRDS("db", "backend", "test-stack")
	
	cache := schema.NewS3("cache", "backend", "test-stack")
	cache.Spec.DependsOn = []string{"db"}
	
	api := schema.NewMicroService("api", "backend", "test-stack")
	api.Spec.DependsOn = []string{"cache"}
	api.Spec.Image.Repository = "myrepo/api"
	api.Spec.Image.Tag = "v1.0.0"
	
	result := &parser.ParseResult{
		Stack:      schema.NewStack("test-stack"),
		Services:   []*schema.Service{schema.NewService("backend", "test-stack")},
		Components: []schema.Resource{db, cache, api},
	}
	
	g, err := builder.Build(result)
	require.NoError(t, err)
	
	// Verify levels
	assert.Equal(t, 0, g.Nodes["db"].Level)
	assert.Equal(t, 1, g.Nodes["cache"].Level)
	assert.Equal(t, 2, g.Nodes["api"].Level)
	
	stats := g.GetStats()
	assert.Equal(t, 2, stats.MaxDepth)
}

func TestBuilder_BuildFromComponents(t *testing.T) {
	builder := NewBuilder()
	
	db := schema.NewRDS("db", "backend", "test-stack")
	api := schema.NewMicroService("api", "backend", "test-stack")
	api.Spec.DependsOn = []string{"db"}
	api.Spec.Image.Repository = "myrepo/api"
	api.Spec.Image.Tag = "v1.0.0"
	
	components := []schema.Resource{db, api}
	
	g, err := builder.BuildFromComponents("test-stack", components)
	require.NoError(t, err)
	assert.Equal(t, 2, g.NodeCount())
	assert.Equal(t, "test-stack", g.StackName)
}

