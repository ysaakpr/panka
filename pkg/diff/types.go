// Package diff provides infrastructure change detection and comparison.
// It compares desired infrastructure state with actual state to determine
// what changes need to be applied.
package diff

import (
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/state"
)

// ChangeType represents the type of change to be made
type ChangeType string

const (
	// ChangeCreate indicates a resource needs to be created
	ChangeCreate ChangeType = "create"
	// ChangeUpdate indicates a resource needs to be updated
	ChangeUpdate ChangeType = "update"
	// ChangeDelete indicates a resource needs to be deleted
	ChangeDelete ChangeType = "delete"
	// ChangeNoChange indicates no changes are needed
	ChangeNoChange ChangeType = "no-change"
	// ChangeRecreate indicates a resource needs to be deleted and recreated
	ChangeRecreate ChangeType = "recreate"
)

// Change represents a single resource change
type Change struct {
	// ResourceID is the identifier of the resource
	ResourceID string `json:"resource_id"`

	// ResourceName is the human-readable name of the resource
	ResourceName string `json:"resource_name"`

	// ResourceKind is the type of resource
	ResourceKind schema.Kind `json:"resource_kind"`

	// Type of change (create, update, delete)
	Type ChangeType `json:"type"`

	// Service the resource belongs to
	Service string `json:"service,omitempty"`

	// Before is the current state of the resource (nil if create)
	Before *state.Resource `json:"before,omitempty"`

	// After is the desired state of the resource (nil if delete)
	After schema.Resource `json:"-"` // Don't serialize the full resource

	// AttributeChanges contains the specific attribute changes
	AttributeChanges []AttributeChange `json:"attribute_changes,omitempty"`

	// Reason explains why the change is needed
	Reason string `json:"reason,omitempty"`

	// RequiresRecreate indicates if the resource must be recreated
	RequiresRecreate bool `json:"requires_recreate,omitempty"`

	// DependsOn lists resources this change depends on
	DependsOn []string `json:"depends_on,omitempty"`
}

// AttributeChange represents a change to a specific attribute
type AttributeChange struct {
	// Path is the JSON path to the attribute (e.g., "spec.replicas")
	Path string `json:"path"`

	// OldValue is the current value
	OldValue interface{} `json:"old_value,omitempty"`

	// NewValue is the desired value
	NewValue interface{} `json:"new_value,omitempty"`

	// Sensitive indicates if the value should be masked in output
	Sensitive bool `json:"sensitive,omitempty"`

	// ForceRecreate indicates if this change requires resource recreation
	ForceRecreate bool `json:"force_recreate,omitempty"`
}

// ChangeSet represents a collection of changes to be applied
type ChangeSet struct {
	// StackName is the name of the stack being changed
	StackName string `json:"stack_name"`

	// Environment is the target environment
	Environment string `json:"environment"`

	// TenantID is the tenant being changed
	TenantID string `json:"tenant_id,omitempty"`

	// Changes is the list of all changes
	Changes []*Change `json:"changes"`

	// Summary provides change counts by type
	Summary ChangeSummary `json:"summary"`

	// CreatedAt is when the change set was generated
	CreatedAt time.Time `json:"created_at"`
}

// ChangeSummary provides a summary of changes by type
type ChangeSummary struct {
	Create   int `json:"create"`
	Update   int `json:"update"`
	Delete   int `json:"delete"`
	NoChange int `json:"no_change"`
	Recreate int `json:"recreate"`
	Total    int `json:"total"`
}

// NewChangeSet creates a new change set
func NewChangeSet(stackName, environment string) *ChangeSet {
	return &ChangeSet{
		StackName:   stackName,
		Environment: environment,
		Changes:     make([]*Change, 0),
		CreatedAt:   time.Now(),
	}
}

// AddChange adds a change to the set
func (cs *ChangeSet) AddChange(change *Change) {
	cs.Changes = append(cs.Changes, change)
	cs.updateSummary(change.Type)
}

// updateSummary updates the summary counts
func (cs *ChangeSet) updateSummary(changeType ChangeType) {
	switch changeType {
	case ChangeCreate:
		cs.Summary.Create++
	case ChangeUpdate:
		cs.Summary.Update++
	case ChangeDelete:
		cs.Summary.Delete++
	case ChangeNoChange:
		cs.Summary.NoChange++
	case ChangeRecreate:
		cs.Summary.Recreate++
	}
	cs.Summary.Total++
}

// HasChanges returns true if there are any changes to apply
func (cs *ChangeSet) HasChanges() bool {
	return cs.Summary.Create > 0 || cs.Summary.Update > 0 ||
		cs.Summary.Delete > 0 || cs.Summary.Recreate > 0
}

// GetChangesByType returns changes filtered by type
func (cs *ChangeSet) GetChangesByType(changeType ChangeType) []*Change {
	var filtered []*Change
	for _, change := range cs.Changes {
		if change.Type == changeType {
			filtered = append(filtered, change)
		}
	}
	return filtered
}

// GetCreates returns all create changes
func (cs *ChangeSet) GetCreates() []*Change {
	return cs.GetChangesByType(ChangeCreate)
}

// GetUpdates returns all update changes
func (cs *ChangeSet) GetUpdates() []*Change {
	return cs.GetChangesByType(ChangeUpdate)
}

// GetDeletes returns all delete changes
func (cs *ChangeSet) GetDeletes() []*Change {
	return cs.GetChangesByType(ChangeDelete)
}

// String returns a human-readable summary
func (cs *ChangeSet) String() string {
	return fmt.Sprintf("ChangeSet for %s/%s: %d create, %d update, %d delete, %d no-change",
		cs.StackName, cs.Environment,
		cs.Summary.Create, cs.Summary.Update, cs.Summary.Delete, cs.Summary.NoChange)
}

// Symbol returns a symbol for display purposes
func (c ChangeType) Symbol() string {
	switch c {
	case ChangeCreate:
		return "+"
	case ChangeUpdate:
		return "~"
	case ChangeDelete:
		return "-"
	case ChangeRecreate:
		return "±"
	case ChangeNoChange:
		return " "
	default:
		return "?"
	}
}

// Color returns the ANSI color code for the change type
func (c ChangeType) Color() string {
	switch c {
	case ChangeCreate:
		return "\033[32m" // Green
	case ChangeUpdate:
		return "\033[33m" // Yellow
	case ChangeDelete:
		return "\033[31m" // Red
	case ChangeRecreate:
		return "\033[35m" // Magenta
	case ChangeNoChange:
		return "\033[90m" // Gray
	default:
		return "\033[0m" // Reset
	}
}

// String returns a human-readable representation of the change
func (c *Change) String() string {
	symbol := c.Type.Symbol()
	return fmt.Sprintf("%s [%s] %s", symbol, c.ResourceKind, c.ResourceName)
}

// DetailedString returns a detailed representation including attribute changes
func (c *Change) DetailedString() string {
	var sb strings.Builder
	sb.WriteString(c.String())

	if c.Reason != "" {
		sb.WriteString(fmt.Sprintf("\n    Reason: %s", c.Reason))
	}

	if len(c.AttributeChanges) > 0 {
		sb.WriteString("\n    Changes:")
		for _, attr := range c.AttributeChanges {
			if attr.Sensitive {
				sb.WriteString(fmt.Sprintf("\n      %s: (sensitive)", attr.Path))
			} else {
				sb.WriteString(fmt.Sprintf("\n      %s: %v -> %v", attr.Path, attr.OldValue, attr.NewValue))
			}
		}
	}

	return sb.String()
}

