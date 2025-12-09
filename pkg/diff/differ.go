package diff

import (
	"reflect"
	"strings"

	"github.com/yourusername/panka/pkg/parser"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/state"
)

// Differ compares desired state with actual state to determine changes
type Differ struct {
	// Options for diff behavior
	options *DifferOptions
}

// DifferOptions configures the diff behavior
type DifferOptions struct {
	// IgnoreTags ignores tag changes
	IgnoreTags bool

	// IgnoreMetadataFields lists metadata fields to ignore in comparison
	IgnoreMetadataFields []string

	// RecreateOnTypeChange forces recreation when type changes
	RecreateOnTypeChange bool

	// DeepCompare enables deep comparison of nested objects
	DeepCompare bool
}

// DefaultDifferOptions returns default diff options
func DefaultDifferOptions() *DifferOptions {
	return &DifferOptions{
		IgnoreTags:           false,
		IgnoreMetadataFields: []string{"created_at", "updated_at"},
		RecreateOnTypeChange: true,
		DeepCompare:          true,
	}
}

// NewDiffer creates a new Differ instance
func NewDiffer(opts *DifferOptions) *Differ {
	if opts == nil {
		opts = DefaultDifferOptions()
	}
	return &Differ{options: opts}
}

// ComputeChanges compares desired infrastructure with current state and returns changes
func (d *Differ) ComputeChanges(
	desired *parser.ParseResult,
	currentState *state.State,
	stackName, environment string,
) (*ChangeSet, error) {
	changeSet := NewChangeSet(stackName, environment)

	if currentState == nil {
		currentState = state.NewState(stackName, environment)
	}

	// Build a map of desired resources by name
	desiredByName := make(map[string]schema.Resource)
	for _, component := range desired.Components {
		metadata := component.GetMetadata()
		desiredByName[metadata.Name] = component
	}

	// Build a set of resource names we've processed
	processed := make(map[string]bool)

	// Compare desired against current state
	for name, desiredResource := range desiredByName {
		processed[name] = true

		// Check if resource exists in current state
		currentResource, exists := currentState.GetResource(name)

		if !exists {
			// Resource needs to be created
			changeSet.AddChange(&Change{
				ResourceID:   "",
				ResourceName: name,
				ResourceKind: desiredResource.GetKind(),
				Type:         ChangeCreate,
				Service:      desiredResource.GetMetadata().Service,
				Before:       nil,
				After:        desiredResource,
				Reason:       "Resource does not exist",
			})
		} else {
			// Resource exists, compare for changes
			change := d.compareResource(desiredResource, currentResource)
			changeSet.AddChange(change)
		}
	}

	// Find resources in state that aren't in desired (should be deleted)
	for name, currentResource := range currentState.Resources {
		if !processed[name] {
			changeSet.AddChange(&Change{
				ResourceID:   currentResource.ID,
				ResourceName: name,
				ResourceKind: schema.Kind(currentResource.Type),
				Type:         ChangeDelete,
				Before:       currentResource,
				After:        nil,
				Reason:       "Resource no longer in configuration",
			})
		}
	}

	return changeSet, nil
}

// compareResource compares a desired resource with its current state
func (d *Differ) compareResource(desired schema.Resource, current *state.Resource) *Change {
	metadata := desired.GetMetadata()
	change := &Change{
		ResourceID:   current.ID,
		ResourceName: metadata.Name,
		ResourceKind: desired.GetKind(),
		Service:      metadata.Service,
		Before:       current,
		After:        desired,
	}

	// Check if resource type changed (requires recreate)
	if string(desired.GetKind()) != current.Type {
		change.Type = ChangeRecreate
		change.RequiresRecreate = true
		change.Reason = "Resource type changed"
		change.AttributeChanges = append(change.AttributeChanges, AttributeChange{
			Path:          "kind",
			OldValue:      current.Type,
			NewValue:      string(desired.GetKind()),
			ForceRecreate: true,
		})
		return change
	}

	// Compare attributes based on resource type
	attrChanges := d.compareAttributes(desired, current)

	if len(attrChanges) == 0 {
		change.Type = ChangeNoChange
		change.Reason = "No changes detected"
	} else {
		// Check if any changes require recreation
		requiresRecreate := false
		for _, ac := range attrChanges {
			if ac.ForceRecreate {
				requiresRecreate = true
				break
			}
		}

		if requiresRecreate {
			change.Type = ChangeRecreate
			change.RequiresRecreate = true
			change.Reason = "One or more changes require recreation"
		} else {
			change.Type = ChangeUpdate
			change.Reason = "Configuration changes detected"
		}
		change.AttributeChanges = attrChanges
	}

	return change
}

