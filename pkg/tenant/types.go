package tenant

import (
	"time"
)

// Tenant represents a tenant in the multi-tenant system
type Tenant struct {
	// Identity
	ID          string    `yaml:"id" json:"id"`
	DisplayName string    `yaml:"displayName" json:"displayName"`
	Email       string    `yaml:"email" json:"email"`
	Status      Status    `yaml:"status" json:"status"`
	Created     time.Time `yaml:"created" json:"created"`
	Updated     time.Time `yaml:"updated" json:"updated"`
	
	// Credentials
	Credentials Credentials `yaml:"credentials" json:"credentials"`
	
	// Storage configuration
	Storage StorageConfig `yaml:"storage" json:"storage"`
	
	// Lock configuration
	Locks LockConfig `yaml:"locks" json:"locks"`
	
	// AWS configuration
	AWS AWSConfig `yaml:"aws,omitempty" json:"aws,omitempty"`
	
	// Limits and quotas
	Limits Limits `yaml:"limits" json:"limits"`
	
	// Metadata
	Metadata map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// Status represents the tenant status
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDeleted   Status = "deleted"
)

// Credentials stores the tenant's authentication credentials
type Credentials struct {
	Hash         string    `yaml:"hash" json:"hash"`
	Algorithm    string    `yaml:"algorithm" json:"algorithm"`
	Rotations    int       `yaml:"rotations" json:"rotations"`
	LastRotated  *time.Time `yaml:"lastRotated,omitempty" json:"lastRotated,omitempty"`
}

// StorageConfig defines where tenant state is stored
type StorageConfig struct {
	Prefix  string `yaml:"prefix" json:"prefix"`
	Version string `yaml:"version" json:"version"`
	Path    string `yaml:"path" json:"path"`
}

// LockConfig defines how tenant locks are namespaced
type LockConfig struct {
	Prefix string `yaml:"prefix" json:"prefix"`
}

// AWSConfig stores AWS-specific configuration
type AWSConfig struct {
	AccountID string `yaml:"accountId,omitempty" json:"accountId,omitempty"`
	Region    string `yaml:"region,omitempty" json:"region,omitempty"`
}

// Limits defines resource limits for the tenant
type Limits struct {
	CostTracking     bool `yaml:"costTracking" json:"costTracking"`
	MonthlyCostLimit int  `yaml:"monthlyCostLimit" json:"monthlyCostLimit"` // USD
	MaxStacks        int  `yaml:"maxStacks" json:"maxStacks"`
	MaxServices      int  `yaml:"maxServices" json:"maxServices"`
}

// Registry represents the tenants.yaml file structure
type Registry struct {
	Version  string           `yaml:"version" json:"version"`
	Metadata RegistryMetadata `yaml:"metadata" json:"metadata"`
	Config   RegistryConfig   `yaml:"config" json:"config"`
	Tenants  []Tenant         `yaml:"tenants" json:"tenants"`
}

// RegistryMetadata contains registry-level metadata
type RegistryMetadata struct {
	Created time.Time `yaml:"created" json:"created"`
	Updated time.Time `yaml:"updated" json:"updated"`
	Bucket  string    `yaml:"bucket" json:"bucket"`
	Region  string    `yaml:"region" json:"region"`
}

// RegistryConfig contains registry-level configuration
type RegistryConfig struct {
	LockTable      string `yaml:"lockTable" json:"lockTable"`
	DefaultVersion string `yaml:"defaultVersion" json:"defaultVersion"`
}

// Session represents an authenticated session
type Session struct {
	Mode          SessionMode `yaml:"mode" json:"mode"`
	Tenant        *TenantInfo `yaml:"tenant,omitempty" json:"tenant,omitempty"`
	Backend       *BackendConfig `yaml:"backend,omitempty" json:"backend,omitempty"`
	Locks         *LocksConfig `yaml:"locks,omitempty" json:"locks,omitempty"`
	AWS           *AWSConfig `yaml:"aws,omitempty" json:"aws,omitempty"`
	Authenticated time.Time `yaml:"authenticated" json:"authenticated"`
	Expires       time.Time `yaml:"expires" json:"expires"`
}

// SessionMode defines the type of session
type SessionMode string

const (
	ModeAdmin  SessionMode = "admin"
	ModeTenant SessionMode = "tenant"
)

// TenantInfo contains basic tenant information for sessions
type TenantInfo struct {
	ID          string `yaml:"id" json:"id"`
	DisplayName string `yaml:"displayName" json:"displayName"`
	Version     string `yaml:"version" json:"version"`
}

// BackendConfig contains S3 backend configuration
type BackendConfig struct {
	Type   string `yaml:"type" json:"type"`
	Bucket string `yaml:"bucket" json:"bucket"`
	Region string `yaml:"region" json:"region"`
	Prefix string `yaml:"prefix" json:"prefix"`
}

// LocksConfig contains DynamoDB lock configuration
type LocksConfig struct {
	Type   string `yaml:"type" json:"type"`
	Table  string `yaml:"table" json:"table"`
	Region string `yaml:"region" json:"region"`
	Prefix string `yaml:"prefix" json:"prefix"`
}

// CreateTenantRequest represents a request to create a new tenant
type CreateTenantRequest struct {
	Name             string
	DisplayName      string
	Email            string
	AWSAccountID     string
	Version          string
	CostTracking     bool
	MonthlyCostLimit int
	MaxStacks        int
	MaxServices      int
	Metadata         map[string]string
}

// TenantCredentials represents generated tenant credentials
type TenantCredentials struct {
	TenantID string
	Secret   string // Plain text, shown once
	Hash     string // Bcrypt hash, stored
}

