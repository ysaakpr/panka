// Package rollback provides functionality to revert infrastructure changes on failure.
package rollback

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
	"github.com/yourusername/panka/pkg/state"
)

// Action represents a reversible action that was performed
type Action struct {
	// Type of action (create, update, delete)
	Type ActionType `json:"type"`

	// ResourceName is the name of the resource
	ResourceName string `json:"resource_name"`

	// ResourceID is the provider resource ID
	ResourceID string `json:"resource_id"`

	// ResourceKind is the type of resource
	ResourceKind schema.Kind `json:"resource_kind"`

	// BeforeState is the state before the action (nil for create)
	BeforeState *state.Resource `json:"before_state,omitempty"`

	// AfterState is the state after the action (nil for delete)
	AfterState *state.Resource `json:"after_state,omitempty"`

	// PerformedAt is when the action was performed
	PerformedAt time.Time `json:"performed_at"`

	// Success indicates if the action succeeded
	Success bool `json:"success"`

	// Error if the action failed
	Error string `json:"error,omitempty"`
}

// ActionType represents the type of infrastructure action
type ActionType string

const (
	ActionCreate ActionType = "create"
	ActionUpdate ActionType = "update"
	ActionDelete ActionType = "delete"
)

// RollbackPlan contains the plan to revert changes
type RollbackPlan struct {
	// StackName is the stack being rolled back
	StackName string `json:"stack_name"`

	// TenantID is the tenant being rolled back
	TenantID string `json:"tenant_id,omitempty"`

	// Actions to reverse in order
	Actions []*Action `json:"actions"`

	// SnapshotState is the state snapshot taken before changes
	SnapshotState *state.State `json:"snapshot_state,omitempty"`

	// CreatedAt is when the rollback plan was created
	CreatedAt time.Time `json:"created_at"`
}

// RollbackResult contains the result of a rollback operation
type RollbackResult struct {
	// Plan is the rollback plan that was executed
	Plan *RollbackPlan `json:"plan"`

	// Success indicates if the rollback completed successfully
	Success bool `json:"success"`

	// SuccessCount is the number of actions successfully reversed
	SuccessCount int `json:"success_count"`

	// FailedCount is the number of actions that failed to reverse
	FailedCount int `json:"failed_count"`

	// SkippedCount is the number of actions that were skipped
	SkippedCount int `json:"skipped_count"`

	// Errors contains details of any failures
	Errors []RollbackError `json:"errors,omitempty"`

	// Duration is how long the rollback took
	Duration time.Duration `json:"duration"`
}

// RollbackError represents an error during rollback
type RollbackError struct {
	ResourceName string `json:"resource_name"`
	Action       string `json:"action"`
	Error        string `json:"error"`
}

// Manager manages rollback operations
type Manager struct {
	// provider is the cloud provider for executing rollback
	provider provider.Provider

	// currentPlan is the rollback plan being built
	currentPlan *RollbackPlan

	// maxRetries is the number of times to retry a rollback action
	maxRetries int
}

// NewManager creates a new rollback manager
func NewManager(prov provider.Provider) *Manager {
	return &Manager{
		provider:   prov,
		maxRetries: 3,
	}
}

// StartTransaction begins a new rollback transaction
func (m *Manager) StartTransaction(stackName, tenantID string, snapshotState *state.State) {
	m.currentPlan = &RollbackPlan{
		StackName:     stackName,
		TenantID:      tenantID,
		Actions:       make([]*Action, 0),
		SnapshotState: snapshotState.Clone(),
		CreatedAt:     time.Now(),
	}
}

// RecordAction records an action for potential rollback
func (m *Manager) RecordAction(action *Action) {
	if m.currentPlan == nil {
		return
	}
	action.PerformedAt = time.Now()
	m.currentPlan.Actions = append(m.currentPlan.Actions, action)
}

// RecordCreate records a create action
func (m *Manager) RecordCreate(name string, id string, kind schema.Kind, resultState *state.Resource, success bool, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	m.RecordAction(&Action{
		Type:         ActionCreate,
		ResourceName: name,
		ResourceID:   id,
		ResourceKind: kind,
		BeforeState:  nil,
		AfterState:   resultState,
		Success:      success,
		Error:        errStr,
	})
}

// RecordUpdate records an update action
func (m *Manager) RecordUpdate(name string, id string, kind schema.Kind, beforeState, afterState *state.Resource, success bool, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	m.RecordAction(&Action{
		Type:         ActionUpdate,
		ResourceName: name,
		ResourceID:   id,
		ResourceKind: kind,
		BeforeState:  beforeState,
		AfterState:   afterState,
		Success:      success,
		Error:        errStr,
	})
}

