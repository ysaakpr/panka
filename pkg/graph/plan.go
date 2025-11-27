package graph

import (
	"fmt"
	"time"

	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/parser/schema"
	"go.uber.org/zap"
)

// DeploymentPlan represents a plan for deploying resources
type DeploymentPlan struct {
	// Metadata
	StackName string
	CreatedAt time.Time
	
	// Deployment stages
	Stages []*DeploymentStage
	
	// Summary
	TotalResources int
	TotalStages    int
	EstimatedTime  time.Duration
	
	// Graph reference
	Graph *Graph
}

// DeploymentStage represents a stage in the deployment plan
// All resources in a stage can be deployed in parallel
type DeploymentStage struct {
	// Stage information
	Number      int
	Level       int
	Description string
	
	// Resources to deploy
	Resources []*DeploymentResource
	
	// Timing
	EstimatedDuration time.Duration
}

// DeploymentResource represents a resource to be deployed
type DeploymentResource struct {
	ID          string
	Kind        schema.Kind
	Resource    schema.Resource
	Dependencies []string
	
	// Action to perform
	Action ResourceAction
}

// ResourceAction defines the action to perform on a resource
type ResourceAction string

const (
	// ActionCreate creates a new resource
	ActionCreate ResourceAction = "create"
	
	// ActionUpdate updates an existing resource
	ActionUpdate ResourceAction = "update"
	
	// ActionDelete deletes a resource
	ActionDelete ResourceAction = "delete"
	
	// ActionNone does nothing (resource unchanged)
	ActionNone ResourceAction = "none"
)

// Planner generates deployment plans from graphs
type Planner struct {
	logger *logger.Logger
	sorter *Sorter
}

// NewPlanner creates a new deployment planner
func NewPlanner() *Planner {
	log, _ := logger.NewDevelopment()
	return &Planner{
		logger: log,
		sorter: NewSorter(),
	}
}

// CreateDeploymentPlan creates a deployment plan from a graph
func (p *Planner) CreateDeploymentPlan(g *Graph, action ResourceAction) (*DeploymentPlan, error) {
	if g.IsEmpty() {
		return &DeploymentPlan{
			StackName:      g.StackName,
			CreatedAt:      time.Now(),
			Stages:         []*DeploymentStage{},
			TotalResources: 0,
			TotalStages:    0,
			Graph:          g,
		}, nil
	}
	
	p.logger.Info("Creating deployment plan",
		zap.String("stack", g.StackName),
		zap.String("action", string(action)),
	)
	
	// Get sorted levels
	var levels [][]*Node
	var err error
	
	if action == ActionDelete {
		// For deletion, reverse the order
		sorted, err := p.sorter.ReverseTopologicalSort(g)
		if err != nil {
			return nil, err
		}
		
		// Group by level (in reverse)
		levelMap := make(map[int][]*Node)
		for _, node := range sorted {
			levelMap[node.Level] = append(levelMap[node.Level], node)
		}
		
		// Convert to slice
		maxLevel := 0
		for level := range levelMap {
			if level > maxLevel {
				maxLevel = level
			}
		}
		
		levels = make([][]*Node, maxLevel+1)
		for i := range levels {
			levels[maxLevel-i] = levelMap[i]
		}
	} else {
		// For create/update, use normal order
		levels, err = p.sorter.SortByLevel(g)
		if err != nil {
			return nil, err
		}
	}
	
	// Create stages
	stages := make([]*DeploymentStage, 0, len(levels))
	totalResources := 0
	
	for i, level := range levels {
		if len(level) == 0 {
			continue
		}
		
		stage := &DeploymentStage{
			Number:      i + 1,
			Level:       level[0].Level,
			Description: fmt.Sprintf("Deploy level %d resources", level[0].Level),
			Resources:   make([]*DeploymentResource, 0, len(level)),
			EstimatedDuration: p.estimateStageDuration(level, action),
		}
		
		// Add resources to stage
		for _, node := range level {
			resource := &DeploymentResource{
				ID:           node.ID,
				Kind:         node.Kind,
				Resource:     node.Resource,
				Dependencies: node.DependsOn,
				Action:       action,
			}
			
			stage.Resources = append(stage.Resources, resource)
			totalResources++
		}
		
		stages = append(stages, stage)
		
		p.logger.Debug("Created stage",
			zap.Int("stage", stage.Number),
			zap.Int("resources", len(stage.Resources)),
			zap.Duration("estimated_duration", stage.EstimatedDuration),
		)
	}
	
	// Calculate total estimated time
	totalTime := time.Duration(0)
	for _, stage := range stages {
		totalTime += stage.EstimatedDuration
	}
	
	plan := &DeploymentPlan{
		StackName:      g.StackName,
		CreatedAt:      time.Now(),
		Stages:         stages,
		TotalResources: totalResources,
		TotalStages:    len(stages),
		EstimatedTime:  totalTime,
		Graph:          g,
	}
	
	p.logger.Info("Deployment plan created",
		zap.Int("stages", plan.TotalStages),
		zap.Int("resources", plan.TotalResources),
		zap.Duration("estimated_time", plan.EstimatedTime),
	)
	
	return plan, nil
}

