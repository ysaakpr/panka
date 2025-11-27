package parser

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/parser/schema"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Parser handles parsing and validation of Panka YAML files
type Parser struct {
	logger *logger.Logger
	
	// Variables for interpolation
	variables map[string]string
	
	// Component outputs for cross-reference
	componentOutputs map[string]map[string]string
}

// NewParser creates a new parser instance
func NewParser() *Parser {
	log, _ := logger.NewDevelopment()
	return &Parser{
		logger:           log,
		variables:        make(map[string]string),
		componentOutputs: make(map[string]map[string]string),
	}
}

// ParseResult contains the parsed resources
type ParseResult struct {
	Stack      *schema.Stack
	Services   []*schema.Service
	Components []schema.Resource
}

// ParseFile parses a YAML file and returns all resources
func (p *Parser) ParseFile(path string) (*ParseResult, error) {
	p.logger.Info("Parsing file", zap.String("path", path))
	
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	return p.Parse(content)
}

// Parse parses YAML content and returns all resources
func (p *Parser) Parse(content []byte) (*ParseResult, error) {
	// Split multi-document YAML
	docs := p.splitDocuments(content)
	
	result := &ParseResult{
		Services:   make([]*schema.Service, 0),
		Components: make([]schema.Resource, 0),
	}
	
	// Parse each document
	for i, doc := range docs {
		resource, err := p.parseDocument(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to parse document %d: %w", i, err)
		}
		
		// Categorize the resource
		switch r := resource.(type) {
		case *schema.Stack:
			if result.Stack != nil {
				return nil, fmt.Errorf("multiple Stack definitions found")
			}
			result.Stack = r
			
			// Store stack variables
			for k, v := range r.Spec.Variables {
				p.variables[k] = v
			}
			
		case *schema.Service:
			result.Services = append(result.Services, r)
			
			// Store service variables
			for k, v := range r.Spec.Variables {
				p.variables[fmt.Sprintf("%s.%s", r.Metadata.Name, k)] = v
			}
			
		default:
			result.Components = append(result.Components, r)
		}
	}
	
	// Validate result
	if result.Stack == nil {
		return nil, fmt.Errorf("no Stack definition found")
	}
	
	return result, nil
}

// splitDocuments splits multi-document YAML
func (p *Parser) splitDocuments(content []byte) [][]byte {
	// Split by "---" document separator
	parts := strings.Split(string(content), "\n---\n")
	
	docs := make([][]byte, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			docs = append(docs, []byte(trimmed))
		}
	}
	
	return docs
}

