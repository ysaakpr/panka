package graph

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanner_CreateDeploymentPlan(t *testing.T) {
	g := createTestGraph()
	planner := NewPlanner()
	
	plan, err := planner.CreateDeploymentPlan(g, ActionCreate)
	require.NoError(t, err)
	assert.NotNil(t, plan)
	
	// Verify plan metadata
	assert.Equal(t, "test", plan.StackName)
	assert.Equal(t, 4, plan.TotalResources)
	assert.Equal(t, 3, plan.TotalStages)
	assert.Greater(t, plan.EstimatedTime, time.Duration(0))
	
	// Verify stages
	assert.Len(t, plan.Stages, 3)
	
	// Stage 1: db, cache (level 0)
	assert.Equal(t, 1, plan.Stages[0].Number)
	assert.Len(t, plan.Stages[0].Resources, 2)
	
	// Stage 2: api (level 1)
	assert.Equal(t, 2, plan.Stages[1].Number)
	assert.Len(t, plan.Stages[1].Resources, 1)
	assert.Equal(t, "api", plan.Stages[1].Resources[0].ID)
	
	// Stage 3: frontend (level 2)
	assert.Equal(t, 3, plan.Stages[2].Number)
	assert.Len(t, plan.Stages[2].Resources, 1)
	assert.Equal(t, "frontend", plan.Stages[2].Resources[0].ID)
}

func TestPlanner_CreateDeletionPlan(t *testing.T) {
	g := createTestGraph()
	planner := NewPlanner()
	
	plan, err := planner.CreateDeletionPlan(g)
	require.NoError(t, err)
	assert.NotNil(t, plan)
	
	// Verify all resources have delete action
	for _, stage := range plan.Stages {
		for _, resource := range stage.Resources {
			assert.Equal(t, ActionDelete, resource.Action)
		}
	}
	
	// Deletion order should be reverse: frontend first, then api, then db/cache
	firstStage := plan.Stages[0]
	assert.Contains(t, []string{"frontend"}, firstStage.Resources[0].ID)
}

func TestPlanner_EmptyGraph(t *testing.T) {
	g := NewGraph("empty")
	planner := NewPlanner()
	
	plan, err := planner.CreateDeploymentPlan(g, ActionCreate)
	require.NoError(t, err)
	assert.True(t, plan.IsEmpty())
	assert.Len(t, plan.Stages, 0)
	assert.Equal(t, 0, plan.TotalResources)
}

func TestDeploymentPlan_GetStageByNumber(t *testing.T) {
	g := createTestGraph()
	planner := NewPlanner()
	plan, _ := planner.CreateDeploymentPlan(g, ActionCreate)
	
	stage := plan.GetStageByNumber(2)
	assert.NotNil(t, stage)
	assert.Equal(t, 2, stage.Number)
	
	stage = plan.GetStageByNumber(999)
	assert.Nil(t, stage)
}

func TestDeploymentPlan_GetResourceByID(t *testing.T) {
	g := createTestGraph()
	planner := NewPlanner()
	plan, _ := planner.CreateDeploymentPlan(g, ActionCreate)
	
	resource := plan.GetResourceByID("api")
	assert.NotNil(t, resource)
	assert.Equal(t, "api", resource.ID)
	
	resource = plan.GetResourceByID("non-existent")
	assert.Nil(t, resource)
}

func TestDeploymentPlan_GetResourcesByKind(t *testing.T) {
	g := createTestGraph()
	planner := NewPlanner()
	plan, _ := planner.CreateDeploymentPlan(g, ActionCreate)
	
	microservices := plan.GetResourcesByKind("MicroService")
	assert.Len(t, microservices, 2) // api and frontend
	
	databases := plan.GetResourcesByKind("RDS")
	assert.Len(t, databases, 1) // db
}

func TestDeploymentPlan_Summary(t *testing.T) {
	g := createTestGraph()
	planner := NewPlanner()
	plan, _ := planner.CreateDeploymentPlan(g, ActionCreate)
	
	summary := plan.Summary()
	assert.NotEmpty(t, summary)
	assert.Contains(t, summary, "Deployment Plan")
	assert.Contains(t, summary, "test")
	assert.Contains(t, summary, "Stage 1")
}

func TestDeploymentPlan_Validate(t *testing.T) {
	g := createTestGraph()
	planner := NewPlanner()
	plan, _ := planner.CreateDeploymentPlan(g, ActionCreate)
	
	err := plan.Validate()
	assert.NoError(t, err)
	
	// Corrupt the plan
	plan.TotalResources = 999
	err = plan.Validate()
	assert.Error(t, err)
}

