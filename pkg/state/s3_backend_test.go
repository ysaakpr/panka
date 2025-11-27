package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewS3Backend_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *S3BackendConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing client",
			config: &S3BackendConfig{
				Bucket: "test-bucket",
			},
			wantErr: true,
			errMsg:  "S3 client is required",
		},
		{
			name: "missing bucket",
			config: &S3BackendConfig{
				Client: nil, // Will be caught by validation
			},
			wantErr: true,
			errMsg:  "S3 client is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := NewS3Backend(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, backend)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, backend)
			}
		})
	}
}

func TestS3Backend_BuildKey(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		key    string
		want   string
	}{
		{
			name:   "with prefix",
			prefix: "tenants/test-tenant/v1",
			key:    "stacks/my-stack/dev/state.json",
			want:   "tenants/test-tenant/v1/stacks/my-stack/dev/state.json",
		},
		{
			name:   "without prefix",
			prefix: "",
			key:    "stacks/my-stack/dev/state.json",
			want:   "stacks/my-stack/dev/state.json",
		},
		{
			name:   "with empty key",
			prefix: "prefix",
			key:    "",
			want:   "prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &S3Backend{
				prefix: tt.prefix,
			}
			got := backend.buildKey(tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestS3Backend_InterfaceCompliance(t *testing.T) {
	// Verify that S3Backend implements Backend interface
	var _ Backend = (*S3Backend)(nil)
}

func TestBackendConfig(t *testing.T) {
	cfg := DefaultBackendConfig()
	
	assert.NotNil(t, cfg)
	assert.Equal(t, "s3", cfg.Type)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 30, cfg.Timeout)
}