// compareAttributes compares the attributes of a desired resource with current state
func (d *Differ) compareAttributes(desired schema.Resource, current *state.Resource) []AttributeChange {
	var changes []AttributeChange

	// Get spec-level changes based on resource type
	specChanges := d.compareSpec(desired, current.Attributes)
	changes = append(changes, specChanges...)

	return changes
}

// compareSpec compares the specification of a resource with stored attributes
func (d *Differ) compareSpec(desired schema.Resource, currentAttrs map[string]interface{}) []AttributeChange {
	var changes []AttributeChange

	// Compare based on resource type
	switch res := desired.(type) {
	case *schema.S3:
		changes = append(changes, d.compareS3(res, currentAttrs)...)
	case *schema.DynamoDB:
		changes = append(changes, d.compareDynamoDB(res, currentAttrs)...)
	case *schema.SQS:
		changes = append(changes, d.compareSQS(res, currentAttrs)...)
	case *schema.SNS:
		changes = append(changes, d.compareSNS(res, currentAttrs)...)
	case *schema.RDS:
		changes = append(changes, d.compareRDS(res, currentAttrs)...)
	}

	return changes
}

// compareS3 compares S3 bucket configuration
func (d *Differ) compareS3(desired *schema.S3, current map[string]interface{}) []AttributeChange {
	var changes []AttributeChange

	// Versioning
	if desired.Spec.Versioning != nil {
		if current["versioning"] != nil {
			currentVersioning, _ := current["versioning"].(bool)
			if currentVersioning != desired.Spec.Versioning.Enabled {
				changes = append(changes, AttributeChange{
					Path:     "spec.versioning.enabled",
					OldValue: currentVersioning,
					NewValue: desired.Spec.Versioning.Enabled,
				})
			}
		}
	}

	// ACL (might require recreate)
	if current["acl"] != nil {
		currentACL, _ := current["acl"].(string)
		if currentACL != desired.Spec.Bucket.ACL && desired.Spec.Bucket.ACL != "" {
			changes = append(changes, AttributeChange{
				Path:     "spec.bucket.acl",
				OldValue: currentACL,
				NewValue: desired.Spec.Bucket.ACL,
			})
		}
	}

	// Encryption
	if desired.Spec.Encryption != nil {
		if current["encryption_enabled"] != nil {
			currentEncryption, _ := current["encryption_enabled"].(bool)
			if currentEncryption != desired.Spec.Encryption.Enabled {
				changes = append(changes, AttributeChange{
					Path:     "spec.encryption.enabled",
					OldValue: currentEncryption,
					NewValue: desired.Spec.Encryption.Enabled,
				})
			}
		}
	}

	return changes
}

