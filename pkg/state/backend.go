package state

import (
	"context"
)

// Backend defines the interface for state storage backends
type Backend interface {
	// Save saves the state to the backend
	Save(ctx context.Context, key string, state *State) error

	// Load loads the state from the backend
	Load(ctx context.Context, key string) (*State, error)

	// Exists checks if a state exists
	Exists(ctx context.Context, key string) (bool, error)

	// Delete deletes the state from the backend
	Delete(ctx context.Context, key string) error

	// List lists all state keys
	List(ctx context.Context, prefix string) ([]string, error)

	// ListVersions lists all versions of a state
	ListVersions(ctx context.Context, key string) ([]*StateVersion, error)

	// GetVersion gets a specific version of the state
	GetVersion(ctx context.Context, key string, versionID string) (*State, error)

	// Close closes the backend connection
	Close() error
}

// BackendConfig holds common backend configuration
type BackendConfig struct {
	// Type is the backend type (s3, local, etc.)
	Type string

	// Additional configuration can be added here
	MaxRetries int
	Timeout    int // seconds
}

// DefaultBackendConfig returns default backend configuration
func DefaultBackendConfig() *BackendConfig {
	return &BackendConfig{
		Type:       "s3",
		MaxRetries: 3,
		Timeout:    30,
	}
}

