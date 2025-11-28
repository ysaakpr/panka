package tenant

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// AdminSessionDuration is how long admin sessions last
	AdminSessionDuration = 8 * time.Hour
	
	// TenantSessionDuration is how long tenant sessions last
	TenantSessionDuration = 7 * 24 * time.Hour // 7 days
)

// SessionManager manages user sessions
type SessionManager struct {
	sessionDir string
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	home, _ := os.UserHomeDir()
	sessionDir := filepath.Join(home, ".panka")
	
	// Create directory if it doesn't exist
	os.MkdirAll(sessionDir, 0700)
	
	return &SessionManager{
		sessionDir: sessionDir,
	}
}

// SaveAdminSession saves an admin session
func (sm *SessionManager) SaveAdminSession(bucket, region string) error {
	session := &Session{
		Mode: ModeAdmin,
		Backend: &BackendConfig{
			Type:   "s3",
			Bucket: bucket,
			Region: region,
		},
		Authenticated: time.Now(),
		Expires:       time.Now().Add(AdminSessionDuration),
	}
	
	return sm.saveSession("admin-session", session)
}

// SaveTenantSession saves a tenant session
func (sm *SessionManager) SaveTenantSession(tenant *Tenant, bucket, region, lockTable string) error {
	session := &Session{
		Mode: ModeTenant,
		Tenant: &TenantInfo{
			ID:          tenant.ID,
			DisplayName: tenant.DisplayName,
			Version:     tenant.Storage.Version,
		},
		Backend: &BackendConfig{
			Type:   "s3",
			Bucket: bucket,
			Region: region,
			Prefix: tenant.Storage.Path,
		},
		Locks: &LocksConfig{
			Type:   "dynamodb",
			Table:  lockTable,
			Region: region,
			Prefix: tenant.Locks.Prefix,
		},
		AWS: &AWSConfig{
			AccountID: tenant.AWS.AccountID,
			Region:    tenant.AWS.Region,
		},
		Authenticated: time.Now(),
		Expires:       time.Now().Add(TenantSessionDuration),
	}
	
	return sm.saveSession("session", session)
}

// LoadSession loads the current session
func (sm *SessionManager) LoadSession() (*Session, error) {
	// Try tenant session first
	session, err := sm.loadSession("session")
	if err == nil && !sm.isExpired(session) {
		return session, nil
	}
	
	// Try admin session
	session, err = sm.loadSession("admin-session")
	if err == nil && !sm.isExpired(session) {
		return session, nil
	}
	
	return nil, fmt.Errorf("no valid session found")
}

// ClearSession removes the current session
func (sm *SessionManager) ClearSession() error {
	// Remove tenant session
	tenantPath := filepath.Join(sm.sessionDir, "session")
	os.Remove(tenantPath)
	
	// Remove admin session
	adminPath := filepath.Join(sm.sessionDir, "admin-session")
	os.Remove(adminPath)
	
	return nil
}

// IsAuthenticated checks if user is authenticated
func (sm *SessionManager) IsAuthenticated() bool {
	session, err := sm.LoadSession()
	return err == nil && !sm.isExpired(session)
}

// GetCurrentMode returns the current session mode
func (sm *SessionManager) GetCurrentMode() (SessionMode, error) {
	session, err := sm.LoadSession()
	if err != nil {
		return "", err
	}
	
	return session.Mode, nil
}

// RequireAdminSession ensures user is authenticated as admin
func (sm *SessionManager) RequireAdminSession() (*Session, error) {
	session, err := sm.loadSession("admin-session")
	if err != nil {
		return nil, fmt.Errorf("not logged in as admin (use 'panka admin login')")
	}
	
	if sm.isExpired(session) {
		return nil, fmt.Errorf("admin session expired (use 'panka admin login')")
	}
	
	return session, nil
}

// RequireTenantSession ensures user is authenticated as tenant
func (sm *SessionManager) RequireTenantSession() (*Session, error) {
	session, err := sm.loadSession("session")
	if err != nil {
		return nil, fmt.Errorf("not logged in (use 'panka login')")
	}
	
	if sm.isExpired(session) {
		return nil, fmt.Errorf("session expired (use 'panka login')")
	}
	
	return session, nil
}

// Private methods

func (sm *SessionManager) saveSession(filename string, session *Session) error {
	path := filepath.Join(sm.sessionDir, filename)
	
	data, err := yaml.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write session: %w", err)
	}
	
	return nil
}

func (sm *SessionManager) loadSession(filename string) (*Session, error) {
	path := filepath.Join(sm.sessionDir, filename)
	
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var session Session
	if err := yaml.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session: %w", err)
	}
	
	return &session, nil
}

func (sm *SessionManager) isExpired(session *Session) bool {
	return time.Now().After(session.Expires)
}

