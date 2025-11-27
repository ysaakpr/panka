package lock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLock(t *testing.T) {
	key := "test-key"
	id := "lock-id-123"
	owner := "test-owner"
	ttl := int64(300)

	lock := NewLock(key, id, owner, ttl)

	require.NotNil(t, lock)
	assert.Equal(t, key, lock.Key)
	assert.Equal(t, id, lock.ID)
	assert.Equal(t, owner, lock.Owner)
	assert.Equal(t, ttl, lock.TTL)
	assert.False(t, lock.AcquiredAt.IsZero())
	assert.False(t, lock.ExpiresAt.IsZero())
	assert.NotNil(t, lock.Metadata)
	
	// Verify expiry is approximately TTL seconds in future
	expectedExpiry := time.Now().Add(time.Duration(ttl) * time.Second)
	assert.WithinDuration(t, expectedExpiry, lock.ExpiresAt, 2*time.Second)
}

func TestLock_IsExpired(t *testing.T) {
	tests := []struct {
		name    string
		lock    *Lock
		want    bool
	}{
		{
			name: "not expired",
			lock: &Lock{
				ExpiresAt: time.Now().Add(5 * time.Minute),
			},
			want: false,
		},
		{
			name: "expired",
			lock: &Lock{
				ExpiresAt: time.Now().Add(-5 * time.Minute),
			},
			want: true,
		},
		{
			name: "just expired",
			lock: &Lock{
				ExpiresAt: time.Now().Add(-1 * time.Second),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.lock.IsExpired()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLock_Age(t *testing.T) {
	acquiredAt := time.Now().Add(-10 * time.Minute)
	lock := &Lock{
		AcquiredAt: acquiredAt,
	}

	age := lock.Age()
	
	// Age should be approximately 10 minutes
	assert.InDelta(t, 10*time.Minute, age, float64(2*time.Second))
}

func TestLock_TimeUntilExpiry(t *testing.T) {
	tests := []struct {
		name     string
		lock     *Lock
		wantZero bool
	}{
		{
			name: "time remaining",
			lock: &Lock{
				ExpiresAt: time.Now().Add(5 * time.Minute),
			},
			wantZero: false,
		},
		{
			name: "expired returns zero",
			lock: &Lock{
				ExpiresAt: time.Now().Add(-5 * time.Minute),
			},
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remaining := tt.lock.TimeUntilExpiry()
			
			if tt.wantZero {
				assert.Equal(t, time.Duration(0), remaining)
			} else {
				assert.Greater(t, remaining, time.Duration(0))
				assert.InDelta(t, 5*time.Minute, remaining, float64(2*time.Second))
			}
		})
	}
}

func TestLock_ToLockInfo(t *testing.T) {
	now := time.Now()
	lock := &Lock{
		Key:        "test-key",
		ID:         "lock-id-123",
		Owner:      "test-owner",
		AcquiredAt: now.Add(-5 * time.Minute),
		ExpiresAt:  now.Add(5 * time.Minute),
		TTL:        600,
		Metadata: map[string]string{
			"stack": "test-stack",
			"env":   "production",
		},
	}

	info := lock.ToLockInfo()

	require.NotNil(t, info)
	assert.Equal(t, lock.Key, info.Key)
	assert.Equal(t, lock.Owner, info.Owner)
	assert.Equal(t, lock.AcquiredAt, info.AcquiredAt)
	assert.Equal(t, lock.ExpiresAt, info.ExpiresAt)
	assert.False(t, info.IsExpired)
	assert.Greater(t, info.Age, time.Duration(0))
	assert.Equal(t, lock.Metadata, info.Metadata)
}

func TestLockInfo(t *testing.T) {
	now := time.Now()
	info := &LockInfo{
		Key:        "test-key",
		Owner:      "test-owner",
		AcquiredAt: now.Add(-10 * time.Minute),
		ExpiresAt:  now.Add(5 * time.Minute),
		Age:        10 * time.Minute,
		IsExpired:  false,
		Metadata: map[string]string{
			"test": "value",
		},
	}

	assert.Equal(t, "test-key", info.Key)
	assert.Equal(t, "test-owner", info.Owner)
	assert.False(t, info.IsExpired)
	assert.Equal(t, 10*time.Minute, info.Age)
	assert.NotNil(t, info.Metadata)
}

