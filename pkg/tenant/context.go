package tenant

import (
	"context"
	"fmt"
)

// ContextKey is the type for tenant context keys
type ContextKey string

const (
	// TenantContextKey is the key for storing tenant info in context
	TenantContextKey ContextKey = "tenant"
)

// TenantContext holds tenant information for the current operation
type TenantContext struct {
	TenantID    string
	StoragePath string
	LockPrefix  string
	Enabled     bool
}

// WithTenant adds tenant context to the context
func WithTenant(ctx context.Context, tenantCtx *TenantContext) context.Context {
	return context.WithValue(ctx, TenantContextKey, tenantCtx)
}

// FromContext retrieves tenant context from the context
func FromContext(ctx context.Context) (*TenantContext, bool) {
	tenantCtx, ok := ctx.Value(TenantContextKey).(*TenantContext)
	return tenantCtx, ok
}

// GetTenantPrefix returns the tenant prefix for state keys
func GetTenantPrefix(ctx context.Context) string {
	if tenantCtx, ok := FromContext(ctx); ok && tenantCtx.Enabled {
		return tenantCtx.StoragePath
	}
	return ""
}

// GetLockPrefix returns the tenant prefix for lock keys
func GetLockPrefix(ctx context.Context, lockKey string) string {
	if tenantCtx, ok := FromContext(ctx); ok && tenantCtx.Enabled {
		return fmt.Sprintf("%s:%s", tenantCtx.LockPrefix, lockKey)
	}
	return lockKey
}

// LoadTenantContext loads tenant context from the current session
func LoadTenantContext() (*TenantContext, error) {
	sessionMgr := NewSessionManager()
	session, err := sessionMgr.LoadSession()
	if err != nil {
		// No session, return empty context (single-tenant mode)
		return &TenantContext{Enabled: false}, nil
	}

	// Admin mode doesn't use tenant isolation
	if session.Mode == ModeAdmin {
		return &TenantContext{Enabled: false}, nil
	}

	// Tenant mode
	if session.Mode == ModeTenant && session.Tenant != nil {
		return &TenantContext{
			TenantID:    session.Tenant.ID,
			StoragePath: session.Backend.Prefix,
			LockPrefix:  session.Locks.Prefix,
			Enabled:     true,
		}, nil
	}

	return &TenantContext{Enabled: false}, nil
}