// compareDynamoDB compares DynamoDB table configuration
func (d *Differ) compareDynamoDB(desired *schema.DynamoDB, current map[string]interface{}) []AttributeChange {
	var changes []AttributeChange

	// Hash key change requires recreation
	if current["hash_key"] != nil {
		currentPK, _ := current["hash_key"].(string)
		if currentPK != desired.Spec.HashKey.Name {
			changes = append(changes, AttributeChange{
				Path:          "spec.hashKey.name",
				OldValue:      currentPK,
				NewValue:      desired.Spec.HashKey.Name,
				ForceRecreate: true,
			})
		}
	}

	// Range key change requires recreation
	if desired.Spec.RangeKey != nil && current["range_key"] != nil {
		currentSK, _ := current["range_key"].(string)
		if currentSK != desired.Spec.RangeKey.Name {
			changes = append(changes, AttributeChange{
				Path:          "spec.rangeKey.name",
				OldValue:      currentSK,
				NewValue:      desired.Spec.RangeKey.Name,
				ForceRecreate: true,
			})
		}
	}

	// Billing mode can be updated
	if current["billing_mode"] != nil {
		currentBilling, _ := current["billing_mode"].(string)
		if currentBilling != desired.Spec.BillingMode {
			changes = append(changes, AttributeChange{
				Path:     "spec.billingMode",
				OldValue: currentBilling,
				NewValue: desired.Spec.BillingMode,
			})
		}
	}

	// Capacity changes (if provisioned)
	if desired.Spec.BillingMode == "PROVISIONED" {
		if current["read_capacity"] != nil {
			currentRead, _ := current["read_capacity"].(float64)
			if int(currentRead) != desired.Spec.ReadCapacity {
				changes = append(changes, AttributeChange{
					Path:     "spec.readCapacity",
					OldValue: int(currentRead),
					NewValue: desired.Spec.ReadCapacity,
				})
			}
		}
		if current["write_capacity"] != nil {
			currentWrite, _ := current["write_capacity"].(float64)
			if int(currentWrite) != desired.Spec.WriteCapacity {
				changes = append(changes, AttributeChange{
					Path:     "spec.writeCapacity",
					OldValue: int(currentWrite),
					NewValue: desired.Spec.WriteCapacity,
				})
			}
		}
	}

	return changes
}

// compareSQS compares SQS queue configuration
func (d *Differ) compareSQS(desired *schema.SQS, current map[string]interface{}) []AttributeChange {
	var changes []AttributeChange

	// Queue type change requires recreation (Standard vs FIFO)
	if current["type"] != nil {
		currentType, _ := current["type"].(string)
		isFifo := desired.Spec.Type == "fifo"
		currentIsFifo := currentType == "fifo"
		if currentIsFifo != isFifo {
			changes = append(changes, AttributeChange{
				Path:          "spec.type",
				OldValue:      currentType,
				NewValue:      desired.Spec.Type,
				ForceRecreate: true,
			})
		}
	}

	// Visibility timeout can be updated
	if current["visibility_timeout"] != nil {
		currentVT, _ := current["visibility_timeout"].(float64)
		if int(currentVT) != desired.Spec.VisibilityTimeout && desired.Spec.VisibilityTimeout > 0 {
			changes = append(changes, AttributeChange{
				Path:     "spec.visibilityTimeout",
				OldValue: int(currentVT),
				NewValue: desired.Spec.VisibilityTimeout,
			})
		}
	}

	// Message retention can be updated
	if current["message_retention"] != nil {
		currentMR, _ := current["message_retention"].(float64)
		if int(currentMR) != desired.Spec.MessageRetentionPeriod && desired.Spec.MessageRetentionPeriod > 0 {
			changes = append(changes, AttributeChange{
				Path:     "spec.messageRetentionPeriod",
				OldValue: int(currentMR),
				NewValue: desired.Spec.MessageRetentionPeriod,
			})
		}
	}

	return changes
}