// CreateDeletionPlan creates a plan for deleting resources (reverse order)
func (p *Planner) CreateDeletionPlan(g *Graph) (*DeploymentPlan, error) {
	return p.CreateDeploymentPlan(g, ActionDelete)
}

// estimateStageDuration estimates how long a stage will take
// This is a rough estimate based on resource types
func (p *Planner) estimateStageDuration(nodes []*Node, action ResourceAction) time.Duration {
	// Base duration per resource type (rough estimates)
	durations := map[schema.Kind]time.Duration{
		schema.KindS3:            30 * time.Second,
		schema.KindDynamoDB:      45 * time.Second,
		schema.KindSQS:           20 * time.Second,
		schema.KindSNS:           20 * time.Second,
		schema.KindRDS:           10 * time.Minute, // RDS takes much longer
		schema.KindMicroService:  3 * time.Minute,  // ECS deployment
		schema.KindComponentInfra: 2 * time.Minute,
	}
	
	// Find the longest duration in this stage
	// Since resources in the same stage deploy in parallel,
	// the stage duration is the maximum of all resource durations
	maxDuration := 30 * time.Second // Default minimum
	
	for _, node := range nodes {
		if duration, exists := durations[node.Kind]; exists {
			if duration > maxDuration {
				maxDuration = duration
			}
		}
	}
	
	// Deletion is usually faster
	if action == ActionDelete {
		maxDuration = maxDuration / 2
	}
	
	return maxDuration
}

// GetStageByNumber returns a stage by its number
func (p *DeploymentPlan) GetStageByNumber(number int) *DeploymentStage {
	for _, stage := range p.Stages {
		if stage.Number == number {
			return stage
		}
	}
	return nil
}

// GetResourceByID returns a deployment resource by its ID
func (p *DeploymentPlan) GetResourceByID(id string) *DeploymentResource {
	for _, stage := range p.Stages {
		for _, resource := range stage.Resources {
			if resource.ID == id {
				return resource
			}
		}
	}
	return nil
}

// GetResourcesByKind returns all resources of a specific kind
func (p *DeploymentPlan) GetResourcesByKind(kind schema.Kind) []*DeploymentResource {
	resources := make([]*DeploymentResource, 0)
	for _, stage := range p.Stages {
		for _, resource := range stage.Resources {
			if resource.Kind == kind {
				resources = append(resources, resource)
			}
		}
	}
	return resources
}

// Summary returns a human-readable summary of the plan
func (p *DeploymentPlan) Summary() string {
	summary := fmt.Sprintf("Deployment Plan for %s\n", p.StackName)
	summary += fmt.Sprintf("Created: %s\n", p.CreatedAt.Format(time.RFC3339))
	summary += fmt.Sprintf("Total Stages: %d\n", p.TotalStages)
	summary += fmt.Sprintf("Total Resources: %d\n", p.TotalResources)
	summary += fmt.Sprintf("Estimated Time: %s\n\n", p.EstimatedTime)
	
	for _, stage := range p.Stages {
		summary += fmt.Sprintf("Stage %d: %s\n", stage.Number, stage.Description)
		summary += fmt.Sprintf("  Resources: %d (parallel deployment)\n", len(stage.Resources))
		summary += fmt.Sprintf("  Estimated Duration: %s\n", stage.EstimatedDuration)
		
		for _, resource := range stage.Resources {
			summary += fmt.Sprintf("    - %s (%s) [%s]\n", 
				resource.ID, resource.Kind, resource.Action)
		}
		summary += "\n"
	}
	
	return summary
}

// Validate validates the deployment plan
func (p *DeploymentPlan) Validate() error {
	if p.TotalStages != len(p.Stages) {
		return fmt.Errorf("total stages mismatch: expected %d, got %d", 
			p.TotalStages, len(p.Stages))
	}
	
	resourceCount := 0
	for _, stage := range p.Stages {
		resourceCount += len(stage.Resources)
	}
	
	if p.TotalResources != resourceCount {
		return fmt.Errorf("total resources mismatch: expected %d, got %d",
			p.TotalResources, resourceCount)
	}
	
	// Validate stage numbers are sequential
	for i, stage := range p.Stages {
		if stage.Number != i+1 {
			return fmt.Errorf("stage numbers not sequential: expected %d, got %d",
				i+1, stage.Number)
		}
	}
	
	return nil
}

// IsEmpty returns true if the plan has no stages
func (p *DeploymentPlan) IsEmpty() bool {
	return len(p.Stages) == 0
}