// RecordDelete records a delete action
func (m *Manager) RecordDelete(name string, id string, kind schema.Kind, beforeState *state.Resource, success bool, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	m.RecordAction(&Action{
		Type:         ActionDelete,
		ResourceName: name,
		ResourceID:   id,
		ResourceKind: kind,
		BeforeState:  beforeState,
		AfterState:   nil,
		Success:      success,
		Error:        errStr,
	})
}

// GetPlan returns the current rollback plan
func (m *Manager) GetPlan() *RollbackPlan {
	return m.currentPlan
}

// ClearTransaction clears the current transaction (on successful completion)
func (m *Manager) ClearTransaction() {
	m.currentPlan = nil
}

// Rollback executes a rollback of all recorded actions
func (m *Manager) Rollback(ctx context.Context) (*RollbackResult, error) {
	if m.currentPlan == nil || len(m.currentPlan.Actions) == 0 {
		return &RollbackResult{Success: true}, nil
	}

	startTime := time.Now()
	result := &RollbackResult{
		Plan:   m.currentPlan,
		Errors: make([]RollbackError, 0),
	}

	// Process actions in reverse order
	for i := len(m.currentPlan.Actions) - 1; i >= 0; i-- {
		action := m.currentPlan.Actions[i]

		// Skip unsuccessful actions - they don't need rollback
		if !action.Success {
			result.SkippedCount++
			continue
		}

		// Rollback based on action type
		var err error
		switch action.Type {
		case ActionCreate:
			// Reverse create by deleting
			err = m.rollbackCreate(ctx, action)
		case ActionUpdate:
			// Reverse update by restoring previous state
			err = m.rollbackUpdate(ctx, action)
		case ActionDelete:
			// Cannot easily reverse delete - log warning
			err = fmt.Errorf("cannot automatically restore deleted resource")
			result.SkippedCount++
			continue
		}

		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, RollbackError{
				ResourceName: action.ResourceName,
				Action:       string(action.Type),
				Error:        err.Error(),
			})
		} else {
			result.SuccessCount++
		}
	}

	result.Duration = time.Since(startTime)
	result.Success = result.FailedCount == 0

	// Clear the plan after rollback attempt
	m.ClearTransaction()

	return result, nil
}

// rollbackCreate reverses a create action by deleting the resource
func (m *Manager) rollbackCreate(ctx context.Context, action *Action) error {
	if action.ResourceID == "" {
		return fmt.Errorf("no resource ID to delete")
	}

	resourceProvider, err := m.provider.GetResourceProvider(action.ResourceKind)
	if err != nil {
		return fmt.Errorf("no provider for kind %s: %w", action.ResourceKind, err)
	}

	_, err = resourceProvider.Delete(ctx, action.ResourceID, &provider.ResourceOptions{})
	return err
}

// rollbackUpdate reverses an update action by restoring the previous state
func (m *Manager) rollbackUpdate(ctx context.Context, action *Action) error {
	if action.BeforeState == nil {
		return fmt.Errorf("no before state to restore")
	}

	// Note: In a full implementation, you would need to:
	// 1. Convert the state.Resource back to a schema.Resource
	// 2. Call Update with the previous configuration
	// For now, we just log that this would need to happen

	return fmt.Errorf("update rollback not yet implemented - manual intervention required")
}

// CanRollback returns true if there are actions that can be rolled back
func (m *Manager) CanRollback() bool {
	if m.currentPlan == nil {
		return false
	}
	for _, action := range m.currentPlan.Actions {
		if action.Success && action.Type == ActionCreate {
			return true
		}
	}
	return false
}

// ActionCount returns the number of recorded actions
func (m *Manager) ActionCount() int {
	if m.currentPlan == nil {
		return 0
	}
	return len(m.currentPlan.Actions)
}

// String returns a summary of the rollback plan
func (p *RollbackPlan) String() string {
	creates := 0
	updates := 0
	deletes := 0
	for _, a := range p.Actions {
		switch a.Type {
		case ActionCreate:
			creates++
		case ActionUpdate:
			updates++
		case ActionDelete:
			deletes++
		}
	}
	return fmt.Sprintf("RollbackPlan for %s: %d creates, %d updates, %d deletes to reverse",
		p.StackName, creates, updates, deletes)
}

