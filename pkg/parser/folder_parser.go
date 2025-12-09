package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/tenant"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// FolderParser parses stack folders with the structure:
//
//	my-stack/
//	├── stack.yaml
//	└── services/
//	    ├── api/
//	    │   ├── service.yaml
//	    │   └── *.yaml
//	    └── worker/
//	        ├── service.yaml
//	        └── *.yaml
type FolderParser struct {
	parser *Parser
	logger *logger.Logger

	// Tenant configuration (for networking inheritance)
	tenantConfig *tenant.Tenant
}

// NewFolderParser creates a new folder parser
func NewFolderParser() *FolderParser {
	return &FolderParser{
		parser: NewParser(),
		logger: logger.Global(),
	}
}

// NewFolderParserWithTenant creates a folder parser with tenant config for networking
func NewFolderParserWithTenant(tenantCfg *tenant.Tenant) *FolderParser {
	fp := NewFolderParser()
	fp.tenantConfig = tenantCfg
	return fp
}

// StackParseResult contains the complete parsed stack
type StackParseResult struct {
	// Stack definition from stack.yaml
	Stack *schema.Stack

	// Services parsed from services/ subfolders
	Services map[string]*ServiceParseResult

	// All components flattened (for dependency resolution)
	AllComponents []schema.Resource

	// Tenant networking (if available)
	TenantNetworking *tenant.NetworkingConfig

	// Stack folder path
	StackPath string

	// Validation errors (non-fatal)
	Warnings []string
}

// ServiceParseResult contains a parsed service
type ServiceParseResult struct {
	// Service definition from service.yaml
	Service *schema.Service

	// Components in this service
	Components []schema.Resource

	// Config files (non-YAML)
	ConfigFiles map[string][]byte

	// Service folder path
	ServicePath string
}

// ParseStackFolder parses an entire stack folder
func (fp *FolderParser) ParseStackFolder(stackPath string) (*StackParseResult, error) {
	fp.logger.Info("Parsing stack folder", zap.String("path", stackPath))

	// Normalize path
	stackPath, err := filepath.Abs(stackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Validate folder structure
	if err := fp.validateFolderStructure(stackPath); err != nil {
		return nil, err
	}

	result := &StackParseResult{
		Services:      make(map[string]*ServiceParseResult),
		AllComponents: make([]schema.Resource, 0),
		StackPath:     stackPath,
		Warnings:      make([]string, 0),
	}

	// 1. Parse stack.yaml
	stackFile := filepath.Join(stackPath, "stack.yaml")
	stack, err := fp.parseStackFile(stackFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse stack.yaml: %w", err)
	}
	result.Stack = stack

	// Store stack variables for interpolation
	if stack.Spec.Variables != nil {
		for k, v := range stack.Spec.Variables {
			fp.parser.SetVariable(k, v)
		}
	}

	// 2. Parse services folder
	servicesPath := filepath.Join(stackPath, "services")
	if _, err := os.Stat(servicesPath); err == nil {
		services, err := fp.parseServicesFolder(servicesPath, stack)
		if err != nil {
			return nil, fmt.Errorf("failed to parse services: %w", err)
		}
		result.Services = services

		// Flatten all components
		for _, svc := range services {
			result.AllComponents = append(result.AllComponents, svc.Components...)
		}
	}

	// 3. Add tenant networking if available
	if fp.tenantConfig != nil {
		result.TenantNetworking = &fp.tenantConfig.Networking
	}

	// 4. Validate cross-references
	if err := fp.validateCrossReferences(result); err != nil {
		result.Warnings = append(result.Warnings, err.Error())
	}

	fp.logger.Info("Stack parsing complete",
		zap.String("stack", stack.Metadata.Name),
		zap.Int("services", len(result.Services)),
		zap.Int("components", len(result.AllComponents)),
	)

	return result, nil
}

// validateFolderStructure checks that the folder has the expected structure
func (fp *FolderParser) validateFolderStructure(stackPath string) error {
	// Check if directory exists
	info, err := os.Stat(stackPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("stack folder does not exist: %s", stackPath)
		}
		return fmt.Errorf("failed to access stack folder: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", stackPath)
	}

	// Check for stack.yaml
	stackFile := filepath.Join(stackPath, "stack.yaml")
	if _, err := os.Stat(stackFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("stack.yaml not found in %s", stackPath)
		}
		return fmt.Errorf("failed to access stack.yaml: %w", err)
	}

	return nil
}

