package tenant

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/panka/internal/logger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Manager manages tenant operations
type Manager struct {
	logger   *logger.Logger
	registry *Registry
	backend  RegistryBackend
}

// RegistryBackend defines the interface for loading/saving the tenant registry
type RegistryBackend interface {
	LoadRegistry(ctx context.Context) (*Registry, error)
	SaveRegistry(ctx context.Context, registry *Registry) error
}

// NewManager creates a new tenant manager
func NewManager(backend RegistryBackend) *Manager {
	log, _ := logger.NewDevelopment()
	return &Manager{
		logger:  log,
		backend: backend,
	}
}

// LoadRegistry loads the tenant registry
func (m *Manager) LoadRegistry(ctx context.Context) error {
	registry, err := m.backend.LoadRegistry(ctx)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	
	m.registry = registry
	m.logger.Info("Registry loaded",
		zap.Int("tenants", len(registry.Tenants)),
		zap.String("bucket", registry.Metadata.Bucket),
	)
	
	return nil
}

// SaveRegistry saves the tenant registry
func (m *Manager) SaveRegistry(ctx context.Context) error {
	if m.registry == nil {
		return fmt.Errorf("registry not loaded")
	}
	
	m.registry.Metadata.Updated = time.Now()
	
	if err := m.backend.SaveRegistry(ctx, m.registry); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}
	
	m.logger.Info("Registry saved",
		zap.Int("tenants", len(m.registry.Tenants)),
	)
	
	return nil
}

// CreateTenant creates a new tenant
func (m *Manager) CreateTenant(ctx context.Context, req *CreateTenantRequest) (*Tenant, *TenantCredentials, error) {
	if m.registry == nil {
		return nil, nil, fmt.Errorf("registry not loaded")
	}
	
	// Validate tenant name
	if err := validateTenantName(req.Name); err != nil {
		return nil, nil, err
	}
	
	// Check if tenant already exists
	if m.GetTenant(req.Name) != nil {
		return nil, nil, fmt.Errorf("tenant already exists: %s", req.Name)
	}
	
	m.logger.Info("Creating tenant", zap.String("name", req.Name))
	
	// Generate credentials
	creds, err := GenerateCredentials(req.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate credentials: %w", err)
	}
	
	// Set defaults
	if req.Version == "" {
		req.Version = m.registry.Config.DefaultVersion
	}
	if req.MaxStacks == 0 {
		req.MaxStacks = 100
	}
	if req.MaxServices == 0 {
		req.MaxServices = 500
	}
	
	// Create tenant
	tenant := &Tenant{
		ID:          req.Name,
		DisplayName: req.DisplayName,
		Email:       req.Email,
		Status:      StatusActive,
		Created:     time.Now(),
		Updated:     time.Now(),
		
		Credentials: Credentials{
			Hash:        creds.Hash,
			Algorithm:   "bcrypt",
			Rotations:   0,
			LastRotated: nil,
		},
		
		Storage: StorageConfig{
			Prefix:  fmt.Sprintf("tenants/%s", req.Name),
			Version: req.Version,
			Path:    fmt.Sprintf("tenants/%s/%s", req.Name, req.Version),
		},
		
		Locks: LockConfig{
			Prefix: fmt.Sprintf("tenant:%s", req.Name),
		},
		
		AWS: AWSConfig{
			AccountID: req.AWSAccountID,
			Region:    m.registry.Metadata.Region,
		},
		
		Limits: Limits{
			CostTracking:     req.CostTracking,
			MonthlyCostLimit: req.MonthlyCostLimit,
			MaxStacks:        req.MaxStacks,
			MaxServices:      req.MaxServices,
		},
		
		Metadata: req.Metadata,
	}
	
	// Add to registry
	m.registry.Tenants = append(m.registry.Tenants, *tenant)
	
	// Save registry
	if err := m.SaveRegistry(ctx); err != nil {
		// Rollback: remove from registry
		m.registry.Tenants = m.registry.Tenants[:len(m.registry.Tenants)-1]
		return nil, nil, fmt.Errorf("failed to save registry: %w", err)
	}
	
	m.logger.Info("Tenant created successfully",
		zap.String("tenant", req.Name),
		zap.String("prefix", tenant.Storage.Prefix),
	)
	
	return tenant, creds, nil
}

// GetTenant retrieves a tenant by ID
func (m *Manager) GetTenant(tenantID string) *Tenant {
	if m.registry == nil {
		return nil
	}
	
	for i := range m.registry.Tenants {
		if m.registry.Tenants[i].ID == tenantID {
			return &m.registry.Tenants[i]
		}
	}
	
	return nil
}

// ListTenants returns all tenants
func (m *Manager) ListTenants() []Tenant {
	if m.registry == nil {
		return []Tenant{}
	}
	
	return m.registry.Tenants
}

