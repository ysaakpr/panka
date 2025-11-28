package lock

import (
	"context"
	"fmt"
	"time"
	
	"github.com/yourusername/panka/pkg/tenant"
)

// TenantAwareManager wraps a Manager and applies tenant isolation
type TenantAwareManager struct {
	manager Manager
}

// NewTenantAwareManager creates a new tenant-aware lock manager wrapper
func NewTenantAwareManager(manager Manager) *TenantAwareManager {
	return &TenantAwareManager{
		manager: manager,
	}
}

// Acquire acquires a lock with tenant isolation
func (tm *TenantAwareManager) Acquire(ctx context.Context, key string, ttl time.Duration, owner string) (*Lock, error) {
	key = tm.applyTenantPrefix(ctx, key)
	return tm.manager.Acquire(ctx, key, ttl, owner)
}

// Refresh refreshes a lock with tenant isolation
func (tm *TenantAwareManager) Refresh(ctx context.Context, lock *Lock) error {
	// Lock already has the full key with tenant prefix
	return tm.manager.Refresh(ctx, lock)
}

// Release releases a lock with tenant isolation
func (tm *TenantAwareManager) Release(ctx context.Context, lock *Lock) error {
	// Lock already has the full key with tenant prefix
	return tm.manager.Release(ctx, lock)
}

// ForceRelease forces release of a lock with tenant isolation
func (tm *TenantAwareManager) ForceRelease(ctx context.Context, key string) error {
	key = tm.applyTenantPrefix(ctx, key)
	return tm.manager.ForceRelease(ctx, key)
}

// Get retrieves lock info with tenant isolation
func (tm *TenantAwareManager) Get(ctx context.Context, key string) (*LockInfo, error) {
	key = tm.applyTenantPrefix(ctx, key)
	return tm.manager.Get(ctx, key)
}

// List lists locks with tenant isolation
func (tm *TenantAwareManager) List(ctx context.Context, prefix string) ([]*LockInfo, error) {
	prefix = tm.applyTenantPrefix(ctx, prefix)
	locks, err := tm.manager.List(ctx, prefix)
	if err != nil {
		return nil, err
	}
	
	// Filter locks by tenant prefix
	return tm.filterTenantLocks(ctx, locks), nil
}

// Close closes the lock manager
func (tm *TenantAwareManager) Close() error {
	return tm.manager.Close()
}

// applyTenantPrefix adds tenant prefix to key if in tenant mode
func (tm *TenantAwareManager) applyTenantPrefix(ctx context.Context, key string) string {
	if tenantCtx, ok := tenant.FromContext(ctx); ok && tenantCtx.Enabled {
		// Tenant lock format: tenant:<tenant-id>:<key>
		return fmt.Sprintf("%s:%s", tenantCtx.LockPrefix, key)
	}
	return key
}

// filterTenantLocks filters locks to only include those for the current tenant
func (tm *TenantAwareManager) filterTenantLocks(ctx context.Context, locks []*LockInfo) []*LockInfo {
	if tenantCtx, ok := tenant.FromContext(ctx); ok && tenantCtx.Enabled {
		filtered := make([]*LockInfo, 0)
		prefix := tenantCtx.LockPrefix + ":"
		for _, lock := range locks {
			if len(lock.Key) > len(prefix) && lock.Key[:len(prefix)] == prefix {
				filtered = append(filtered, lock)
			}
		}
		return filtered
	}
	return locks
}

