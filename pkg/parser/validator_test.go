package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yourusername/panka/pkg/parser/schema"
)

func TestValidator_ValidStack(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("my-stack"),
		Services: []*schema.Service{
			schema.NewService("backend", "my-stack"),
		},
		Components: []schema.Resource{
			schema.NewMicroService("api", "backend", "my-stack"),
		},
	}
	
	// Setup valid stack
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	// Setup valid microservice
	ms := result.Components[0].(*schema.MicroService)
	ms.Spec.Image.Repository = "myrepo/api"
	ms.Spec.Image.Tag = "v1.0.0"
	ms.Spec.Runtime.Platform = "fargate"
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.NoError(t, err)
}

func TestValidator_InvalidStackName(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("My_Stack"), // Invalid: uppercase and underscore
		Services: []*schema.Service{
			schema.NewService("backend", "My_Stack"),
		},
		Components: []schema.Resource{
			schema.NewMicroService("api", "backend", "My_Stack"),
		},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	ms := result.Components[0].(*schema.MicroService)
	ms.Spec.Image.Repository = "myrepo/api"
	ms.Spec.Image.Tag = "v1.0.0"
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestValidator_MissingProvider(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("backend", "test-stack"),
		},
		Components: []schema.Resource{
			schema.NewMicroService("api", "backend", "test-stack"),
		},
	}
	
	// Provider name is empty (invalid)
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	ms := result.Components[0].(*schema.MicroService)
	ms.Spec.Image.Repository = "myrepo/api"
	ms.Spec.Image.Tag = "v1.0.0"
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider name")
}

func TestValidator_ServiceWithoutComponents(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("backend", "test-stack"),
		},
		Components: []schema.Resource{},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has no components")
}

func TestValidator_ComponentWithInvalidService(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("backend", "test-stack"),
		},
		Components: []schema.Resource{
			schema.NewMicroService("api", "non-existent-service", "test-stack"),
		},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	ms := result.Components[0].(*schema.MicroService)
	ms.Spec.Image.Repository = "myrepo/api"
	ms.Spec.Image.Tag = "v1.0.0"
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown service")
}

func TestValidator_MicroServiceValidation(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("backend", "test-stack"),
		},
		Components: []schema.Resource{
			schema.NewMicroService("api", "backend", "test-stack"),
		},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	// Missing image repository and tag
	ms := result.Components[0].(*schema.MicroService)
	ms.Spec.Runtime.Platform = "fargate"
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image")
}

func TestValidator_RDSValidation(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("data", "test-stack"),
		},
		Components: []schema.Resource{
			schema.NewRDS("db", "data", "test-stack"),
		},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	rds := result.Components[0].(*schema.RDS)
	rds.Spec.Engine.Type = "postgres"
	rds.Spec.Engine.Version = "14.7"
	rds.Spec.Instance.Class = "db.t3.medium"
	rds.Spec.Instance.Storage.Type = "gp3"
	rds.Spec.Instance.Storage.AllocatedGB = 100
	rds.Spec.Database.Name = "appdb"
	rds.Spec.Database.Username = "admin"
	rds.Spec.Database.PasswordSecret.Ref = "db-password"
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.NoError(t, err)
}

func TestValidator_RDSInvalidStorage(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("data", "test-stack"),
		},
		Components: []schema.Resource{
			schema.NewRDS("db", "data", "test-stack"),
		},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	rds := result.Components[0].(*schema.RDS)
	rds.Spec.Engine.Type = "postgres"
	rds.Spec.Engine.Version = "14.7"
	rds.Spec.Instance.Class = "db.t3.medium"
	rds.Spec.Instance.Storage.Type = "gp3"
	rds.Spec.Instance.Storage.AllocatedGB = 10 // Too small (< 20GB)
	rds.Spec.Database.Name = "appdb"
	rds.Spec.Database.Username = "admin"
	rds.Spec.Database.PasswordSecret.Ref = "db-password"
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "minimum allocated storage")
}

