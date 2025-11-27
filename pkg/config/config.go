package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the panka configuration
type Config struct {
	Version string         `yaml:"version"`
	Backend BackendConfig  `yaml:"backend"`
	Locks   LocksConfig    `yaml:"locks"`
	AWS     AWSConfig      `yaml:"aws"`
	Tenant  *TenantConfig  `yaml:"tenant,omitempty"`
}

// BackendConfig configures the state backend
type BackendConfig struct {
	Type   string `yaml:"type"`   // s3, local
	Region string `yaml:"region"`
	Bucket string `yaml:"bucket"` // S3 bucket name
	Prefix string `yaml:"prefix,omitempty"`
}

// LocksConfig configures the distributed locking system
type LocksConfig struct {
	Type   string `yaml:"type"`  // dynamodb, local
	Region string `yaml:"region"`
	Table  string `yaml:"table"` // DynamoDB table name
}

// AWSConfig configures AWS settings
type AWSConfig struct {
	Profile string `yaml:"profile,omitempty"`
	Region  string `yaml:"region"`
}

// TenantConfig configures tenant-specific settings
type TenantConfig struct {
	Name   string `yaml:"name"`
	Secret string `yaml:"secret,omitempty"` // Not stored in file
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Version: "v1",
		Backend: BackendConfig{
			Type:   "s3",
			Region: "us-east-1",
		},
		Locks: LocksConfig{
			Type:   "dynamodb",
			Region: "us-east-1",
		},
		AWS: AWSConfig{
			Region: "us-east-1",
		},
	}
}

// Load loads configuration from file, environment, and flags
// Priority: flags > environment > file > defaults
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Load from file if it exists
	if configPath != "" {
		fileCfg, err := loadFromFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
		// Merge file config into defaults
		mergeConfig(cfg, fileCfg)
	}

	// Override with environment variables
	loadFromEnv(cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// loadFromFile loads configuration from a YAML file
func loadFromFile(path string) (*Config, error) {
	// Expand home directory
	if path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist, return nil (not an error)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(cfg *Config) {
	// Backend settings
	if v := os.Getenv("PANKA_BACKEND_TYPE"); v != "" {
		cfg.Backend.Type = v
	}
	if v := os.Getenv("PANKA_BACKEND_REGION"); v != "" {
		cfg.Backend.Region = v
	}
	if v := os.Getenv("PANKA_BACKEND_BUCKET"); v != "" {
		cfg.Backend.Bucket = v
	}
	if v := os.Getenv("PANKA_BACKEND_PREFIX"); v != "" {
		cfg.Backend.Prefix = v
	}

	// Locks settings
	if v := os.Getenv("PANKA_LOCK_TYPE"); v != "" {
		cfg.Locks.Type = v
	}
	if v := os.Getenv("PANKA_LOCK_REGION"); v != "" {
		cfg.Locks.Region = v
	}
	if v := os.Getenv("PANKA_LOCK_TABLE"); v != "" {
		cfg.Locks.Table = v
	}

	// AWS settings
	if v := os.Getenv("AWS_PROFILE"); v != "" {
		cfg.AWS.Profile = v
	}
	if v := os.Getenv("AWS_REGION"); v != "" {
		cfg.AWS.Region = v
		// Also update backend and locks regions if not set
		if cfg.Backend.Region == "" {
			cfg.Backend.Region = v
		}
		if cfg.Locks.Region == "" {
			cfg.Locks.Region = v
		}
	}

	// Tenant settings
	if v := os.Getenv("PANKA_TENANT_NAME"); v != "" {
		if cfg.Tenant == nil {
			cfg.Tenant = &TenantConfig{}
		}
		cfg.Tenant.Name = v
	}
	if v := os.Getenv("PANKA_TENANT_SECRET"); v != "" {
		if cfg.Tenant == nil {
			cfg.Tenant = &TenantConfig{}
		}
		cfg.Tenant.Secret = v
	}
}

// mergeConfig merges source config into destination
func mergeConfig(dst, src *Config) {
	if src == nil {
		return
	}

	if src.Version != "" {
		dst.Version = src.Version
	}

	// Backend
	if src.Backend.Type != "" {
		dst.Backend.Type = src.Backend.Type
	}
	if src.Backend.Region != "" {
		dst.Backend.Region = src.Backend.Region
	}
	if src.Backend.Bucket != "" {
		dst.Backend.Bucket = src.Backend.Bucket
	}
	if src.Backend.Prefix != "" {
		dst.Backend.Prefix = src.Backend.Prefix
	}

	// Locks
	if src.Locks.Type != "" {
		dst.Locks.Type = src.Locks.Type
	}
	if src.Locks.Region != "" {
		dst.Locks.Region = src.Locks.Region
	}
	if src.Locks.Table != "" {
		dst.Locks.Table = src.Locks.Table
	}

	// AWS
	if src.AWS.Profile != "" {
		dst.AWS.Profile = src.AWS.Profile
	}
	if src.AWS.Region != "" {
		dst.AWS.Region = src.AWS.Region
	}

	// Tenant
	if src.Tenant != nil {
		if dst.Tenant == nil {
			dst.Tenant = &TenantConfig{}
		}
		if src.Tenant.Name != "" {
			dst.Tenant.Name = src.Tenant.Name
		}
		if src.Tenant.Secret != "" {
			dst.Tenant.Secret = src.Tenant.Secret
		}
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate backend
	if c.Backend.Type != "s3" && c.Backend.Type != "local" {
		return fmt.Errorf("invalid backend type: %s (must be 's3' or 'local')", c.Backend.Type)
	}
	if c.Backend.Type == "s3" {
		if c.Backend.Bucket == "" {
			return fmt.Errorf("backend bucket is required for s3 backend")
		}
		if c.Backend.Region == "" {
			return fmt.Errorf("backend region is required for s3 backend")
		}
	}

	// Validate locks
	if c.Locks.Type != "dynamodb" && c.Locks.Type != "local" {
		return fmt.Errorf("invalid locks type: %s (must be 'dynamodb' or 'local')", c.Locks.Type)
	}
	if c.Locks.Type == "dynamodb" {
		if c.Locks.Table == "" {
			return fmt.Errorf("locks table is required for dynamodb locks")
		}
		if c.Locks.Region == "" {
			return fmt.Errorf("locks region is required for dynamodb locks")
		}
	}

	// Validate AWS
	if c.AWS.Region == "" {
		return fmt.Errorf("AWS region is required")
	}

	return nil
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	// Expand home directory
	if path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DefaultConfigPath returns the default configuration file path
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".panka", "config.yaml")
}

// IsTenantMode returns true if running in tenant mode
func (c *Config) IsTenantMode() bool {
	return c.Tenant != nil && c.Tenant.Name != ""
}

// GetStatePrefix returns the S3 prefix for state storage
func (c *Config) GetStatePrefix() string {
	if c.IsTenantMode() {
		// Tenant mode: tenants/{tenant-name}/v1/
		if c.Backend.Prefix != "" {
			return filepath.Join(c.Backend.Prefix, "tenants", c.Tenant.Name, "v1")
		}
		return filepath.Join("tenants", c.Tenant.Name, "v1")
	}
	
	// Non-tenant mode: use configured prefix or default
	if c.Backend.Prefix != "" {
		return c.Backend.Prefix
	}
	return "stacks"
}

// GetLockKeyPrefix returns the prefix for lock keys
func (c *Config) GetLockKeyPrefix() string {
	if c.IsTenantMode() {
		return fmt.Sprintf("tenant:%s:", c.Tenant.Name)
	}
	return ""
}

