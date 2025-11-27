package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yourusername/panka/pkg/parser/schema"
)

// Validator provides comprehensive validation for parsed resources
type Validator struct {
	errors []error
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{
		errors: make([]error, 0),
	}
}

// Validate performs comprehensive validation on the parse result
func (v *Validator) Validate(result *ParseResult) error {
	v.errors = make([]error, 0)
	
	// Validate Stack
	if err := v.validateStack(result.Stack); err != nil {
		v.addError(err)
	}
	
	// Validate Services
	for _, service := range result.Services {
		if err := v.validateService(service, result.Stack); err != nil {
			v.addError(err)
		}
	}
	
	// Validate Components
	componentsByService := make(map[string][]schema.Resource)
	for _, comp := range result.Components {
		metadata := comp.GetMetadata()
		if metadata.Service == "" {
			v.addError(fmt.Errorf("component %s has no service reference", metadata.Name))
			continue
		}
		
		componentsByService[metadata.Service] = append(componentsByService[metadata.Service], comp)
		
		if err := v.validateComponent(comp, result); err != nil {
			v.addError(err)
		}
	}
	
	// Validate that all services have components
	for _, service := range result.Services {
		if len(componentsByService[service.Metadata.Name]) == 0 {
			v.addError(fmt.Errorf("service %s has no components", service.Metadata.Name))
		}
	}
	
	// Check for circular dependencies
	if err := v.validateNoCycles(result.Components); err != nil {
		v.addError(err)
	}
	
	if len(v.errors) > 0 {
		return v.formatErrors()
	}
	
	return nil
}

// validateStack validates stack configuration
func (v *Validator) validateStack(stack *schema.Stack) error {
	if stack == nil {
		return fmt.Errorf("stack is nil")
	}
	
	// Validate name follows naming conventions
	if !isValidName(stack.Metadata.Name) {
		return fmt.Errorf("stack name %s is invalid (must be alphanumeric with hyphens)", 
			stack.Metadata.Name)
	}
	
	// Validate provider
	if stack.Spec.Provider.Name == "" {
		return fmt.Errorf("stack provider name is required")
	}
	
	if stack.Spec.Provider.Region == "" {
		return fmt.Errorf("stack provider region is required")
	}
	
	return nil
}

// validateService validates service configuration
func (v *Validator) validateService(service *schema.Service, stack *schema.Stack) error {
	if service == nil {
		return fmt.Errorf("service is nil")
	}
	
	// Validate name
	if !isValidName(service.Metadata.Name) {
		return fmt.Errorf("service name %s is invalid", service.Metadata.Name)
	}
	
	// Validate stack reference
	if service.Metadata.Stack != stack.Metadata.Name {
		return fmt.Errorf("service %s references unknown stack: %s", 
			service.Metadata.Name, service.Metadata.Stack)
	}
	
	return nil
}

// validateComponent validates component configuration
func (v *Validator) validateComponent(comp schema.Resource, result *ParseResult) error {
	metadata := comp.GetMetadata()
	
	// Validate name
	if !isValidName(metadata.Name) {
		return fmt.Errorf("component name %s is invalid", metadata.Name)
	}
	
	// Validate service reference
	serviceExists := false
	for _, service := range result.Services {
		if service.Metadata.Name == metadata.Service {
			serviceExists = true
			break
		}
	}
	
	if !serviceExists {
		return fmt.Errorf("component %s references unknown service: %s", 
			metadata.Name, metadata.Service)
	}
	
	// Type-specific validation
	switch c := comp.(type) {
	case *schema.MicroService:
		return v.validateMicroService(c)
	case *schema.RDS:
		return v.validateRDS(c)
	case *schema.DynamoDB:
		return v.validateDynamoDB(c)
	case *schema.S3:
		return v.validateS3(c)
	}
	
	return nil
}

// validateMicroService validates microservice-specific configuration
func (v *Validator) validateMicroService(ms *schema.MicroService) error {
	// Validate image
	if ms.Spec.Image.Repository == "" {
		return fmt.Errorf("microservice %s: image repository is required", 
			ms.Metadata.Name)
	}
	
	if ms.Spec.Image.Tag == "" {
		return fmt.Errorf("microservice %s: image tag is required", 
			ms.Metadata.Name)
	}
	
	// Validate runtime platform
	validPlatforms := map[string]bool{"fargate": true, "ec2": true, "lambda": true}
	if !validPlatforms[ms.Spec.Runtime.Platform] {
		return fmt.Errorf("microservice %s: invalid platform %s", 
			ms.Metadata.Name, ms.Spec.Runtime.Platform)
	}
	
	// Validate ports
	portNames := make(map[string]bool)
	for _, port := range ms.Spec.Ports {
		if portNames[port.Name] {
			return fmt.Errorf("microservice %s: duplicate port name %s", 
				ms.Metadata.Name, port.Name)
		}
		portNames[port.Name] = true
	}
	
	return nil
}

