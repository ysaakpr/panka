package lock

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	assert.NotNil(t, cfg)
	assert.Equal(t, 5*time.Minute, cfg.DefaultTTL)
	assert.Equal(t, 30*time.Second, cfg.HeartbeatInterval)
	assert.Equal(t, 3, cfg.RetryAttempts)
	assert.Equal(t, 1*time.Second, cfg.RetryDelay)
}

func TestErrors(t *testing.T) {
	// Verify all error types are defined and unique
	errorTypes := []error{
		ErrLockAlreadyHeld,
		ErrLockNotFound,
		ErrLockExpired,
		ErrInvalidLockID,
		ErrLockNotHeld,
	}

	// Check they're all different
	for i, err1 := range errorTypes {
		assert.NotNil(t, err1)
		for j, err2 := range errorTypes {
			if i != j {
				assert.NotEqual(t, err1, err2, "errors should be unique")
				assert.False(t, errors.Is(err1, err2), "errors should not wrap each other")
			}
		}
	}
}

func TestLockErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrLockAlreadyHeld",
			err:  ErrLockAlreadyHeld,
			msg:  "lock is already held",
		},
		{
			name: "ErrLockNotFound",
			err:  ErrLockNotFound,
			msg:  "lock not found",
		},
		{
			name: "ErrLockExpired",
			err:  ErrLockExpired,
			msg:  "lock has expired",
		},
		{
			name: "ErrInvalidLockID",
			err:  ErrInvalidLockID,
			msg:  "invalid lock ID",
		},
		{
			name: "ErrLockNotHeld",
			err:  ErrLockNotHeld,
			msg:  "lock is not held",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.msg, tt.err.Error())
		})
	}
}

