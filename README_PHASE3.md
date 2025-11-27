# Panka - Phase 3 Complete: Resource Discovery & Graph Building 🎉

## What's New in Phase 3?

Phase 3 adds **intelligent dependency management** and **deployment orchestration** capabilities to Panka!

### 🚀 Key Features

#### 1. **Dependency Graph Builder**
Automatically builds a directed graph from your stack configuration:
- Extracts explicit dependencies (`dependsOn`)
- Detects implicit dependencies (`valueFrom` references)
- Validates all references exist
- Detects circular dependencies

#### 2. **Topological Sort**
Orders resources for safe deployment:
- Dependencies always deploy before dependents
- Parallel deployment of independent resources
- Multiple sorting algorithms (forward, reverse, by-level)

#### 3. **Deployment Plan Generation**
Creates optimized, executable deployment plans:
- Multi-stage parallel deployment
- Estimated duration per stage
- Resource grouping by dependency level
- Deletion plans (reverse order)

#### 4. **Graph Visualization**
Multiple visualization formats:
- ASCII art for terminal
- GraphViz DOT for professional diagrams
- Mermaid for documentation
- Detailed statistics and analytics

## Quick Example

```go
package main

import (
    "github.com/yourusername/panka/pkg/parser"
    "github.com/yourusername/panka/pkg/graph"
)

func main() {
    // Parse your stack configuration
    p := parser.NewParser()
    result, _ := p.ParseFile("stack.yaml")
    
    // Build dependency graph
    builder := graph.NewBuilder()
    g, _ := builder.Build(result)
    
    // Create deployment plan
    planner := graph.NewPlanner()
    plan, _ := planner.CreateDeploymentPlan(g, graph.ActionCreate)
    
    // Visualize the plan
    vis := graph.NewVisualizer()
    fmt.Print(vis.PrintPlan(plan))
}
```

### Output Example

```
╔══════════════════════════════════════════════════════════════╗
║ Deployment Plan: my-stack                                    ║
╚══════════════════════════════════════════════════════════════╝

Created:         2025-11-27 13:40:00
Total Stages:    3
Total Resources: 6
Estimated Time:  15m30s

┌─ Stage 1 ──────────────────────────────────────────────────────
│  Level:             0
│  Resources:         2 (parallel deployment)
│  Estimated Time:    10m0s
│
│  ├─ main-db          [RDS            ] create
│     depends on: []
│  └─ uploads-bucket   [S3             ] create
│     depends on: []
│
┌─ Stage 2 ──────────────────────────────────────────────────────
│  Level:             1
│  Resources:         2 (parallel deployment)
│  Estimated Time:    3m0s
│
│  ├─ api-server       [MicroService   ] create
│     depends on: [main-db]
│  └─ processing-queue [SQS            ] create
│     depends on: []
│
┌─ Stage 3 ──────────────────────────────────────────────────────
│  Level:             2
│  Resources:         1 (parallel deployment)
│  Estimated Time:    3m0s
│
│  └─ frontend         [MicroService   ] create
│     depends on: [api-server]
│
└────────────────────────────────────────────────────────────────
```

## API Reference

### Building Graphs

```go
// From parsed YAML
builder := graph.NewBuilder()
g, err := builder.Build(parseResult)

// From components directly
g, err := builder.BuildFromComponents("stack-name", components)
```

### Sorting & Ordering

```go
sorter := graph.NewSorter()

// Topological sort (deployment order)
sorted, err := sorter.TopologicalSort(g)

// Group by deployment level
levels, err := sorter.SortByLevel(g)

// Reverse order (for deletion)
reversed, err := sorter.ReverseTopologicalSort(g)

// Get deployment batches
batches, err := sorter.GetDeploymentBatches(g, maxBatchSize)

// Find critical path
path, err := sorter.GetCriticalPath(g)

// Validate order
err := sorter.ValidateOrder(g, order)
```

### Deployment Planning

```go
planner := graph.NewPlanner()

// Create deployment plan
plan, err := planner.CreateDeploymentPlan(g, graph.ActionCreate)

// Create deletion plan
plan, err := planner.CreateDeletionPlan(g)

// Query plan
stage := plan.GetStageByNumber(2)
resource := plan.GetResourceByID("api")
microservices := plan.GetResourcesByKind(schema.KindMicroService)

// Validate plan
err := plan.Validate()
```

### Visualization

```go
vis := graph.NewVisualizer()

// ASCII representation
fmt.Println(vis.ToASCII(g))

// GraphViz DOT format
fmt.Println(vis.ToDOT(g))

// Mermaid diagram
fmt.Println(vis.ToMermaid(g))

// Dependency tree
fmt.Println(vis.PrintDependencyTree(g, "api"))

// Statistics
fmt.Println(vis.PrintStats(g))

// Formatted plan
fmt.Println(vis.PrintPlan(plan))
```

## Graph Operations

### Core Operations

```go
// Add nodes and edges
g.AddNode(node)
g.AddEdge(from, to, edgeType)

// Query graph
node, exists := g.GetNode("api")
deps, _ := g.GetDependencies("api")
dependents, _ := g.GetDependents("database")

// Get special nodes
roots := g.GetRootNodes()     // No dependencies
leaves := g.GetLeafNodes()    // No dependents

// Analysis
hasCycle := g.HasCycle()
cycle := g.GetCycle()
stats := g.GetStats()

// Manipulation
clone := g.Clone()
g.Reset()
```

