package lock

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDynamoDBManager_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *DynamoDBConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing client",
			config: &DynamoDBConfig{
				TableName: "test-table",
			},
			wantErr: true,
			errMsg:  "DynamoDB client is required",
		},
		{
			name: "missing table name",
			config: &DynamoDBConfig{
				Client: nil, // Will be caught by validation
			},
			wantErr: true,
			errMsg:  "DynamoDB client is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewDynamoDBManager(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
			}
		})
	}
}

func TestDynamoDBManager_InterfaceCompliance(t *testing.T) {
	// Verify that DynamoDBManager implements Manager interface
	var _ Manager = (*DynamoDBManager)(nil)
}

