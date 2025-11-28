package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/panka/pkg/config"
	"github.com/yourusername/panka/pkg/tenant"
	"golang.org/x/term"
)

// loadConfig loads the configuration from .panka.yaml
func loadConfig() (*config.Config, error) {
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

// loginCmd represents the login command for tenants
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login as a tenant",
	Long: `Login as a tenant to deploy and manage your stacks.

This command will:
  • Read backend config from .panka.yaml
  • Prompt for tenant credentials (ID and secret only)
  • Verify credentials against the registry
  • Create a tenant session
  
After login, you can use all stack management commands.

Note: The S3 bucket and region are configured in your .panka.yaml file.
Make sure to set backend.bucket and backend.region before logging in.`,
	RunE: runLogin,
}

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from current session",
	Long:  `Logout from current tenant session and clear stored credentials.`,
	RunE:  runLogout,
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n🔐 Tenant Authentication")
	cyan.Println(strings.Repeat("─", 50))

	// Load backend config from .panka.yaml
	cfg, err := loadConfig()
	if err != nil {
		red.Printf("\n✗ Failed to load config: %v\n", err)
		yellow.Println("\nMake sure .panka.yaml exists with backend configuration:")
		yellow.Println("  backend:")
		yellow.Println("    bucket: your-panka-state-bucket")
		yellow.Println("    region: us-east-1")
		return err
	}

	loginBucket := cfg.Backend.Bucket
	loginRegion := cfg.Backend.Region

	if loginBucket == "" {
		red.Println("\n✗ Backend bucket not configured")
		yellow.Println("\nPlease set backend.bucket in .panka.yaml:")
		yellow.Println("  backend:")
		yellow.Println("    bucket: your-panka-state-bucket")
		yellow.Println("    region: us-east-1")
		return fmt.Errorf("backend bucket not configured")
	}

	if loginRegion == "" {
		loginRegion = "us-east-1" // Default
	}

	// Show which backend we're using
	fmt.Printf("\n📦 Using backend: s3://%s (region: %s)\n", loginBucket, loginRegion)

	reader := bufio.NewReader(os.Stdin)

	// Prompt for tenant name
	fmt.Print("? Tenant Name: ")
	tenantNameInput, _ := reader.ReadString('\n')
	tenantName := strings.TrimSpace(tenantNameInput)
	
	if tenantName == "" {
		return fmt.Errorf("tenant name cannot be empty")
	}

	// Prompt for tenant secret (hidden input)
	fmt.Print("? Tenant Secret: ")
	secretBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // New line after password input
	
	if err != nil {
		return fmt.Errorf("failed to read secret: %w", err)
	}
	
	secret := strings.TrimSpace(string(secretBytes))
	if secret == "" {
		return fmt.Errorf("secret cannot be empty")
	}

	// Authenticate
	fmt.Println("\nAuthenticating...")
	fmt.Println("├── Loading tenants.yaml...")
	
	// Create tenant manager
	backend, err := tenant.NewS3RegistryBackend(loginBucket, loginRegion)
	if err != nil {
		red.Printf("✗ Failed to connect to S3: %v\n", err)
		return err
	}

	manager := tenant.NewManager(backend)
	
	// Load registry
	ctx := context.Background()
	if err := manager.LoadRegistry(ctx); err != nil {
		red.Printf("✗ Failed to load tenant registry: %v\n", err)
		return err
	}
	
	fmt.Println("├── Finding tenant...")
	
	// Verify credentials
	fmt.Println("├── Verifying credentials...")
	t, err := manager.VerifyTenantCredentials(tenantName, secret)
	if err != nil {
		red.Printf("✗ Authentication failed: %v\n", err)
		return err
	}
	
	fmt.Println("├── Loading tenant configuration...")
	
	// Get lock table from registry
	registry := manager.ListTenants()
	lockTable := "panka-locks" // Default
	if len(registry) > 0 {
		// Try to get lock table from config
		// This would come from tenants.yaml metadata
		lockTable = fmt.Sprintf("%s-panka-locks", strings.Split(loginBucket, "-")[0])
	}
	
	// Create session
	sessionMgr := tenant.NewSessionManager()
	if err := sessionMgr.SaveTenantSession(t, loginBucket, loginRegion, lockTable); err != nil {
		red.Printf("✗ Failed to save session: %v\n", err)
		return err
	}
	
	fmt.Println("└── Authentication successful ✓")
	
	green.Println("\n✓ Logged in as: " + t.ID)
	
	// Show session info
	cyan.Println("\n" + strings.Repeat("─", 50))
	fmt.Printf("Tenant:       %s\n", t.DisplayName)
	fmt.Printf("Email:        %s\n", t.Email)
	fmt.Printf("S3 Path:      %s\n", t.Storage.Path)
	fmt.Printf("Version:      %s\n", t.Storage.Version)
	
	fmt.Printf("\nSession saved to: %s/.panka/session\n", os.Getenv("HOME"))
	
	cyan.Println("\nAvailable Commands:")
	fmt.Println("  panka validate          - Validate infrastructure config")
	fmt.Println("  panka plan              - Generate deployment plan")
	fmt.Println("  panka apply             - Deploy infrastructure")
	fmt.Println("  panka destroy           - Destroy infrastructure")
	fmt.Println("  panka state list        - List resources in state")
	fmt.Println("  panka logout            - Logout")
	
	cyan.Println(strings.Repeat("─", 50))
	
	green.Println("\n✨ You are now logged in! Happy deploying!")
	
	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan)

	cyan.Println("\n👋 Logging out...")

	sessionMgr := tenant.NewSessionManager()
	
	// Check if session exists
	session, err := sessionMgr.LoadSession()
	if err != nil {
		return fmt.Errorf("no active session found")
	}
	
	tenantName := "session"
	if session.Mode == tenant.ModeTenant && session.Tenant != nil {
		tenantName = session.Tenant.DisplayName
	} else if session.Mode == tenant.ModeAdmin {
		tenantName = "admin"
	}
	
	// Clear session
	if err := sessionMgr.ClearSession(); err != nil {
		return fmt.Errorf("failed to clear session: %w", err)
	}
	
	green.Printf("✓ Logged out from %s\n", tenantName)
	
	return nil
}

