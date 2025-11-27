package lock

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrLockAlreadyHeld indicates the lock is already held by another process
	ErrLockAlreadyHeld = errors.New("lock is already held")

	// ErrLockNotFound indicates the lock does not exist
	ErrLockNotFound = errors.New("lock not found")

	// ErrLockExpired indicates the lock has expired
	ErrLockExpired = errors.New("lock has expired")

	// ErrInvalidLockID indicates the lock ID is invalid or mismatched
	ErrInvalidLockID = errors.New("invalid lock ID")

	// ErrLockNotHeld indicates the lock is not currently held
	ErrLockNotHeld = errors.New("lock is not held")
)

// Manager defines the interface for distributed lock management
type Manager interface {
	// Acquire attempts to acquire a lock with the given key
	// Returns the lock if successful, or an error if the lock is already held
	Acquire(ctx context.Context, key string, ttl time.Duration, owner string) (*Lock, error)

	// Refresh refreshes an existing lock, extending its TTL
	// This is used for heartbeat/keep-alive
	Refresh(ctx context.Context, lock *Lock) error

	// Release releases a lock
	Release(ctx context.Context, lock *Lock) error

	// ForceRelease forcibly releases a lock (admin operation)
	// This should be used with caution
	ForceRelease(ctx context.Context, key string) error

	// Get retrieves information about a lock
	Get(ctx context.Context, key string) (*LockInfo, error)

	// List lists all locks with the given prefix
	List(ctx context.Context, prefix string) ([]*LockInfo, error)

	// Close closes the lock manager
	Close() error
}

// Config holds common lock manager configuration
type Config struct {
	// DefaultTTL is the default lock TTL in seconds
	DefaultTTL time.Duration

	// HeartbeatInterval is how often to refresh locks
	HeartbeatInterval time.Duration

	// RetryAttempts is the number of times to retry lock acquisition
	RetryAttempts int

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration
}

// DefaultConfig returns default lock manager configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultTTL:        5 * time.Minute,
		HeartbeatInterval: 30 * time.Second,
		RetryAttempts:     3,
		RetryDelay:        1 * time.Second,
	}
}

