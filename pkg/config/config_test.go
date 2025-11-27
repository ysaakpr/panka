package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)
	
	assert.Equal(t, "v1", cfg.Version)
	assert.Equal(t, "s3", cfg.Backend.Type)
	assert.Equal(t, "dynamodb", cfg.Locks.Type)
	assert.Equal(t, "us-east-1", cfg.AWS.Region)
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	configData := `version: v1
backend:
  type: s3
  region: us-west-2
  bucket: test-bucket
  prefix: test-prefix
locks:
  type: dynamodb
  region: us-west-2
  table: test-table
aws:
  profile: test-profile
  region: us-west-2
`
	
	err := os.WriteFile(configPath, []byte(configData), 0600)
	require.NoError(t, err)
	
	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	
	assert.Equal(t, "s3", cfg.Backend.Type)
	assert.Equal(t, "us-west-2", cfg.Backend.Region)
	assert.Equal(t, "test-bucket", cfg.Backend.Bucket)
	assert.Equal(t, "test-prefix", cfg.Backend.Prefix)
	assert.Equal(t, "dynamodb", cfg.Locks.Type)
	assert.Equal(t, "test-table", cfg.Locks.Table)
	assert.Equal(t, "test-profile", cfg.AWS.Profile)
}

func TestLoadFromEnv(t *testing.T) {
	// Save original env vars
	originalVars := make(map[string]string)
	envVars := []string{
		"PANKA_BACKEND_TYPE",
		"PANKA_BACKEND_BUCKET",
		"PANKA_BACKEND_REGION",
		"PANKA_LOCK_TABLE",
		"AWS_REGION",
	}
	
	for _, key := range envVars {
		originalVars[key] = os.Getenv(key)
	}
	
	// Cleanup
	defer func() {
		for key, val := range originalVars {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()
	
	// Set test env vars
	os.Setenv("PANKA_BACKEND_TYPE", "s3")
	os.Setenv("PANKA_BACKEND_BUCKET", "env-bucket")
	os.Setenv("PANKA_BACKEND_REGION", "eu-west-1")
	os.Setenv("PANKA_LOCK_TABLE", "env-table")
	os.Setenv("AWS_REGION", "eu-west-1")
	
	cfg := DefaultConfig()
	loadFromEnv(cfg)
	
	assert.Equal(t, "s3", cfg.Backend.Type)
	assert.Equal(t, "env-bucket", cfg.Backend.Bucket)
	assert.Equal(t, "eu-west-1", cfg.Backend.Region)
	assert.Equal(t, "env-table", cfg.Locks.Table)
	assert.Equal(t, "eu-west-1", cfg.AWS.Region)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid s3 backend",
			config: &Config{
				Backend: BackendConfig{
					Type:   "s3",
					Bucket: "test-bucket",
					Region: "us-east-1",
				},
				Locks: LocksConfig{
					Type:   "dynamodb",
					Table:  "test-table",
					Region: "us-east-1",
				},
				AWS: AWSConfig{
					Region: "us-east-1",
				},
			},
			wantErr: false,
		},
		{
			name: "missing s3 bucket",
			config: &Config{
				Backend: BackendConfig{
					Type:   "s3",
					Region: "us-east-1",
				},
				Locks: LocksConfig{
					Type:   "dynamodb",
					Table:  "test-table",
					Region: "us-east-1",
				},
				AWS: AWSConfig{
					Region: "us-east-1",
				},
			},
			wantErr: true,
			errMsg:  "backend bucket is required",
		},
		{
			name: "missing dynamodb table",
			config: &Config{
				Backend: BackendConfig{
					Type:   "s3",
					Bucket: "test-bucket",
					Region: "us-east-1",
				},
				Locks: LocksConfig{
					Type:   "dynamodb",
					Region: "us-east-1",
				},
				AWS: AWSConfig{
					Region: "us-east-1",
				},
			},
			wantErr: true,
			errMsg:  "locks table is required",
		},
		{
			name: "invalid backend type",
			config: &Config{
				Backend: BackendConfig{
					Type: "invalid",
				},
				Locks: LocksConfig{
					Type:   "dynamodb",
					Table:  "test-table",
					Region: "us-east-1",
				},
				AWS: AWSConfig{
					Region: "us-east-1",
				},
			},
			wantErr: true,
			errMsg:  "invalid backend type",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	cfg := &Config{
		Version: "v1",
		Backend: BackendConfig{
			Type:   "s3",
			Bucket: "test-bucket",
			Region: "us-east-1",
		},
		Locks: LocksConfig{
			Type:   "dynamodb",
			Table:  "test-table",
			Region: "us-east-1",
		},
		AWS: AWSConfig{
			Region: "us-east-1",
		},
	}
	
	err := cfg.Save(configPath)
	require.NoError(t, err)
	
	// Verify file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)
	
	// Load it back
	loaded, err := Load(configPath)
	require.NoError(t, err)
	
	assert.Equal(t, cfg.Backend.Bucket, loaded.Backend.Bucket)
	assert.Equal(t, cfg.Locks.Table, loaded.Locks.Table)
}

func TestIsTenantMode(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name: "tenant mode enabled",
			config: &Config{
				Tenant: &TenantConfig{
					Name: "test-tenant",
				},
			},
			want: true,
		},
		{
			name: "tenant mode disabled - nil tenant",
			config: &Config{
				Tenant: nil,
			},
			want: false,
		},
		{
			name: "tenant mode disabled - empty name",
			config: &Config{
				Tenant: &TenantConfig{
					Name: "",
				},
			},
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config.IsTenantMode())
		})
	}
}

func TestGetStatePrefix(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   string
	}{
		{
			name: "tenant mode",
			config: &Config{
				Backend: BackendConfig{
					Prefix: "",
				},
				Tenant: &TenantConfig{
					Name: "test-tenant",
				},
			},
			want: "tenants/test-tenant/v1",
		},
		{
			name: "tenant mode with custom prefix",
			config: &Config{
				Backend: BackendConfig{
					Prefix: "custom",
				},
				Tenant: &TenantConfig{
					Name: "test-tenant",
				},
			},
			want: "custom/tenants/test-tenant/v1",
		},
		{
			name: "non-tenant mode default",
			config: &Config{
				Backend: BackendConfig{
					Prefix: "",
				},
			},
			want: "stacks",
		},
		{
			name: "non-tenant mode with custom prefix",
			config: &Config{
				Backend: BackendConfig{
					Prefix: "my-prefix",
				},
			},
			want: "my-prefix",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetStatePrefix()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetLockKeyPrefix(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   string
	}{
		{
			name: "tenant mode",
			config: &Config{
				Tenant: &TenantConfig{
					Name: "test-tenant",
				},
			},
			want: "tenant:test-tenant:",
		},
		{
			name: "non-tenant mode",
			config: &Config{
				Tenant: nil,
			},
			want: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetLockKeyPrefix()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMergeConfig(t *testing.T) {
	dst := DefaultConfig()
	dst.Backend.Bucket = "original-bucket"
	
	src := &Config{
		Backend: BackendConfig{
			Bucket: "new-bucket",
			Prefix: "new-prefix",
		},
		Locks: LocksConfig{
			Table: "new-table",
		},
	}
	
	mergeConfig(dst, src)
	
	assert.Equal(t, "new-bucket", dst.Backend.Bucket)
	assert.Equal(t, "new-prefix", dst.Backend.Prefix)
	assert.Equal(t, "new-table", dst.Locks.Table)
	// Original values should remain if not in source
	assert.Equal(t, "s3", dst.Backend.Type)
}