// validateRDS validates RDS-specific configuration
func (v *Validator) validateRDS(rds *schema.RDS) error {
	// Validate engine
	validEngines := map[string]bool{
		"postgres": true, "mysql": true, "mariadb": true,
		"aurora-postgresql": true, "aurora-mysql": true,
	}
	if !validEngines[rds.Spec.Engine.Type] {
		return fmt.Errorf("RDS %s: invalid engine type %s", 
			rds.Metadata.Name, rds.Spec.Engine.Type)
	}
	
	// Validate storage
	if rds.Spec.Instance.Storage.AllocatedGB < 20 {
		return fmt.Errorf("RDS %s: minimum allocated storage is 20GB", 
			rds.Metadata.Name)
	}
	
	// Validate password secret is provided
	if rds.Spec.Database.PasswordSecret.Ref == "" {
		return fmt.Errorf("RDS %s: password secret reference is required", 
			rds.Metadata.Name)
	}
	
	return nil
}

// validateDynamoDB validates DynamoDB-specific configuration
func (v *Validator) validateDynamoDB(dynamo *schema.DynamoDB) error {
	// Validate billing mode
	if dynamo.Spec.BillingMode != "PAY_PER_REQUEST" && dynamo.Spec.BillingMode != "PROVISIONED" {
		return fmt.Errorf("DynamoDB %s: invalid billing mode %s", 
			dynamo.Metadata.Name, dynamo.Spec.BillingMode)
	}
	
	// Validate provisioned throughput if PROVISIONED
	if dynamo.Spec.BillingMode == "PROVISIONED" {
		if dynamo.Spec.ReadCapacity < 1 {
			return fmt.Errorf("DynamoDB %s: read capacity must be >= 1 for PROVISIONED mode", 
				dynamo.Metadata.Name)
		}
		if dynamo.Spec.WriteCapacity < 1 {
			return fmt.Errorf("DynamoDB %s: write capacity must be >= 1 for PROVISIONED mode", 
				dynamo.Metadata.Name)
		}
	}
	
	// Validate attribute types
	validTypes := map[string]bool{"S": true, "N": true, "B": true}
	if !validTypes[dynamo.Spec.HashKey.Type] {
		return fmt.Errorf("DynamoDB %s: invalid hash key type %s", 
			dynamo.Metadata.Name, dynamo.Spec.HashKey.Type)
	}
	
	return nil
}

// validateS3 validates S3-specific configuration
func (v *Validator) validateS3(s3 *schema.S3) error {
	// Validate ACL
	if s3.Spec.Bucket.ACL != "" {
		validACLs := map[string]bool{
			"private": true, "public-read": true, 
			"public-read-write": true, "authenticated-read": true,
		}
		if !validACLs[s3.Spec.Bucket.ACL] {
			return fmt.Errorf("S3 %s: invalid ACL %s", 
				s3.Metadata.Name, s3.Spec.Bucket.ACL)
		}
	}
	
	// Validate lifecycle rules
	for _, rule := range s3.Spec.Lifecycle {
		if rule.ID == "" {
			return fmt.Errorf("S3 %s: lifecycle rule ID is required", 
				s3.Metadata.Name)
		}
		
		// Validate transitions
		for _, transition := range rule.Transition {
			validClasses := map[string]bool{
				"STANDARD_IA": true, "ONEZONE_IA": true, 
				"INTELLIGENT_TIERING": true, "GLACIER": true, "DEEP_ARCHIVE": true,
			}
			if !validClasses[transition.StorageClass] {
				return fmt.Errorf("S3 %s: invalid storage class %s in lifecycle rule", 
					s3.Metadata.Name, transition.StorageClass)
			}
		}
	}
	
	return nil
}

// validateNoCycles checks for circular dependencies
func (v *Validator) validateNoCycles(components []schema.Resource) error {
	// Build dependency graph
	graph := make(map[string][]string)
	
	for _, comp := range components {
		name := comp.GetMetadata().Name
		graph[name] = extractDependenciesFromResource(comp)
	}
	
	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	
	var hasCycle func(string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		
		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if hasCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}
		
		recStack[node] = false
		return false
	}
	
	for name := range graph {
		if !visited[name] {
			if hasCycle(name) {
				return fmt.Errorf("circular dependency detected involving component: %s", name)
			}
		}
	}
	
	return nil
}

// isValidName checks if a name follows naming conventions
func isValidName(name string) bool {
	// Must be alphanumeric with hyphens, start with letter
	pattern := regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	return pattern.MatchString(name)
}

// extractDependenciesFromResource extracts dependencies from a resource
func extractDependenciesFromResource(resource schema.Resource) []string {
	switch r := resource.(type) {
	case *schema.MicroService:
		return r.Spec.DependsOn
	case *schema.RDS:
		return r.Spec.DependsOn
	case *schema.DynamoDB:
		return r.Spec.DependsOn
	case *schema.S3:
		return r.Spec.DependsOn
	case *schema.SQS:
		return r.Spec.DependsOn
	case *schema.SNS:
		return r.Spec.DependsOn
	default:
		return nil
	}
}

// addError adds a validation error
func (v *Validator) addError(err error) {
	if err != nil {
		v.errors = append(v.errors, err)
	}
}

// formatErrors formats all validation errors into a single error
func (v *Validator) formatErrors() error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("validation failed with %d error(s):\n", len(v.errors)))
	
	for i, err := range v.errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	
	return fmt.Errorf("%s", sb.String())
}

