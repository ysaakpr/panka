package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/panka/internal/logger"
	"go.uber.org/zap"
)

var (
	cfgFile    string
	logLevel   string
	logFormat  string
	tenantMode bool
	tenantID   string
	verbose    bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "panka",
	Short: "Panka - Multi-tenant Infrastructure Deployment Tool",
	Long: `Panka is a multi-tenant infrastructure deployment tool that helps teams
manage their cloud resources declaratively using YAML configuration.

Features:
  • Multi-tenant isolation with state separation
  • Declarative YAML configuration
  • Dependency-aware deployment ordering
  • State management with S3 backend
  • Distributed locking with DynamoDB
  • AWS resource management (S3, DynamoDB, SQS, SNS, etc.)`,
	Version: "0.1.0-dev",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip logger setup for completion commands
		if cmd.Name() == "completion" || cmd.Name() == "__complete" {
			return
		}
		
		// Initialize logger
		initLogger()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.panka.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "console", "log format (console, json)")
	rootCmd.PersistentFlags().BoolVar(&tenantMode, "tenant-mode", false, "enable tenant mode")
	rootCmd.PersistentFlags().StringVar(&tenantID, "tenant-id", "", "tenant ID (required in tenant mode)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Bind flags to viper
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log.format", rootCmd.PersistentFlags().Lookup("log-format"))
	viper.BindPFlag("tenant.mode", rootCmd.PersistentFlags().Lookup("tenant-mode"))
	viper.BindPFlag("tenant.id", rootCmd.PersistentFlags().Lookup("tenant-id"))
}

// initConfig reads in config file and ENV variables
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
			os.Exit(1)
		}

		// Search for config in home directory
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".panka")
	}

	// Read in environment variables
	viper.SetEnvPrefix("PANKA")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

// initLogger initializes the global logger
func initLogger() {
	var log *logger.Logger
	var err error

	format := viper.GetString("log.format")

	if format == "json" {
		log, err = logger.NewProduction()
	} else {
		log, err = logger.NewDevelopment()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.SetGlobal(log)
	
	level := viper.GetString("log.level")
	logger.Global().WithFields(
		zap.String("level", level),
		zap.String("format", format),
	).Debug("Logger initialized")
}

// GetTenantConfig returns tenant configuration from flags/config
func GetTenantConfig() (bool, string) {
	tenantMode := viper.GetBool("tenant.mode")
	tenantID := viper.GetString("tenant.id")
	
	return tenantMode, tenantID
}

// GetVerbose returns verbose flag
func GetVerbose() bool {
	return verbose
}