func TestValidator_CircularDependencies(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("backend", "test-stack"),
		},
		Components: []schema.Resource{},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	// Create circular dependency: api -> worker -> api
	api := schema.NewMicroService("api", "backend", "test-stack")
	api.Spec.Image.Repository = "myrepo/api"
	api.Spec.Image.Tag = "v1.0.0"
	api.Spec.DependsOn = []string{"worker"}
	
	worker := schema.NewMicroService("worker", "backend", "test-stack")
	worker.Spec.Image.Repository = "myrepo/worker"
	worker.Spec.Image.Tag = "v1.0.0"
	worker.Spec.DependsOn = []string{"api"}
	
	result.Components = []schema.Resource{api, worker}
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestValidator_ValidDependencies(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("backend", "test-stack"),
		},
		Components: []schema.Resource{},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	// Create valid dependency chain: api -> db
	db := schema.NewRDS("db", "backend", "test-stack")
	db.Spec.Engine.Type = "postgres"
	db.Spec.Engine.Version = "14.7"
	db.Spec.Instance.Class = "db.t3.medium"
	db.Spec.Instance.Storage.Type = "gp3"
	db.Spec.Instance.Storage.AllocatedGB = 100
	db.Spec.Database.Name = "appdb"
	db.Spec.Database.Username = "admin"
	db.Spec.Database.PasswordSecret.Ref = "db-password"
	
	api := schema.NewMicroService("api", "backend", "test-stack")
	api.Spec.Image.Repository = "myrepo/api"
	api.Spec.Image.Tag = "v1.0.0"
	api.Spec.DependsOn = []string{"db"}
	
	result.Components = []schema.Resource{db, api}
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.NoError(t, err)
}

func TestValidator_DuplicatePortNames(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("backend", "test-stack"),
		},
		Components: []schema.Resource{},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	api := schema.NewMicroService("api", "backend", "test-stack")
	api.Spec.Image.Repository = "myrepo/api"
	api.Spec.Image.Tag = "v1.0.0"
	api.Spec.Ports = []schema.Port{
		{Name: "http", Port: 8080, Protocol: "tcp"},
		{Name: "http", Port: 9090, Protocol: "tcp"}, // Duplicate name
	}
	
	result.Components = []schema.Resource{api}
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate port name")
}

func TestValidator_isValidName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid lowercase", "my-stack", true},
		{"valid with numbers", "stack123", true},
		{"valid with hyphens", "my-test-stack", true},
		{"invalid uppercase", "MyStack", false},
		{"invalid underscore", "my_stack", false},
		{"invalid starts with number", "1stack", false},
		{"invalid starts with hyphen", "-stack", false},
		{"invalid special chars", "stack@123", false},
		{"valid single char", "a", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidator_DynamoDBValidation(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("data", "test-stack"),
		},
		Components: []schema.Resource{},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	dynamo := schema.NewDynamoDB("sessions", "data", "test-stack")
	dynamo.Spec.BillingMode = "PAY_PER_REQUEST"
	dynamo.Spec.HashKey = schema.AttributeDefinition{Name: "id", Type: "S"}
	
	result.Components = []schema.Resource{dynamo}
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.NoError(t, err)
}

func TestValidator_DynamoDBInvalidBillingMode(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("data", "test-stack"),
		},
		Components: []schema.Resource{},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	dynamo := schema.NewDynamoDB("sessions", "data", "test-stack")
	dynamo.Spec.BillingMode = "INVALID_MODE"
	dynamo.Spec.HashKey = schema.AttributeDefinition{Name: "id", Type: "S"}
	
	result.Components = []schema.Resource{dynamo}
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid billing mode")
}

func TestValidator_S3Validation(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("storage", "test-stack"),
		},
		Components: []schema.Resource{},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	s3 := schema.NewS3("uploads", "storage", "test-stack")
	s3.Spec.Bucket.ACL = "private"
	
	result.Components = []schema.Resource{s3}
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.NoError(t, err)
}

func TestValidator_S3InvalidACL(t *testing.T) {
	result := &ParseResult{
		Stack: schema.NewStack("test-stack"),
		Services: []*schema.Service{
			schema.NewService("storage", "test-stack"),
		},
		Components: []schema.Resource{},
	}
	
	result.Stack.Spec.Provider.Name = "aws"
	result.Stack.Spec.Provider.Region = "us-east-1"
	
	s3 := schema.NewS3("uploads", "storage", "test-stack")
	s3.Spec.Bucket.ACL = "invalid-acl"
	
	result.Components = []schema.Resource{s3}
	
	validator := NewValidator()
	err := validator.Validate(result)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid ACL")
}

