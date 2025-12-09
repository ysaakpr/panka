package diff

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
	"github.com/yourusername/panka/pkg/state"
)

// DriftType represents the type of drift detected
type DriftType string

const (
	// DriftNone indicates no drift was detected
	DriftNone DriftType = "none"
	// DriftModified indicates resource was modified outside of Panka
	DriftModified DriftType = "modified"
	// DriftDeleted indicates resource was deleted outside of Panka
	DriftDeleted DriftType = "deleted"
	// DriftUnknown indicates drift status could not be determined
	DriftUnknown DriftType = "unknown"
)

// DriftResult represents the result of drift detection for a single resource
type DriftResult struct {
	// ResourceName is the name of the resource
	ResourceName string `json:"resource_name"`

	// ResourceID is the AWS resource ID
	ResourceID string `json:"resource_id"`

	// ResourceKind is the type of resource
	ResourceKind schema.Kind `json:"resource_kind"`

	// Type of drift detected
	Type DriftType `json:"type"`

	// StoredState is what Panka has recorded
	StoredState *state.Resource `json:"stored_state,omitempty"`

	// ActualState is what exists in AWS
	ActualState map[string]string `json:"actual_state,omitempty"`

	// Diffs lists the specific differences found
	Diffs []DriftDiff `json:"diffs,omitempty"`

	// DetectedAt is when drift was detected
	DetectedAt time.Time `json:"detected_at"`

	// Error if detection failed
	Error string `json:"error,omitempty"`
}

// DriftDiff represents a single difference between stored and actual state
type DriftDiff struct {
	// Attribute is the name of the attribute that differs
	Attribute string `json:"attribute"`

	// StoredValue is the value in Panka's state
	StoredValue interface{} `json:"stored_value,omitempty"`

	// ActualValue is the value in AWS
	ActualValue interface{} `json:"actual_value,omitempty"`

	// Sensitive indicates if the value should be masked
	Sensitive bool `json:"sensitive,omitempty"`
}

// DriftReport contains the full drift detection report
type DriftReport struct {
	// StackName is the stack being checked
	StackName string `json:"stack_name"`

	// Environment is the target environment
	Environment string `json:"environment"`

	// TenantID is the tenant being checked
	TenantID string `json:"tenant_id,omitempty"`

	// Results contains drift detection results for each resource
	Results []*DriftResult `json:"results"`

	// Summary provides counts by drift type
	Summary DriftSummary `json:"summary"`

	// GeneratedAt is when the report was generated
	GeneratedAt time.Time `json:"generated_at"`

	// Duration is how long the scan took
	Duration time.Duration `json:"duration"`
}

// DriftSummary provides a summary of drift detection results
type DriftSummary struct {
	Total    int `json:"total"`
	Clean    int `json:"clean"`      // No drift
	Modified int `json:"modified"`   // Changed in AWS
	Deleted  int `json:"deleted"`    // Deleted from AWS
	Unknown  int `json:"unknown"`    // Could not determine
	Errors   int `json:"errors"`     // Detection failed
}

// DriftDetector checks for configuration drift
type DriftDetector struct {
	// provider is the cloud provider to query
	provider provider.Provider

	// options for drift detection
	options *DriftDetectorOptions
}

// DriftDetectorOptions configures drift detection behavior
type DriftDetectorOptions struct {
	// IgnoreAttributes lists attributes to ignore during comparison
	IgnoreAttributes []string

	// Parallel sets the number of parallel checks
	Parallel int

	// Timeout for each resource check
	Timeout time.Duration
}

// DefaultDriftDetectorOptions returns default drift detector options
func DefaultDriftDetectorOptions() *DriftDetectorOptions {
	return &DriftDetectorOptions{
		IgnoreAttributes: []string{"updated_at", "created_at", "tags"},
		Parallel:         5,
		Timeout:          30 * time.Second,
	}
}

// NewDriftDetector creates a new drift detector
func NewDriftDetector(prov provider.Provider, opts *DriftDetectorOptions) *DriftDetector {
	if opts == nil {
		opts = DefaultDriftDetectorOptions()
	}
	return &DriftDetector{
		provider: prov,
		options:  opts,
	}
}