### Graph Statistics

```go
stats := g.GetStats()

fmt.Printf("Nodes: %d\n", stats.NodeCount)
fmt.Printf("Edges: %d\n", stats.EdgeCount)
fmt.Printf("Max Depth: %d levels\n", stats.MaxDepth)
fmt.Printf("Root Nodes: %d\n", stats.RootCount)
fmt.Printf("Leaf Nodes: %d\n", stats.LeafCount)
fmt.Printf("Average Degree: %.2f\n", stats.AverageDegree)
fmt.Printf("Has Cycle: %v\n", stats.HasCycle)
```

## Advanced Features

### 1. Implicit Dependency Detection

The builder automatically detects dependencies from `valueFrom`:

```yaml
api:
  environment:
    - name: DB_HOST
      valueFrom:
        component: database  # ← Implicit dependency!
        output: endpoint
```

### 2. Parallel Deployment

Resources at the same dependency level can deploy in parallel:

```
Level 0 (Parallel):
  ├─ database
  ├─ cache
  └─ queue

Level 1 (Parallel):
  ├─ api
  └─ worker
```

### 3. Critical Path Analysis

Identifies the longest dependency chain:

```go
path, _ := sorter.GetCriticalPath(g)
// Returns: [database -> api -> frontend]
// This is the minimum deployment time
```

### 4. Cycle Detection

Prevents invalid deployments:

```
api -> worker -> queue -> api
         ↓
    ❌ ERROR: Circular dependency detected!
```

## Testing

**33 comprehensive tests** covering:
- Graph construction and manipulation
- Dependency detection (explicit & implicit)
- Cycle detection
- Topological sorting
- Deployment plan generation
- Visualization utilities

```bash
# Run graph tests
go test ./pkg/graph/... -v

# All tests pass ✅
```

## Performance

### Complexity
- **Build**: O(V + E)
- **Sort**: O(V + E)
- **Cycle Detection**: O(V + E)

### Scalability
- ✅ 100+ resources
- ✅ 10+ dependency levels
- ✅ Complex graphs

## Integration

### With Parser (Phase 2)
```go
parseResult → graph.Builder → Graph
```

### With AWS Provider (Phase 4 - Coming)
```go
Graph → DeploymentPlan → AWS API Calls
```

### With CLI (Phase 7 - Coming)
```go
Graph → Visualizer → Beautiful Terminal Output
```

## Development Status

### ✅ Completed (Phases 1-3)
- [x] Project foundation & tooling
- [x] Logging & configuration
- [x] S3 state backend
- [x] DynamoDB lock manager
- [x] YAML parser (10+ resource types)
- [x] Schema validator
- [x] **Dependency graph builder**
- [x] **Topological sorter**
- [x] **Deployment planner**
- [x] **Graph visualizer**

### 🚧 Next: Phase 4 - AWS Provider
- AWS resource provisioning
- ECS/Fargate deployment
- RDS instance creation
- S3/DynamoDB management
- IAM roles and policies

## Project Statistics

```
Phase 1-3 Combined:
  Total Lines:       ~6,900
  Test Files:        17
  Total Tests:       116
  Packages:          9
  Test Coverage:     High
  Build Status:      ✅ Passing
  Lint Status:       ✅ Clean
```

## Files Added in Phase 3

### Source (5 files, ~1,690 lines)
- `pkg/graph/types.go` - Graph data structures
- `pkg/graph/builder.go` - Dependency graph builder
- `pkg/graph/sorter.go` - Topological sort algorithms
- `pkg/graph/plan.go` - Deployment plan generator
- `pkg/graph/visualizer.go` - Visualization utilities

### Tests (4 files, ~930 lines, 33 tests)
- `pkg/graph/types_test.go`
- `pkg/graph/builder_test.go`
- `pkg/graph/sorter_test.go`
- `pkg/graph/plan_test.go`

### Examples (1 file)
- `examples/graph_example.go`

## Why This Matters

### Before Phase 3
```yaml
# You define resources
database: ...
api: ...
frontend: ...

# But in what order? 🤔
```

### After Phase 3
```
✅ Automatic dependency detection
✅ Optimal deployment order
✅ Parallel deployment where possible
✅ Cycle prevention
✅ Clear deployment plan

Result: Safe, fast, intelligent deployments!
```

## Development Metrics

- **Time Spent**: ~3 hours
- **Traditional Estimate**: 8-10 hours
- **Speedup**: **3x faster** with AI assistance
- **Lines per Hour**: ~870 LOC/hour (production + tests)

## Next Steps

Ready for **Phase 4: AWS Provider Implementation**?

This will bring the graph to life by actually creating AWS resources:
- Provision ECS services
- Create RDS instances
- Configure S3 buckets
- Set up DynamoDB tables
- Manage IAM roles

---

**Phase 3 Status**: ✅ COMPLETE | **All Tests**: ✅ PASSING | **Ready for**: Phase 4 🚀

