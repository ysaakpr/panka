package diff

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// Formatter formats change sets for display
type Formatter struct {
	// UseColors enables ANSI color output
	UseColors bool

	// ShowDetails shows attribute-level changes
	ShowDetails bool

	// MaxWidth is the maximum line width
	MaxWidth int
}

// NewFormatter creates a new formatter with default settings
func NewFormatter() *Formatter {
	return &Formatter{
		UseColors:   true,
		ShowDetails: true,
		MaxWidth:    80,
	}
}

// Format formats a change set for display
func (f *Formatter) Format(cs *ChangeSet) string {
	var sb strings.Builder

	// Header
	sb.WriteString(f.formatHeader(cs))

	// Group changes by type
	creates := cs.GetChangesByType(ChangeCreate)
	updates := cs.GetChangesByType(ChangeUpdate)
	deletes := cs.GetChangesByType(ChangeDelete)
	recreates := cs.GetChangesByType(ChangeRecreate)
	noChanges := cs.GetChangesByType(ChangeNoChange)

	// Display in order: creates, updates, recreates, deletes
	if len(creates) > 0 {
		sb.WriteString(f.formatSection("Resources to Create", creates, ChangeCreate))
	}
	if len(updates) > 0 {
		sb.WriteString(f.formatSection("Resources to Update", updates, ChangeUpdate))
	}
	if len(recreates) > 0 {
		sb.WriteString(f.formatSection("Resources to Recreate", recreates, ChangeRecreate))
	}
	if len(deletes) > 0 {
		sb.WriteString(f.formatSection("Resources to Delete", deletes, ChangeDelete))
	}

	// Summary
	sb.WriteString(f.formatSummary(cs.Summary, len(noChanges)))

	return sb.String()
}

// FormatCompact returns a compact single-line summary
func (f *Formatter) FormatCompact(cs *ChangeSet) string {
	return fmt.Sprintf("Plan: %d to add, %d to change, %d to destroy",
		cs.Summary.Create, cs.Summary.Update+cs.Summary.Recreate, cs.Summary.Delete)
}

// formatHeader formats the change set header
func (f *Formatter) formatHeader(cs *ChangeSet) string {
	var sb strings.Builder

	cyan := color.New(color.FgCyan, color.Bold)

	sb.WriteString("\n")
	sb.WriteString(cyan.Sprint("📋 Infrastructure Changes"))
	sb.WriteString("\n")
	sb.WriteString(cyan.Sprint(strings.Repeat("─", 60)))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Stack:       %s\n", cs.StackName))
	sb.WriteString(fmt.Sprintf("Environment: %s\n", cs.Environment))
	if cs.TenantID != "" {
		sb.WriteString(fmt.Sprintf("Tenant:      %s\n", cs.TenantID))
	}
	sb.WriteString("\n")

	return sb.String()
}

// formatSection formats a section of changes
func (f *Formatter) formatSection(title string, changes []*Change, changeType ChangeType) string {
	var sb strings.Builder

	// Section header with appropriate color
	var titleColor *color.Color
	switch changeType {
	case ChangeCreate:
		titleColor = color.New(color.FgGreen, color.Bold)
	case ChangeUpdate:
		titleColor = color.New(color.FgYellow, color.Bold)
	case ChangeDelete:
		titleColor = color.New(color.FgRed, color.Bold)
	case ChangeRecreate:
		titleColor = color.New(color.FgMagenta, color.Bold)
	default:
		titleColor = color.New(color.FgWhite)
	}

	sb.WriteString(titleColor.Sprintf("\n%s (%d)\n", title, len(changes)))

	for _, change := range changes {
		sb.WriteString(f.formatChange(change))
	}

	return sb.String()
}

