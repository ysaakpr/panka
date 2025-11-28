package state

import (
	"context"
	"path/filepath"
	
	"github.com/yourusername/panka/pkg/tenant"
)

// TenantAwareBackend wraps a Backend and applies tenant isolation
type TenantAwareBackend struct {
	backend Backend
}

// NewTenantAwareBackend creates a new tenant-aware backend wrapper
func NewTenantAwareBackend(backend Backend) *TenantAwareBackend {
	return &TenantAwareBackend{
		backend: backend,
	}
}

// Save saves state with tenant isolation
func (tb *TenantAwareBackend) Save(ctx context.Context, key string, state *State) error {
	key = tb.applyTenantPrefix(ctx, key)
	return tb.backend.Save(ctx, key, state)
}

// Load loads state with tenant isolation
func (tb *TenantAwareBackend) Load(ctx context.Context, key string) (*State, error) {
	key = tb.applyTenantPrefix(ctx, key)
	return tb.backend.Load(ctx, key)
}

// Exists checks if state exists with tenant isolation
func (tb *TenantAwareBackend) Exists(ctx context.Context, key string) (bool, error) {
	key = tb.applyTenantPrefix(ctx, key)
	return tb.backend.Exists(ctx, key)
}

// Delete deletes state with tenant isolation
func (tb *TenantAwareBackend) Delete(ctx context.Context, key string) error {
	key = tb.applyTenantPrefix(ctx, key)
	return tb.backend.Delete(ctx, key)
}

// List lists states with tenant isolation
func (tb *TenantAwareBackend) List(ctx context.Context, prefix string) ([]string, error) {
	prefix = tb.applyTenantPrefix(ctx, prefix)
	keys, err := tb.backend.List(ctx, prefix)
	if err != nil {
		return nil, err
	}
	
	// Strip tenant prefix from returned keys
	return tb.stripTenantPrefix(ctx, keys), nil
}

// ListVersions lists state versions with tenant isolation
func (tb *TenantAwareBackend) ListVersions(ctx context.Context, key string) ([]*StateVersion, error) {
	key = tb.applyTenantPrefix(ctx, key)
	return tb.backend.ListVersions(ctx, key)
}

// GetVersion gets a specific state version with tenant isolation
func (tb *TenantAwareBackend) GetVersion(ctx context.Context, key string, versionID string) (*State, error) {
	key = tb.applyTenantPrefix(ctx, key)
	return tb.backend.GetVersion(ctx, key, versionID)
}

// Close closes the backend
func (tb *TenantAwareBackend) Close() error {
	return tb.backend.Close()
}

// applyTenantPrefix adds tenant prefix to key if in tenant mode
func (tb *TenantAwareBackend) applyTenantPrefix(ctx context.Context, key string) string {
	if tenantCtx, ok := tenant.FromContext(ctx); ok && tenantCtx.Enabled {
		// Tenant prefix format: tenants/<tenant-id>/v1/<key>
		return filepath.Join(tenantCtx.StoragePath, key)
	}
	return key
}

// stripTenantPrefix removes tenant prefix from keys
func (tb *TenantAwareBackend) stripTenantPrefix(ctx context.Context, keys []string) []string {
	if tenantCtx, ok := tenant.FromContext(ctx); ok && tenantCtx.Enabled {
		stripped := make([]string, len(keys))
		prefix := tenantCtx.StoragePath + "/"
		for i, key := range keys {
			if len(key) > len(prefix) && key[:len(prefix)] == prefix {
				stripped[i] = key[len(prefix):]
			} else {
				stripped[i] = key
			}
		}
		return stripped
	}
	return keys
}

