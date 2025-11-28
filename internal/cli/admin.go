package cli

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/panka/pkg/config"
	"github.com/yourusername/panka/pkg/tenant"
	"golang.org/x/term"
)

// loadAdminConfig loads the configuration from .panka.yaml
func loadAdminConfig() (*config.Config, error) {
	// Initialize viper if not already done
	if viper.ConfigFileUsed() == "" {
		viper.SetConfigName(".panka")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
		
		if err := viper.ReadInConfig(); err != nil {
			return nil, err
		}
	}
	
	cfg := &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}
	
	return cfg, nil
}

// adminCmd represents the admin command
var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Admin operations for managing tenants",
	Long: `Admin operations for platform administrators.

Admin commands allow you to:
  • Login as platform administrator
  • Create and manage tenants
  • View all tenants and their usage
  • Rotate tenant credentials
  • Suspend or activate tenants`,
}

// adminLoginCmd represents the admin login command
var adminLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login as platform administrator",
	Long: `Login as platform administrator to manage tenants.

This command will:
  • Read backend config from .panka.yaml
  • Prompt for admin password
  • Validate credentials
  • Create admin session
  
After login, you can use tenant management commands.

Note: The S3 bucket and region are configured in your .panka.yaml file.
Make sure to set backend.bucket and backend.region before logging in.`,
	RunE: runAdminLogin,
}

// adminLogoutCmd represents the admin logout command
var adminLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from admin session",
	Long:  `Logout from admin session and clear stored credentials.`,
	RunE:  runAdminLogout,
}

// adminSessionCmd shows current admin session
var adminSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Show current admin session",
	Long:  `Display information about the current admin session.`,
	RunE:  runAdminSession,
}

func init() {
	rootCmd.AddCommand(adminCmd)
	adminCmd.AddCommand(adminLoginCmd)
	adminCmd.AddCommand(adminLogoutCmd)
	adminCmd.AddCommand(adminSessionCmd)
}

func runAdminLogin(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n👤 Admin Authentication")
	cyan.Println(strings.Repeat("─", 50))

	// Load backend config from .panka.yaml
	cfg, err := loadAdminConfig()
	if err != nil {
		red.Printf("\n✗ Failed to load config: %v\n", err)
		yellow.Println("\nMake sure .panka.yaml exists with backend configuration:")
		yellow.Println("  backend:")
		yellow.Println("    bucket: your-panka-state-bucket")
		yellow.Println("    region: us-east-1")
		return err
	}

	adminBucket := cfg.Backend.Bucket
	adminRegion := cfg.Backend.Region

	if adminBucket == "" {
		red.Println("\n✗ Backend bucket not configured")
		yellow.Println("\nPlease set backend.bucket in .panka.yaml:")
		yellow.Println("  backend:")
		yellow.Println("    bucket: your-panka-state-bucket")
		yellow.Println("    region: us-east-1")
		return fmt.Errorf("backend bucket not configured")
	}

	if adminRegion == "" {
		adminRegion = "us-east-1" // Default
	}

	// Show which backend we're using
	fmt.Printf("\n📦 Using backend: s3://%s (region: %s)\n", adminBucket, adminRegion)

	// Prompt for admin password (hidden input)
	fmt.Print("\n? Admin Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // New line after password input
	
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	
	password := strings.TrimSpace(string(passwordBytes))
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Validate credentials
	fmt.Println("\nValidating credentials...")
	
	// In production, this would:
	// 1. Load admin password hash from AWS Secrets Manager
	// 2. Verify password against hash
	// 3. Load tenants.yaml from S3
	// 4. Create admin session
	
	// For now, we'll accept any non-empty password and create a session
	yellow.Println("⚠️  Note: Using development mode authentication")
	yellow.Println("   In production, this would verify against AWS Secrets Manager")
	
	// Create session
	sessionMgr := tenant.NewSessionManager()
	if err := sessionMgr.SaveAdminSession(adminBucket, adminRegion); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	
	green.Println("\n✓ Admin authentication successful")
	
	// Show session info
	cyan.Println("\n" + strings.Repeat("─", 50))
	fmt.Printf("Session saved to: %s/.panka/admin-session\n", os.Getenv("HOME"))
	fmt.Println("Mode: ADMIN")
	fmt.Printf("Bucket: %s\n", adminBucket)
	fmt.Printf("Region: %s\n", adminRegion)
	
	cyan.Println("\nAvailable Admin Commands:")
	fmt.Println("  panka tenant init       - Create new tenant")
	fmt.Println("  panka tenant list       - List all tenants")
	fmt.Println("  panka tenant show       - Show tenant details")
	fmt.Println("  panka tenant rotate     - Rotate tenant credentials")
	fmt.Println("  panka tenant suspend    - Suspend tenant")
	fmt.Println("  panka tenant activate   - Activate tenant")
	fmt.Println("  panka admin logout      - Logout from admin mode")
	
	cyan.Println(strings.Repeat("─", 50))
	
	green.Println("\n✨ You are now logged in as administrator!")
	
	return nil
}

func runAdminLogout(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan)

	cyan.Println("\n👋 Logging out from admin session...")

	sessionMgr := tenant.NewSessionManager()
	
	// Check if admin session exists
	session, err := sessionMgr.LoadSession()
	if err != nil {
		return fmt.Errorf("no active session found")
	}
	
	if session.Mode != tenant.ModeAdmin {
		return fmt.Errorf("not logged in as admin")
	}
	
	// Clear session
	if err := sessionMgr.ClearSession(); err != nil {
		return fmt.Errorf("failed to clear session: %w", err)
	}
	
	green.Println("✓ Logged out successfully")
	
	return nil
}

func runAdminSession(cmd *cobra.Command, args []string) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	sessionMgr := tenant.NewSessionManager()
	
	session, err := sessionMgr.RequireAdminSession()
	if err != nil {
		return err
	}
	
	cyan.Println("\n📋 Admin Session")
	cyan.Println(strings.Repeat("─", 50))
	
	fmt.Println("\nMode:      ", "ADMIN")
	fmt.Println("Bucket:    ", session.Backend.Bucket)
	fmt.Println("Region:    ", session.Backend.Region)
	fmt.Println("Authenticated:", session.Authenticated.Format("2006-01-02 15:04:05"))
	fmt.Println("Expires:   ", session.Expires.Format("2006-01-02 15:04:05"))
	
	// Time remaining
	remaining := session.Expires.Sub(time.Now())
	if remaining > 0 {
		hours := int(remaining.Hours())
		minutes := int(remaining.Minutes()) % 60
		green.Printf("\nSession valid for: %d hours %d minutes\n", hours, minutes)
	} else {
		yellow.Println("\n⚠️  Session expired - please login again")
	}
	
	cyan.Println(strings.Repeat("─", 50))
	
	return nil
}