// formatChange formats a single change
func (f *Formatter) formatChange(change *Change) string {
	var sb strings.Builder

	// Symbol and color based on change type
	var changeColor *color.Color
	symbol := change.Type.Symbol()

	switch change.Type {
	case ChangeCreate:
		changeColor = color.New(color.FgGreen)
	case ChangeUpdate:
		changeColor = color.New(color.FgYellow)
	case ChangeDelete:
		changeColor = color.New(color.FgRed)
	case ChangeRecreate:
		changeColor = color.New(color.FgMagenta)
	default:
		changeColor = color.New(color.FgWhite)
	}

	// Main line
	sb.WriteString(changeColor.Sprintf("  %s ", symbol))
	sb.WriteString(fmt.Sprintf("[%s] %s\n", change.ResourceKind, change.ResourceName))

	// Service info
	if change.Service != "" {
		sb.WriteString(fmt.Sprintf("      Service: %s\n", change.Service))
	}

	// Resource ID (for updates/deletes)
	if change.ResourceID != "" && (change.Type == ChangeUpdate || change.Type == ChangeDelete || change.Type == ChangeRecreate) {
		sb.WriteString(fmt.Sprintf("      ID: %s\n", change.ResourceID))
	}

	// Show attribute changes for updates
	if f.ShowDetails && len(change.AttributeChanges) > 0 {
		sb.WriteString("      Changes:\n")
		for _, attr := range change.AttributeChanges {
			sb.WriteString(f.formatAttributeChange(attr))
		}
	}

	// Show reason
	if change.Reason != "" && change.Type != ChangeNoChange {
		dimColor := color.New(color.Faint)
		sb.WriteString(dimColor.Sprintf("      Reason: %s\n", change.Reason))
	}

	return sb.String()
}

// formatAttributeChange formats a single attribute change
func (f *Formatter) formatAttributeChange(attr AttributeChange) string {
	var sb strings.Builder

	if attr.ForceRecreate {
		recreateColor := color.New(color.FgMagenta)
		sb.WriteString(recreateColor.Sprintf("        %s (forces recreate):\n", attr.Path))
	} else {
		sb.WriteString(fmt.Sprintf("        %s:\n", attr.Path))
	}

	if attr.Sensitive {
		sb.WriteString("          (sensitive value)\n")
	} else {
		redColor := color.New(color.FgRed)
		greenColor := color.New(color.FgGreen)

		if attr.OldValue != nil {
			sb.WriteString(redColor.Sprintf("          - %v\n", attr.OldValue))
		}
		if attr.NewValue != nil {
			sb.WriteString(greenColor.Sprintf("          + %v\n", attr.NewValue))
		}
	}

	return sb.String()
}

// formatSummary formats the change summary
func (f *Formatter) formatSummary(summary ChangeSummary, noChangeCount int) string {
	var sb strings.Builder

	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	magenta := color.New(color.FgMagenta, color.Bold)
	dim := color.New(color.Faint)

	sb.WriteString("\n")
	sb.WriteString(cyan.Sprint(strings.Repeat("─", 60)))
	sb.WriteString("\n")
	sb.WriteString(cyan.Sprint("📊 Summary"))
	sb.WriteString("\n\n")

	// Create line
	if summary.Create > 0 {
		sb.WriteString(green.Sprintf("  + Create:   %d resource(s)\n", summary.Create))
	}

	// Update line
	if summary.Update > 0 {
		sb.WriteString(yellow.Sprintf("  ~ Update:   %d resource(s)\n", summary.Update))
	}

	// Recreate line
	if summary.Recreate > 0 {
		sb.WriteString(magenta.Sprintf("  ± Recreate: %d resource(s)\n", summary.Recreate))
	}

	// Delete line
	if summary.Delete > 0 {
		sb.WriteString(red.Sprintf("  - Delete:   %d resource(s)\n", summary.Delete))
	}

	// No change line (dimmed)
	if noChangeCount > 0 {
		sb.WriteString(dim.Sprintf("    Unchanged: %d resource(s)\n", noChangeCount))
	}

	// Total changes
	totalChanges := summary.Create + summary.Update + summary.Delete + summary.Recreate
	sb.WriteString("\n")
	if totalChanges == 0 {
		sb.WriteString(green.Sprint("  ✓ No changes detected. Infrastructure is up-to-date.\n"))
	} else {
		sb.WriteString(fmt.Sprintf("  Total: %d change(s) to apply\n", totalChanges))
	}

	return sb.String()
}

// FormatJSON formats the change set as JSON (useful for CI/CD)
func (f *Formatter) FormatJSON(cs *ChangeSet) (string, error) {
	// For actual JSON formatting, you'd use encoding/json
	// This is a placeholder for the concept
	return fmt.Sprintf(`{"stack":"%s","creates":%d,"updates":%d,"deletes":%d}`,
		cs.StackName, cs.Summary.Create, cs.Summary.Update, cs.Summary.Delete), nil
}

// PrintDiff is a convenience function that creates a formatter and prints the diff
func PrintDiff(cs *ChangeSet) {
	formatter := NewFormatter()
	fmt.Print(formatter.Format(cs))
}