// parseStackFile parses the stack.yaml file
func (fp *FolderParser) parseStackFile(path string) (*schema.Stack, error) {
	fp.logger.Debug("Parsing stack file", zap.String("path", path))

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read stack.yaml: %w", err)
	}

	// Parse as generic resource first to check kind
	var base schema.ResourceBase
	if err := yaml.Unmarshal(content, &base); err != nil {
		return nil, fmt.Errorf("failed to parse stack.yaml: %w", err)
	}

	if base.Kind != schema.KindStack {
		return nil, fmt.Errorf("stack.yaml must contain kind: Stack, got: %s", base.Kind)
	}

	// Parse as Stack
	var stack schema.Stack
	if err := yaml.Unmarshal(content, &stack); err != nil {
		return nil, fmt.Errorf("failed to parse Stack: %w", err)
	}

	return &stack, nil
}

// parseServicesFolder parses all service subfolders
func (fp *FolderParser) parseServicesFolder(servicesPath string, stack *schema.Stack) (map[string]*ServiceParseResult, error) {
	fp.logger.Debug("Parsing services folder", zap.String("path", servicesPath))

	services := make(map[string]*ServiceParseResult)

	// List service directories
	entries, err := os.ReadDir(servicesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read services folder: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		serviceName := entry.Name()
		servicePath := filepath.Join(servicesPath, serviceName)

		fp.logger.Debug("Parsing service", zap.String("service", serviceName))

		service, err := fp.parseServiceFolder(servicePath, stack, serviceName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse service %s: %w", serviceName, err)
		}

		services[serviceName] = service
	}

	return services, nil
}