// parseDocument parses a single YAML document
func (p *Parser) parseDocument(content []byte) (schema.Resource, error) {
	// First, parse just the ResourceBase to determine the kind
	var base schema.ResourceBase
	if err := yaml.Unmarshal(content, &base); err != nil {
		return nil, fmt.Errorf("failed to parse resource base: %w", err)
	}
	
	p.logger.Debug("Parsing resource", zap.String("kind", string(base.Kind)), zap.String("name", base.Metadata.Name))
	
	// Interpolate variables in the content
	interpolated, err := p.interpolateVariables(content)
	if err != nil {
		return nil, fmt.Errorf("failed to interpolate variables: %w", err)
	}
	
	// Parse based on kind
	var resource schema.Resource
	switch base.Kind {
	case schema.KindStack:
		var stack schema.Stack
		if err := yaml.Unmarshal(interpolated, &stack); err != nil {
			return nil, fmt.Errorf("failed to parse Stack: %w", err)
		}
		resource = &stack
		
	case schema.KindService:
		var service schema.Service
		if err := yaml.Unmarshal(interpolated, &service); err != nil {
			return nil, fmt.Errorf("failed to parse Service: %w", err)
		}
		resource = &service
		
	case schema.KindMicroService:
		var ms schema.MicroService
		if err := yaml.Unmarshal(interpolated, &ms); err != nil {
			return nil, fmt.Errorf("failed to parse MicroService: %w", err)
		}
		resource = &ms
		
	case schema.KindComponentInfra:
		var infra schema.ComponentInfra
		if err := yaml.Unmarshal(interpolated, &infra); err != nil {
			return nil, fmt.Errorf("failed to parse ComponentInfra: %w", err)
		}
		resource = &infra
		
	case schema.KindRDS:
		var rds schema.RDS
		if err := yaml.Unmarshal(interpolated, &rds); err != nil {
			return nil, fmt.Errorf("failed to parse RDS: %w", err)
		}
		resource = &rds
		
	case schema.KindDynamoDB:
		var dynamo schema.DynamoDB
		if err := yaml.Unmarshal(interpolated, &dynamo); err != nil {
			return nil, fmt.Errorf("failed to parse DynamoDB: %w", err)
		}
		resource = &dynamo
		
	case schema.KindS3:
		var s3 schema.S3
		if err := yaml.Unmarshal(interpolated, &s3); err != nil {
			return nil, fmt.Errorf("failed to parse S3: %w", err)
		}
		resource = &s3
		
	case schema.KindSQS:
		var sqs schema.SQS
		if err := yaml.Unmarshal(interpolated, &sqs); err != nil {
			return nil, fmt.Errorf("failed to parse SQS: %w", err)
		}
		resource = &sqs
		
	case schema.KindSNS:
		var sns schema.SNS
		if err := yaml.Unmarshal(interpolated, &sns); err != nil {
			return nil, fmt.Errorf("failed to parse SNS: %w", err)
		}
		resource = &sns
		
	default:
		return nil, fmt.Errorf("unsupported resource kind: %s", base.Kind)
	}
	
	// Validate the resource
	if err := resource.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed for %s/%s: %w", 
			resource.GetKind(), resource.GetMetadata().Name, err)
	}
	
	return resource, nil
}

// interpolateVariables replaces variable references with their values
func (p *Parser) interpolateVariables(content []byte) ([]byte, error) {
	str := string(content)
	
	// Pattern: ${VAR_NAME} or ${component.output}
	varPattern := regexp.MustCompile(`\$\{([^}]+)\}`)
	
	result := varPattern.ReplaceAllStringFunc(str, func(match string) string {
		// Extract variable name (remove ${ and })
		varName := match[2 : len(match)-1]
		
		// First check regular variables (including dotted names like service.var)
		if value, ok := p.variables[varName]; ok {
			return value
		}
		
		// Check for component output reference (component.output)
		if strings.Contains(varName, ".") {
			parts := strings.SplitN(varName, ".", 2)
			componentName := parts[0]
			outputName := parts[1]
			
			if outputs, ok := p.componentOutputs[componentName]; ok {
				if value, ok := outputs[outputName]; ok {
					return value
				}
			}
			
			// Return original if not found (will be resolved later)
			return match
		}
		
		// Return original if not found
		return match
	})
	
	return []byte(result), nil
}

// SetVariable sets a variable for interpolation
func (p *Parser) SetVariable(name, value string) {
	p.variables[name] = value
}

// SetComponentOutput sets a component output for cross-reference
func (p *Parser) SetComponentOutput(component, output, value string) {
	if _, ok := p.componentOutputs[component]; !ok {
		p.componentOutputs[component] = make(map[string]string)
	}
	p.componentOutputs[component][output] = value
}

// ValidateCrossReferences validates that all component references exist
func (p *Parser) ValidateCrossReferences(result *ParseResult) error {
	componentNames := make(map[string]bool)
	
	// Collect all component names
	for _, comp := range result.Components {
		componentNames[comp.GetMetadata().Name] = true
	}
	
	// Validate dependencies
	for _, comp := range result.Components {
		deps := p.extractDependencies(comp)
		for _, dep := range deps {
			if !componentNames[dep] {
				return fmt.Errorf("component %s depends on non-existent component: %s",
					comp.GetMetadata().Name, dep)
			}
		}
	}
	
	return nil
}

// extractDependencies extracts dependency names from a component
func (p *Parser) extractDependencies(resource schema.Resource) []string {
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

