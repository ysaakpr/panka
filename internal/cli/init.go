package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	initForce bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Panka configuration",
	Long: `Initialize Panka configuration in the current directory.

This creates a .panka.yaml configuration file with default settings
for backend configuration, logging, and tenant mode.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite existing configuration")
}

func runInit(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow)
	cyan := color.New(color.FgCyan)

	cyan.Println("\n🚀 Initializing Panka...")

	// Check if config already exists
	configPath := filepath.Join(".", ".panka.yaml")
	if _, err := os.Stat(configPath); err == nil && !initForce {
		return fmt.Errorf("configuration file already exists at %s (use --force to overwrite)", configPath)
	}

	// Create default configuration
	config := `# Panka Configuration
# Documentation: https://github.com/yourusername/panka

# Backend configuration for state storage
backend:
  type: s3
  bucket: panka-state-${ACCOUNT_ID}  # Replace with your S3 bucket
  region: us-east-1
  prefix: states/
  dynamodb_table: panka-locks  # For distributed locking

# Logging configuration
log:
  level: info      # debug, info, warn, error
  format: console  # console, json

# Tenant mode (multi-tenant isolation)
tenant:
  mode: false    # Set to true for multi-tenant mode
  # id: ""       # Tenant ID (required when mode=true)

# AWS provider configuration
providers:
  aws:
    region: us-east-1
    # profile: default  # AWS profile to use
    # role_arn: ""      # IAM role to assume

# Default tags applied to all resources
default_tags:
  managed_by: panka
  environment: development
  # team: platform
  # cost_center: engineering
`

	// Write configuration file
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write configuration: %w", err)
	}

	green.Printf("✅ Created configuration file: %s\n", configPath)

	// Create example infrastructure file
	examplePath := filepath.Join(".", "infrastructure.yaml")
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		example := `# Panka Infrastructure Configuration
# This is an example configuration showing various resource types

---
# Stack definition
apiVersion: panka.dev/v1
kind: Stack
metadata:
  name: my-stack
  labels:
    environment: development
spec:
  description: Example infrastructure stack
  region: us-east-1

---
# S3 Bucket for uploads
apiVersion: panka.dev/v1
kind: S3
metadata:
  name: uploads
  stack: my-stack
  service: backend
spec:
  bucket:
    acl: private
  versioning:
    enabled: true
  encryption:
    enabled: true
    algorithm: AES256

---
# DynamoDB Table for sessions
apiVersion: panka.dev/v1
kind: DynamoDB
metadata:
  name: sessions
  stack: my-stack
  service: backend
spec:
  billing_mode: PAY_PER_REQUEST
  hash_key:
    name: userId
    type: S
  range_key:
    name: sessionId
    type: S
  ttl:
    enabled: true
    attribute_name: expiresAt

---
# SQS Queue for processing
apiVersion: panka.dev/v1
kind: SQS
metadata:
  name: processing
  stack: my-stack
  service: backend
spec:
  type: standard
  visibility_timeout: 300
  message_retention_period: 345600
  receive_wait_time: 20

---
# SNS Topic for notifications
apiVersion: panka.dev/v1
kind: SNS
metadata:
  name: notifications
  stack: my-stack
  service: backend
spec:
  display_name: Notifications Topic
  subscriptions:
    - protocol: email
      endpoint: admin@example.com
`

		if err := os.WriteFile(examplePath, []byte(example), 0644); err != nil {
			yellow.Printf("⚠️  Could not create example file: %v\n", err)
		} else {
			green.Printf("✅ Created example file: %s\n", examplePath)
		}
	}

	// Print next steps
	cyan.Println("\n📋 Next steps:")
	fmt.Println("  1. Edit .panka.yaml and configure your backend (S3 bucket, DynamoDB table)")
	fmt.Println("  2. Edit infrastructure.yaml to define your resources")
	fmt.Println("  3. Run 'panka validate infrastructure.yaml' to validate your configuration")
	fmt.Println("  4. Run 'panka plan infrastructure.yaml' to see what will be created")
	fmt.Println("  5. Run 'panka apply infrastructure.yaml' to deploy your infrastructure")

	cyan.Println("\n📚 Documentation:")
	fmt.Println("  • Getting Started: https://github.com/yourusername/panka/blob/main/docs/GETTING_STARTED_GUIDE.md")
	fmt.Println("  • Examples:        https://github.com/yourusername/panka/tree/main/examples")

	green.Println("\n✨ Initialization complete!")

	return nil
}

