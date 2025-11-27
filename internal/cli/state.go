package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// stateCmd represents the state command
var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Advanced state management",
	Long: `Advanced state management commands for inspecting and modifying
the Panka state.

State commands allow you to:
  • List resources in the current state
  • Show detailed information about a resource
  • Remove resources from state (without destroying them)
  • Import existing resources into state

⚠️  State manipulation can be dangerous. Use with caution!`,
}

// stateListCmd lists all resources in state
var stateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources in state",
	Long: `List all resources currently tracked in the Panka state.

This shows you what resources Panka knows about and is managing.`,
	RunE: runStateList,
}

// stateShowCmd shows detailed info about a resource
var stateShowCmd = &cobra.Command{
	Use:   "show [resource-id]",
	Short: "Show detailed resource information",
	Long: `Show detailed information about a specific resource in the state.

The resource ID should be in the format: stack.service.resource`,
	Args: cobra.ExactArgs(1),
	RunE: runStateShow,
}

// stateRemoveCmd removes a resource from state
var stateRemoveCmd = &cobra.Command{
	Use:   "rm [resource-id]",
	Short: "Remove resource from state",
	Long: `Remove a resource from the state without destroying it.

⚠️  WARNING: This does not destroy the actual resource!
This only removes it from Panka's tracking. The resource will
continue to exist in your cloud provider.

Use this when:
  • Resource was manually deleted outside Panka
  • You want to stop managing a resource with Panka
  • State is corrupted and needs manual cleanup`,
	Args: cobra.ExactArgs(1),
	RunE: runStateRemove,
}

func init() {
	rootCmd.AddCommand(stateCmd)
	stateCmd.AddCommand(stateListCmd)
	stateCmd.AddCommand(stateShowCmd)
	stateCmd.AddCommand(stateRemoveCmd)
}

func runStateList(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n📋 Listing resources in state...\n")

	// In a real implementation, this would:
	// 1. Load backend configuration
	// 2. Initialize S3 backend
	// 3. Load state from S3
	// 4. Display resources

	// Mock data for demonstration
	resources := []struct {
		ID       string
		Kind     string
		Status   string
		Provider string
		Updated  time.Time
	}{
		{
			ID:       "demo-stack.backend-api.main-db",
			Kind:     "RDS",
			Status:   "available",
			Provider: "aws",
			Updated:  time.Now().Add(-2 * time.Hour),
		},
		{
			ID:       "demo-stack.backend-api.uploads-bucket",
			Kind:     "S3",
			Status:   "available",
			Provider: "aws",
			Updated:  time.Now().Add(-1 * time.Hour),
		},
		{
			ID:       "demo-stack.backend-api.sessions-table",
			Kind:     "DynamoDB",
			Status:   "available",
			Provider: "aws",
			Updated:  time.Now().Add(-30 * time.Minute),
		},
	}

	if len(resources) == 0 {
		yellow.Println("No resources found in state")
		return nil
	}

	// Display in table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tKIND\tSTATUS\tPROVIDER\tLAST UPDATED")
	fmt.Fprintln(w, strings.Repeat("─", 80))

	for _, res := range resources {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			res.ID,
			res.Kind,
			res.Status,
			res.Provider,
			res.Updated.Format("2006-01-02 15:04"),
		)
	}
	w.Flush()

	fmt.Printf("\n")
	green.Printf("Total: %d resources\n", len(resources))

	cyan.Println("\n💡 Tip: Use 'panka state show <resource-id>' for detailed information")

	return nil
}

func runStateShow(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan)

	resourceID := args[0]

	cyan.Printf("\n📄 Resource Details: %s\n\n", resourceID)

	// In a real implementation, load from state backend
	// Mock data for demonstration
	resource := map[string]interface{}{
		"id":          resourceID,
		"kind":        "S3",
		"provider":    "aws",
		"status":      "available",
		"resource_id": "demo-stack-backend-api-uploads-bucket",
		"outputs": map[string]string{
			"bucket_name": "demo-stack-backend-api-uploads-bucket",
			"arn":         "arn:aws:s3:::demo-stack-backend-api-uploads-bucket",
			"region":      "us-east-1",
		},
		"metadata": map[string]string{
			"stack":   "demo-stack",
			"service": "backend-api",
			"name":    "uploads-bucket",
		},
		"created_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"updated_at": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	}

	// Display as formatted JSON
	data, err := json.MarshalIndent(resource, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format resource: %w", err)
	}

	fmt.Println(string(data))

	fmt.Println()
	green.Println("✨ Resource details displayed")

	return nil
}

func runStateRemove(cmd *cobra.Command, args []string) error {
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	cyan := color.New(color.FgCyan)

	resourceID := args[0]

	red.Printf("\n⚠️  Removing resource from state: %s\n\n", resourceID)

	yellow.Println("WARNING: This will remove the resource from state tracking.")
	yellow.Println("The actual cloud resource will NOT be destroyed!")
	fmt.Println()

	cyan.Println("In a full implementation, this would:")
	fmt.Println("  1. Load current state from S3")
	fmt.Println("  2. Acquire DynamoDB lock")
	fmt.Println("  3. Remove resource from state")
	fmt.Println("  4. Save updated state to S3")
	fmt.Println("  5. Release lock")

	yellow.Println("\n⚠️  This operation requires confirmation and state backend")
	cyan.Println("   Use 'panka state rm' with caution in production")

	return nil
}