// UpdateTenant updates an existing tenant
func (m *Manager) UpdateTenant(ctx context.Context, tenant *Tenant) error {
	if m.registry == nil {
		return fmt.Errorf("registry not loaded")
	}
	
	// Find tenant
	found := false
	for i := range m.registry.Tenants {
		if m.registry.Tenants[i].ID == tenant.ID {
			tenant.Updated = time.Now()
			m.registry.Tenants[i] = *tenant
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("tenant not found: %s", tenant.ID)
	}
	
	// Save registry
	if err := m.SaveRegistry(ctx); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}
	
	m.logger.Info("Tenant updated", zap.String("tenant", tenant.ID))
	
	return nil
}

// RotateTenantCredentials rotates credentials for a tenant
func (m *Manager) RotateTenantCredentials(ctx context.Context, tenantID string) (*TenantCredentials, error) {
	tenant := m.GetTenant(tenantID)
	if tenant == nil {
		return nil, fmt.Errorf("tenant not found: %s", tenantID)
	}
	
	m.logger.Info("Rotating credentials", zap.String("tenant", tenantID))
	
	// Generate new credentials
	creds, err := GenerateCredentials(tenantID)
	if err != nil {
		return nil, err
	}
	
	// Update tenant
	now := time.Now()
	tenant.Credentials.Hash = creds.Hash
	tenant.Credentials.Rotations++
	tenant.Credentials.LastRotated = &now
	tenant.Updated = now
	
	// Save registry
	if err := m.UpdateTenant(ctx, tenant); err != nil {
		return nil, err
	}
	
	m.logger.Info("Credentials rotated",
		zap.String("tenant", tenantID),
		zap.Int("rotations", tenant.Credentials.Rotations),
	)
	
	return creds, nil
}

// SuspendTenant suspends a tenant
func (m *Manager) SuspendTenant(ctx context.Context, tenantID string) error {
	tenant := m.GetTenant(tenantID)
	if tenant == nil {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}
	
	if tenant.Status == StatusSuspended {
		return fmt.Errorf("tenant already suspended: %s", tenantID)
	}
	
	m.logger.Info("Suspending tenant", zap.String("tenant", tenantID))
	
	tenant.Status = StatusSuspended
	tenant.Updated = time.Now()
	
	return m.UpdateTenant(ctx, tenant)
}

// ActivateTenant activates a suspended tenant
func (m *Manager) ActivateTenant(ctx context.Context, tenantID string) error {
	tenant := m.GetTenant(tenantID)
	if tenant == nil {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}
	
	if tenant.Status == StatusActive {
		return fmt.Errorf("tenant already active: %s", tenantID)
	}
	
	m.logger.Info("Activating tenant", zap.String("tenant", tenantID))
	
	tenant.Status = StatusActive
	tenant.Updated = time.Now()
	
	return m.UpdateTenant(ctx, tenant)
}

// DeleteTenant deletes a tenant
func (m *Manager) DeleteTenant(ctx context.Context, tenantID string) error {
	if m.registry == nil {
		return fmt.Errorf("registry not loaded")
	}
	
	tenant := m.GetTenant(tenantID)
	if tenant == nil {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}
	
	m.logger.Info("Deleting tenant", zap.String("tenant", tenantID))
	
	// Mark as deleted (soft delete)
	tenant.Status = StatusDeleted
	tenant.Updated = time.Now()
	
	return m.UpdateTenant(ctx, tenant)
}

// VerifyTenantCredentials verifies tenant credentials
func (m *Manager) VerifyTenantCredentials(tenantID, secret string) (*Tenant, error) {
	tenant := m.GetTenant(tenantID)
	if tenant == nil {
		return nil, fmt.Errorf("tenant not found: %s", tenantID)
	}
	
	if tenant.Status != StatusActive {
		return nil, fmt.Errorf("tenant is not active (status: %s)", tenant.Status)
	}
	
	if !VerifyCredentials(secret, tenant.Credentials.Hash) {
		m.logger.Warn("Invalid credentials for tenant", zap.String("tenant", tenantID))
		return nil, fmt.Errorf("invalid credentials")
	}
	
	m.logger.Info("Tenant credentials verified", zap.String("tenant", tenantID))
	
	return tenant, nil
}

// Helper functions

func validateTenantName(name string) error {
	if name == "" {
		return fmt.Errorf("tenant name cannot be empty")
	}
	
	if len(name) < 3 {
		return fmt.Errorf("tenant name must be at least 3 characters")
	}
	
	if len(name) > 63 {
		return fmt.Errorf("tenant name must be at most 63 characters")
	}
	
	// Must be lowercase alphanumeric with hyphens
	for i, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return fmt.Errorf("tenant name must be lowercase alphanumeric with hyphens")
		}
		
		// Cannot start or end with hyphen
		if (i == 0 || i == len(name)-1) && c == '-' {
			return fmt.Errorf("tenant name cannot start or end with hyphen")
		}
	}
	
	return nil
}

// MarshalRegistry marshals the registry to YAML
func MarshalRegistry(registry *Registry) ([]byte, error) {
	return yaml.Marshal(registry)
}

// UnmarshalRegistry unmarshals YAML to registry
func UnmarshalRegistry(data []byte) (*Registry, error) {
	var registry Registry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, err
	}
	return &registry, nil
}

