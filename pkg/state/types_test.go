package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewState(t *testing.T) {
	state := NewState("test-stack", "production")
	
	require.NotNil(t, state)
	assert.Equal(t, "1.0", state.Version)
	assert.Equal(t, "test-stack", state.Metadata.Stack)
	assert.Equal(t, "production", state.Metadata.Environment)
	assert.NotNil(t, state.Resources)
	assert.NotNil(t, state.Outputs)
	assert.False(t, state.LastUpdate.IsZero())
}

func TestStateAddResource(t *testing.T) {
	state := NewState("test-stack", "dev")
	
	resource := &Resource{
		ID:       "res-001",
		Type:     "aws_s3_bucket",
		Name:     "test-bucket",
		Provider: "aws",
		Status:   ResourceStatusReady,
	}
	
	state.AddResource("bucket-1", resource)
	
	assert.Equal(t, 1, len(state.Resources))
	assert.NotNil(t, state.Resources["bucket-1"])
	assert.Equal(t, "res-001", state.Resources["bucket-1"].ID)
	assert.False(t, state.Resources["bucket-1"].CreatedAt.IsZero())
	assert.False(t, state.Resources["bucket-1"].UpdatedAt.IsZero())
}

func TestStateRemoveResource(t *testing.T) {
	state := NewState("test-stack", "dev")
	
	resource := &Resource{
		ID:   "res-001",
		Type: "aws_s3_bucket",
		Name: "test-bucket",
	}
	
	state.AddResource("bucket-1", resource)
	assert.Equal(t, 1, len(state.Resources))
	
	state.RemoveResource("bucket-1")
	assert.Equal(t, 0, len(state.Resources))
}

func TestStateGetResource(t *testing.T) {
	state := NewState("test-stack", "dev")
	
	resource := &Resource{
		ID:   "res-001",
		Type: "aws_s3_bucket",
		Name: "test-bucket",
	}
	
	state.AddResource("bucket-1", resource)
	
	// Get existing resource
	got, ok := state.GetResource("bucket-1")
	assert.True(t, ok)
	assert.NotNil(t, got)
	assert.Equal(t, "res-001", got.ID)
	
	// Get non-existent resource
	got, ok = state.GetResource("non-existent")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestStateSetOutput(t *testing.T) {
	state := NewState("test-stack", "dev")
	
	state.SetOutput("api_url", "https://api.example.com")
	state.SetOutput("db_endpoint", "db.example.com:5432")
	
	assert.Equal(t, 2, len(state.Outputs))
	assert.Equal(t, "https://api.example.com", state.Outputs["api_url"])
}

func TestStateGetOutput(t *testing.T) {
	state := NewState("test-stack", "dev")
	
	state.SetOutput("api_url", "https://api.example.com")
	
	// Get existing output
	got, ok := state.GetOutput("api_url")
	assert.True(t, ok)
	assert.Equal(t, "https://api.example.com", got)
	
	// Get non-existent output
	got, ok = state.GetOutput("non-existent")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestStateClone(t *testing.T) {
	original := NewState("test-stack", "dev")
	
	resource := &Resource{
		ID:         "res-001",
		Type:       "aws_s3_bucket",
		Name:       "test-bucket",
		Attributes: map[string]interface{}{"region": "us-east-1"},
		DependsOn:  []string{"res-002"},
	}
	
	original.AddResource("bucket-1", resource)
	original.SetOutput("api_url", "https://api.example.com")
	
	// Clone the state
	cloned := original.Clone()
	
	// Verify clone is not nil and has same values
	require.NotNil(t, cloned)
	assert.Equal(t, original.Version, cloned.Version)
	assert.Equal(t, original.Metadata.Stack, cloned.Metadata.Stack)
	assert.Equal(t, len(original.Resources), len(cloned.Resources))
	assert.Equal(t, len(original.Outputs), len(cloned.Outputs))
	
	// Verify deep copy - modifying clone shouldn't affect original
	cloned.Resources["bucket-1"].Name = "modified-bucket"
	assert.NotEqual(t, cloned.Resources["bucket-1"].Name, original.Resources["bucket-1"].Name)
	
	cloned.SetOutput("api_url", "https://modified.example.com")
	assert.NotEqual(t, cloned.Outputs["api_url"], original.Outputs["api_url"])
}

func TestStateIsEmpty(t *testing.T) {
	// Nil state
	var nilState *State
	assert.True(t, nilState.IsEmpty())
	
	// New empty state
	emptyState := NewState("test-stack", "dev")
	assert.True(t, emptyState.IsEmpty())
	
	// State with resources
	stateWithResources := NewState("test-stack", "dev")
	stateWithResources.AddResource("res-1", &Resource{ID: "res-001"})
	assert.False(t, stateWithResources.IsEmpty())
}

func TestStateResourceCount(t *testing.T) {
	// Nil state
	var nilState *State
	assert.Equal(t, 0, nilState.ResourceCount())
	
	// Empty state
	emptyState := NewState("test-stack", "dev")
	assert.Equal(t, 0, emptyState.ResourceCount())
	
	// State with resources
	state := NewState("test-stack", "dev")
	state.AddResource("res-1", &Resource{ID: "res-001"})
	state.AddResource("res-2", &Resource{ID: "res-002"})
	assert.Equal(t, 2, state.ResourceCount())
}

func TestResourceStatus(t *testing.T) {
	// Test all status constants exist and are unique
	statuses := []ResourceStatus{
		ResourceStatusCreating,
		ResourceStatusReady,
		ResourceStatusUpdating,
		ResourceStatusDeleting,
		ResourceStatusFailed,
		ResourceStatusUnknown,
	}
	
	// Verify they're all different
	seen := make(map[ResourceStatus]bool)
	for _, status := range statuses {
		assert.False(t, seen[status], "duplicate status found: %s", status)
		seen[status] = true
		assert.NotEmpty(t, string(status))
	}
}

func TestStateVersion(t *testing.T) {
	now := time.Now()
	
	version := &StateVersion{
		VersionID:  "v123",
		Size:       1024,
		ModifiedAt: now,
		IsLatest:   true,
	}
	
	assert.Equal(t, "v123", version.VersionID)
	assert.Equal(t, int64(1024), version.Size)
	assert.Equal(t, now, version.ModifiedAt)
	assert.True(t, version.IsLatest)
}

