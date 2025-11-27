package lock

import (
	"time"
)

// Lock represents a distributed lock
type Lock struct {
	// Key is the lock identifier
	Key string

	// ID is the unique lock instance identifier
	ID string

	// Owner identifies who acquired the lock
	Owner string

	// AcquiredAt is when the lock was acquired
	AcquiredAt time.Time

	// ExpiresAt is when the lock will expire
	ExpiresAt time.Time

	// TTL is the lock time-to-live in seconds
	TTL int64

	// Metadata stores additional lock information
	Metadata map[string]string
}

// LockInfo contains information about an existing lock
type LockInfo struct {
	Key        string
	Owner      string
	AcquiredAt time.Time
	ExpiresAt  time.Time
	Age        time.Duration
	IsExpired  bool
	Metadata   map[string]string
}

// IsExpired checks if the lock has expired
func (l *Lock) IsExpired() bool {
	return time.Now().After(l.ExpiresAt)
}

// Age returns how long the lock has been held
func (l *Lock) Age() time.Duration {
	return time.Since(l.AcquiredAt)
}

// TimeUntilExpiry returns time remaining until lock expires
func (l *Lock) TimeUntilExpiry() time.Duration {
	remaining := time.Until(l.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// NewLock creates a new lock instance
func NewLock(key, id, owner string, ttl int64) *Lock {
	now := time.Now()
	return &Lock{
		Key:        key,
		ID:         id,
		Owner:      owner,
		AcquiredAt: now,
		ExpiresAt:  now.Add(time.Duration(ttl) * time.Second),
		TTL:        ttl,
		Metadata:   make(map[string]string),
	}
}

// ToLockInfo converts a Lock to LockInfo
func (l *Lock) ToLockInfo() *LockInfo {
	return &LockInfo{
		Key:        l.Key,
		Owner:      l.Owner,
		AcquiredAt: l.AcquiredAt,
		ExpiresAt:  l.ExpiresAt,
		Age:        l.Age(),
		IsExpired:  l.IsExpired(),
		Metadata:   l.Metadata,
	}
}

