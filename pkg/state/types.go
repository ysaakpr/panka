package state

import (
	"time"
)

// State represents the deployment state for a stack
type State struct {
	Version    string                 `json:"version"`
	Metadata   StateMetadata          `json:"metadata"`
	Resources  map[string]*Resource   `json:"resources"`
	Outputs    map[string]interface{} `json:"outputs"`
	LastUpdate time.Time              `json:"last_update"`
}

// StateMetadata contains metadata about the state
type StateMetadata struct {
	Stack       string            `json:"stack"`
	Environment string            `json:"environment"`
	Tenant      string            `json:"tenant,omitempty"`
	Version     string            `json:"version"`
	DeployedBy  string            `json:"deployed_by"`
	Labels      map[string]string `json:"labels,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Resource represents a deployed resource in the state
type Resource struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Provider   string                 `json:"provider"` // aws, local, etc.
	Status     ResourceStatus         `json:"status"`
	Attributes map[string]interface{} `json:"attributes"`
	DependsOn  []string               `json:"depends_on,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// ResourceStatus represents the status of a resource
type ResourceStatus string

const (
	// ResourceStatusCreating indicates resource is being created
	ResourceStatusCreating ResourceStatus = "creating"
	// ResourceStatusReady indicates resource is ready
	ResourceStatusReady ResourceStatus = "ready"
	// ResourceStatusUpdating indicates resource is being updated
	ResourceStatusUpdating ResourceStatus = "updating"
	// ResourceStatusDeleting indicates resource is being deleted
	ResourceStatusDeleting ResourceStatus = "deleting"
	// ResourceStatusFailed indicates resource operation failed
	ResourceStatusFailed ResourceStatus = "failed"
	// ResourceStatusUnknown indicates resource status is unknown
	ResourceStatusUnknown ResourceStatus = "unknown"
)

// StateVersion represents a version of the state with metadata
type StateVersion struct {
	VersionID  string    `json:"version_id"`
	State      *State    `json:"state"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	IsLatest   bool      `json:"is_latest"`
}

// NewState creates a new empty state
func NewState(stack, environment string) *State {
	now := time.Now()
	return &State{
		Version: "1.0",
		Metadata: StateMetadata{
			Stack:       stack,
			Environment: environment,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Resources:  make(map[string]*Resource),
		Outputs:    make(map[string]interface{}),
		LastUpdate: now,
	}
}

// AddResource adds a resource to the state
func (s *State) AddResource(id string, resource *Resource) {
	if s.Resources == nil {
		s.Resources = make(map[string]*Resource)
	}
	resource.UpdatedAt = time.Now()
	if resource.CreatedAt.IsZero() {
		resource.CreatedAt = resource.UpdatedAt
	}
	s.Resources[id] = resource
	s.LastUpdate = time.Now()
	s.Metadata.UpdatedAt = s.LastUpdate
}

// RemoveResource removes a resource from the state
func (s *State) RemoveResource(id string) {
	if s.Resources != nil {
		delete(s.Resources, id)
		s.LastUpdate = time.Now()
		s.Metadata.UpdatedAt = s.LastUpdate
	}
}

// GetResource gets a resource from the state
func (s *State) GetResource(id string) (*Resource, bool) {
	if s.Resources == nil {
		return nil, false
	}
	resource, ok := s.Resources[id]
	return resource, ok
}

// SetOutput sets an output value
func (s *State) SetOutput(key string, value interface{}) {
	if s.Outputs == nil {
		s.Outputs = make(map[string]interface{})
	}
	s.Outputs[key] = value
	s.LastUpdate = time.Now()
	s.Metadata.UpdatedAt = s.LastUpdate
}

// GetOutput gets an output value
func (s *State) GetOutput(key string) (interface{}, bool) {
	if s.Outputs == nil {
		return nil, false
	}
	value, ok := s.Outputs[key]
	return value, ok
}

// Clone creates a deep copy of the state
func (s *State) Clone() *State {
	if s == nil {
		return nil
	}

	clone := &State{
		Version:    s.Version,
		Metadata:   s.Metadata,
		Resources:  make(map[string]*Resource, len(s.Resources)),
		Outputs:    make(map[string]interface{}, len(s.Outputs)),
		LastUpdate: s.LastUpdate,
	}

	// Deep copy resources
	for k, v := range s.Resources {
		resourceCopy := *v
		if v.Attributes != nil {
			resourceCopy.Attributes = make(map[string]interface{}, len(v.Attributes))
			for ak, av := range v.Attributes {
				resourceCopy.Attributes[ak] = av
			}
		}
		if v.DependsOn != nil {
			resourceCopy.DependsOn = make([]string, len(v.DependsOn))
			copy(resourceCopy.DependsOn, v.DependsOn)
		}
		clone.Resources[k] = &resourceCopy
	}

	// Deep copy outputs
	for k, v := range s.Outputs {
		clone.Outputs[k] = v
	}

	return clone
}

// IsEmpty returns true if the state has no resources
func (s *State) IsEmpty() bool {
	return s == nil || len(s.Resources) == 0
}

// ResourceCount returns the number of resources in the state
func (s *State) ResourceCount() int {
	if s == nil || s.Resources == nil {
		return 0
	}
	return len(s.Resources)
}