// parseServiceFolder parses a single service folder
func (fp *FolderParser) parseServiceFolder(servicePath string, stack *schema.Stack, serviceName string) (*ServiceParseResult, error) {
	result := &ServiceParseResult{
		Components:  make([]schema.Resource, 0),
		ConfigFiles: make(map[string][]byte),
		ServicePath: servicePath,
	}

	// Find all YAML files in the service folder
	yamlFiles, err := filepath.Glob(filepath.Join(servicePath, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list YAML files: %w", err)
	}

	// Also check for .yml files
	ymlFiles, err := filepath.Glob(filepath.Join(servicePath, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list YML files: %w", err)
	}
	yamlFiles = append(yamlFiles, ymlFiles...)

	fp.logger.Debug("Found YAML files in service folder",
		zap.String("service", serviceName),
		zap.Int("count", len(yamlFiles)),
		zap.Strings("files", yamlFiles))

	// Parse each YAML file
	for _, yamlFile := range yamlFiles {
		fp.logger.Debug("Parsing YAML file", zap.String("file", yamlFile))
		
		resources, err := fp.parseServiceYAMLFile(yamlFile, stack, serviceName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", filepath.Base(yamlFile), err)
		}

		fp.logger.Debug("Parsed resources from file",
			zap.String("file", filepath.Base(yamlFile)),
			zap.Int("count", len(resources)))

		for _, res := range resources {
			if res == nil {
				continue
			}
			fp.logger.Debug("Processing resource",
				zap.String("kind", string(res.GetKind())),
				zap.String("name", res.GetMetadata().Name))

			switch r := res.(type) {
			case *schema.Service:
				if result.Service != nil {
					return nil, fmt.Errorf("multiple Service definitions in %s", servicePath)
				}
				result.Service = r
				fp.logger.Debug("Found Service definition", zap.String("name", r.Metadata.Name))

				// Store service variables
				if r.Spec.Variables != nil {
					for k, v := range r.Spec.Variables {
						fp.parser.SetVariable(fmt.Sprintf("%s.%s", serviceName, k), v)
					}
				}
			default:
				result.Components = append(result.Components, res)
			}
		}
	}

	// Load config files from config/ subfolder
	configPath := filepath.Join(servicePath, "config")
	if info, err := os.Stat(configPath); err == nil && info.IsDir() {
		result.ConfigFiles, _ = fp.loadConfigFiles(configPath)
	}

	// Validate service has a service.yaml
	if result.Service == nil {
		// Auto-create a minimal service definition
		result.Service = schema.NewService(serviceName, stack.Metadata.Name)
		fp.logger.Warn("No service.yaml found, created default",
			zap.String("service", serviceName))
	}

	return result, nil
}

// parseServiceYAMLFile parses a YAML file within a service folder
func (fp *FolderParser) parseServiceYAMLFile(path string, stack *schema.Stack, serviceName string) ([]schema.Resource, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Split multi-document YAML
	docs := fp.splitDocuments(content)
	resources := make([]schema.Resource, 0, len(docs))

	for i, doc := range docs {
		resource, err := fp.parseDocument(doc, stack, serviceName)
		if err != nil {
			return nil, fmt.Errorf("document %d: %w", i+1, err)
		}
		if resource != nil {
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// splitDocuments splits multi-document YAML (same as Parser)
func (fp *FolderParser) splitDocuments(content []byte) [][]byte {
	parts := strings.Split(string(content), "\n---")

	docs := make([][]byte, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		// Check if this is a comment-only document
		lines := strings.Split(trimmed, "\n")
		hasContent := false
		for _, line := range lines {
			lineContent := strings.TrimSpace(line)
			if lineContent != "" && !strings.HasPrefix(lineContent, "#") {
				hasContent = true
				break
			}
		}

		if hasContent {
			docs = append(docs, []byte(trimmed))
		}
	}

	return docs
}

// parseDocument parses a single YAML document
func (fp *FolderParser) parseDocument(content []byte, stack *schema.Stack, serviceName string) (schema.Resource, error) {
	// First, parse ResourceBase to determine kind
	var base schema.ResourceBase
	if err := yaml.Unmarshal(content, &base); err != nil {
		return nil, fmt.Errorf("failed to parse resource: %w", err)
	}

	// Skip empty documents
	if base.Kind == "" {
		return nil, nil
	}

	// Interpolate variables
	interpolated := fp.parser.interpolateContent(content)

	// Parse based on kind
	var resource schema.Resource
	var err error

	switch base.Kind {
	case schema.KindService:
		var svc schema.Service
		err = yaml.Unmarshal(interpolated, &svc)
		if err == nil {
			// Ensure stack reference is set
			if svc.Metadata.Stack == "" {
				svc.Metadata.Stack = stack.Metadata.Name
			}
			resource = &svc
		}

	case schema.KindMicroService:
		var ms schema.MicroService
		err = yaml.Unmarshal(interpolated, &ms)
		if err == nil {
			fp.setComponentMetadata(&ms.ResourceBase, stack, serviceName)
			resource = &ms
		}

	case schema.KindRDS:
		var rds schema.RDS
		err = yaml.Unmarshal(interpolated, &rds)
		if err == nil {
			fp.setComponentMetadata(&rds.ResourceBase, stack, serviceName)
			resource = &rds
		}

	case schema.KindDynamoDB:
		var dynamo schema.DynamoDB
		err = yaml.Unmarshal(interpolated, &dynamo)
		if err == nil {
			fp.setComponentMetadata(&dynamo.ResourceBase, stack, serviceName)
			resource = &dynamo
		}

	case schema.KindS3:
		var s3 schema.S3
		err = yaml.Unmarshal(interpolated, &s3)
		if err == nil {
			fp.setComponentMetadata(&s3.ResourceBase, stack, serviceName)
			resource = &s3
		}

	case schema.KindSQS:
		var sqs schema.SQS
		err = yaml.Unmarshal(interpolated, &sqs)
		if err == nil {
			fp.setComponentMetadata(&sqs.ResourceBase, stack, serviceName)
			resource = &sqs
		}

	case schema.KindSNS:
		var sns schema.SNS
		err = yaml.Unmarshal(interpolated, &sns)
		if err == nil {
			fp.setComponentMetadata(&sns.ResourceBase, stack, serviceName)
			resource = &sns
		}

	case schema.KindLambda:
		var lambda schema.Lambda
		err = yaml.Unmarshal(interpolated, &lambda)
		if err == nil {
			fp.setComponentMetadata(&lambda.ResourceBase, stack, serviceName)
			resource = &lambda
		}

	case schema.KindComponentInfra:
		var infra schema.ComponentInfra
		err = yaml.Unmarshal(interpolated, &infra)
		if err == nil {
			fp.setComponentMetadata(&infra.ResourceBase, stack, serviceName)
			resource = &infra
		}

	default:
		// Try to parse as generic component
		fp.logger.Warn("Unknown resource kind", zap.String("kind", string(base.Kind)))
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", base.Kind, err)
	}

	return resource, nil
}

// setComponentMetadata sets stack and service references on a component
func (fp *FolderParser) setComponentMetadata(base *schema.ResourceBase, stack *schema.Stack, serviceName string) {
	if base.Metadata.Stack == "" {
		base.Metadata.Stack = stack.Metadata.Name
	}
	if base.Metadata.Service == "" {
		base.Metadata.Service = serviceName
	}
}

// interpolateContent performs variable interpolation on content
func (p *Parser) interpolateContent(content []byte) []byte {
	result, _ := p.interpolateVariables(content)
	return result
}

// loadConfigFiles loads non-YAML config files from a directory
func (fp *FolderParser) loadConfigFiles(configPath string) (map[string][]byte, error) {
	configs := make(map[string][]byte)

	entries, err := os.ReadDir(configPath)
	if err != nil {
		return configs, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(configPath, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		configs[entry.Name()] = content
	}

	return configs, nil
}

// validateCrossReferences validates that all component references exist
func (fp *FolderParser) validateCrossReferences(result *StackParseResult) error {
	// Build map of all component names
	componentNames := make(map[string]bool)
	for _, comp := range result.AllComponents {
		componentNames[comp.GetMetadata().Name] = true
	}

	// Check dependencies
	var errors []string
	for _, comp := range result.AllComponents {
		deps := fp.extractDependencies(comp)
		for _, dep := range deps {
			if !componentNames[dep] {
				errors = append(errors, fmt.Sprintf(
					"component %s depends on non-existent component: %s",
					comp.GetMetadata().Name, dep))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cross-reference errors:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// extractDependencies extracts dependency names from a component
func (fp *FolderParser) extractDependencies(resource schema.Resource) []string {
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
	case *schema.Lambda:
		return r.Spec.DependsOn
	default:
		return nil
	}
}

// GetServiceNames returns the names of all services in the result
func (r *StackParseResult) GetServiceNames() []string {
	names := make([]string, 0, len(r.Services))
	for name := range r.Services {
		names = append(names, name)
	}
	return names
}

// GetComponentsByService returns components grouped by service
func (r *StackParseResult) GetComponentsByService() map[string][]schema.Resource {
	result := make(map[string][]schema.Resource)
	for serviceName, svc := range r.Services {
		result[serviceName] = svc.Components
	}
	return result
}

// GetComponentByName finds a component by name across all services
func (r *StackParseResult) GetComponentByName(name string) schema.Resource {
	for _, comp := range r.AllComponents {
		if comp.GetMetadata().Name == name {
			return comp
		}
	}
	return nil
}

// Summary returns a summary of the parsed stack
func (r *StackParseResult) Summary() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Stack: %s\n", r.Stack.Metadata.Name))
	sb.WriteString(fmt.Sprintf("  Path: %s\n", r.StackPath))
	sb.WriteString(fmt.Sprintf("  Services: %d\n", len(r.Services)))
	sb.WriteString(fmt.Sprintf("  Components: %d\n", len(r.AllComponents)))

	for serviceName, svc := range r.Services {
		sb.WriteString(fmt.Sprintf("\n  Service: %s\n", serviceName))
		sb.WriteString(fmt.Sprintf("    Components: %d\n", len(svc.Components)))
		for _, comp := range svc.Components {
			sb.WriteString(fmt.Sprintf("      - %s (%s)\n", comp.GetMetadata().Name, comp.GetKind()))
		}
	}

	if len(r.Warnings) > 0 {
		sb.WriteString("\n  Warnings:\n")
		for _, w := range r.Warnings {
			sb.WriteString(fmt.Sprintf("    - %s\n", w))
		}
	}

	return sb.String()
}