// compareSNS compares SNS topic configuration
func (d *Differ) compareSNS(desired *schema.SNS, current map[string]interface{}) []AttributeChange {
	var changes []AttributeChange

	// FIFO change requires recreation
	if current["fifo"] != nil {
		currentFIFO, _ := current["fifo"].(bool)
		if currentFIFO != desired.Spec.FifoTopic {
			changes = append(changes, AttributeChange{
				Path:          "spec.fifoTopic",
				OldValue:      currentFIFO,
				NewValue:      desired.Spec.FifoTopic,
				ForceRecreate: true,
			})
		}
	}

	// Display name can be updated
	if current["display_name"] != nil {
		currentDN, _ := current["display_name"].(string)
		if currentDN != desired.Spec.DisplayName && desired.Spec.DisplayName != "" {
			changes = append(changes, AttributeChange{
				Path:     "spec.displayName",
				OldValue: currentDN,
				NewValue: desired.Spec.DisplayName,
			})
		}
	}

	return changes
}

// compareRDS compares RDS instance configuration
func (d *Differ) compareRDS(desired *schema.RDS, current map[string]interface{}) []AttributeChange {
	var changes []AttributeChange

	// Engine change requires recreation
	if current["engine"] != nil {
		currentEngine, _ := current["engine"].(string)
		if currentEngine != desired.Spec.Engine.Type {
			changes = append(changes, AttributeChange{
				Path:          "spec.engine.type",
				OldValue:      currentEngine,
				NewValue:      desired.Spec.Engine.Type,
				ForceRecreate: true,
			})
		}
	}

	// Instance class can be updated (causes restart)
	if current["instance_class"] != nil {
		currentClass, _ := current["instance_class"].(string)
		if currentClass != desired.Spec.Instance.Class {
			changes = append(changes, AttributeChange{
				Path:     "spec.instance.class",
				OldValue: currentClass,
				NewValue: desired.Spec.Instance.Class,
			})
		}
	}

	// Storage size can be increased (not decreased)
	if current["storage_size"] != nil {
		currentStorage, _ := current["storage_size"].(float64)
		if int(currentStorage) != desired.Spec.Instance.Storage.AllocatedGB {
			change := AttributeChange{
				Path:     "spec.instance.storage.allocatedGB",
				OldValue: int(currentStorage),
				NewValue: desired.Spec.Instance.Storage.AllocatedGB,
			}
			// Decreasing storage requires recreation
			if desired.Spec.Instance.Storage.AllocatedGB < int(currentStorage) {
				change.ForceRecreate = true
			}
			changes = append(changes, change)
		}
	}

	// Multi-AZ can be updated
	if current["multi_az"] != nil {
		currentMultiAZ, _ := current["multi_az"].(bool)
		if currentMultiAZ != desired.Spec.Instance.MultiAZ {
			changes = append(changes, AttributeChange{
				Path:     "spec.instance.multiAZ",
				OldValue: currentMultiAZ,
				NewValue: desired.Spec.Instance.MultiAZ,
			})
		}
	}

	return changes
}

// compareValues compares two values and returns true if they differ
func (d *Differ) compareValues(a, b interface{}) bool {
	if d.options.DeepCompare {
		return !reflect.DeepEqual(a, b)
	}
	return a != b
}

// isIgnoredField checks if a field should be ignored in comparison
func (d *Differ) isIgnoredField(field string) bool {
	for _, ignored := range d.options.IgnoreMetadataFields {
		if strings.EqualFold(field, ignored) {
			return true
		}
	}
	return false
}

// ComputeChangesFromFolderParse computes changes from a folder parse result
func (d *Differ) ComputeChangesFromFolderParse(
	desired *parser.StackParseResult,
	currentState *state.State,
) (*ChangeSet, error) {
	// Convert StackParseResult to ParseResult for comparison
	parseResult := &parser.ParseResult{
		Stack:      desired.Stack,
		Components: desired.AllComponents,
	}
	for _, svc := range desired.Services {
		if svc.Service != nil {
			parseResult.Services = append(parseResult.Services, svc.Service)
		}
	}

	stackName := ""
	environment := "default"
	if desired.Stack != nil {
		stackName = desired.Stack.Metadata.Name
	}

	return d.ComputeChanges(parseResult, currentState, stackName, environment)
}