// DetectDrift checks for drift in all resources in the state
func (d *DriftDetector) DetectDrift(
	ctx context.Context,
	currentState *state.State,
) (*DriftReport, error) {
	startTime := time.Now()

	report := &DriftReport{
		StackName:   currentState.Metadata.Stack,
		Environment: currentState.Metadata.Environment,
		TenantID:    currentState.Metadata.Tenant,
		Results:     make([]*DriftResult, 0),
		GeneratedAt: time.Now(),
	}

	// Get all resources from state
	resources := currentState.ListResources()

	for _, res := range resources {
		result := d.checkResource(ctx, res)
		report.Results = append(report.Results, result)
		d.updateSummary(&report.Summary, result)
	}

	report.Duration = time.Since(startTime)
	return report, nil
}

// checkResource checks a single resource for drift
func (d *DriftDetector) checkResource(ctx context.Context, res *state.Resource) *DriftResult {
	result := &DriftResult{
		ResourceName: res.Name,
		ResourceID:   res.ID,
		ResourceKind: schema.Kind(res.Type),
		StoredState:  res,
		DetectedAt:   time.Now(),
	}

	// Get the resource provider
	resourceProvider, err := d.provider.GetResourceProvider(schema.Kind(res.Type))
	if err != nil {
		result.Type = DriftUnknown
		result.Error = fmt.Sprintf("no provider for resource type: %s", res.Type)
		return result
	}

	// Check if resource exists
	exists, err := resourceProvider.Exists(ctx, res.ID, &provider.ResourceOptions{})
	if err != nil {
		result.Type = DriftUnknown
		result.Error = fmt.Sprintf("failed to check existence: %v", err)
		return result
	}

	if !exists {
		result.Type = DriftDeleted
		return result
	}

	// Get current state from AWS
	readResult, err := resourceProvider.Read(ctx, res.ID, &provider.ResourceOptions{})
	if err != nil {
		result.Type = DriftUnknown
		result.Error = fmt.Sprintf("failed to read resource: %v", err)
		return result
	}

	result.ActualState = readResult.Outputs

	// Compare stored vs actual
	diffs := d.compareStates(res, readResult.Outputs)
	result.Diffs = diffs

	if len(diffs) > 0 {
		result.Type = DriftModified
	} else {
		result.Type = DriftNone
	}

	return result
}

// compareStates compares stored state with actual outputs
func (d *DriftDetector) compareStates(stored *state.Resource, actual map[string]string) []DriftDiff {
	var diffs []DriftDiff

	// Compare key attributes based on resource type
	// This is a simplified comparison - in a real implementation,
	// you'd have type-specific comparators

	// Check if any stored attributes differ from actual
	for key, storedValue := range stored.Attributes {
		// Skip ignored attributes
		if d.isIgnored(key) {
			continue
		}

		// Check if the key exists in actual
		actualValue, exists := actual[key]
		if !exists {
			continue // Can't compare if not in outputs
		}

		// Convert stored value to string for comparison
		storedStr := fmt.Sprintf("%v", storedValue)
		if storedStr != actualValue {
			diffs = append(diffs, DriftDiff{
				Attribute:   key,
				StoredValue: storedValue,
				ActualValue: actualValue,
			})
		}
	}

	return diffs
}

// isIgnored checks if an attribute should be ignored in drift detection
func (d *DriftDetector) isIgnored(attr string) bool {
	for _, ignored := range d.options.IgnoreAttributes {
		if ignored == attr {
			return true
		}
	}
	return false
}

// updateSummary updates the drift summary with a result
func (d *DriftDetector) updateSummary(summary *DriftSummary, result *DriftResult) {
	summary.Total++
	switch result.Type {
	case DriftNone:
		summary.Clean++
	case DriftModified:
		summary.Modified++
	case DriftDeleted:
		summary.Deleted++
	case DriftUnknown:
		summary.Unknown++
	}
	if result.Error != "" {
		summary.Errors++
	}
}

// HasDrift returns true if the report contains any drift
func (r *DriftReport) HasDrift() bool {
	return r.Summary.Modified > 0 || r.Summary.Deleted > 0
}

// String returns a summary string of the drift report
func (r *DriftReport) String() string {
	return fmt.Sprintf("Drift Report for %s/%s: %d clean, %d modified, %d deleted, %d unknown",
		r.StackName, r.Environment,
		r.Summary.Clean, r.Summary.Modified, r.Summary.Deleted, r.Summary.Unknown)
}

// Symbol returns a symbol for display purposes
func (t DriftType) Symbol() string {
	switch t {
	case DriftNone:
		return "✓"
	case DriftModified:
		return "~"
	case DriftDeleted:
		return "✗"
	case DriftUnknown:
		return "?"
	default:
		return " "
	}
}

