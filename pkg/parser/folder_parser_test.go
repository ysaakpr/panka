package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/panka/pkg/parser/schema"
)

func TestFolderParser_ParseStackFolder(t *testing.T) {
	// Create a temporary stack folder
	tmpDir := t.TempDir()

	// Create stack.yaml
	stackYAML := `apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: test-stack
  tenant: test-tenant
spec:
  provider:
    name: aws
    region: us-east-1
  variables:
    ENV: production
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "stack.yaml"), []byte(stackYAML), 0644))

	// Create services folder
	servicesDir := filepath.Join(tmpDir, "services")
	require.NoError(t, os.MkdirAll(servicesDir, 0755))

	// Create api service folder
	apiDir := filepath.Join(servicesDir, "api")
	require.NoError(t, os.MkdirAll(apiDir, 0755))

	// Create service.yaml for api
	apiServiceYAML := `apiVersion: core.panka.io/v1
kind: Service
metadata:
  name: api
  stack: test-stack
spec:
  variables:
    PORT: "8080"
`
	require.NoError(t, os.WriteFile(filepath.Join(apiDir, "service.yaml"), []byte(apiServiceYAML), 0644))

	// Create components for api
	apiResourcesYAML := `apiVersion: components.panka.io/v1
kind: SQS
metadata:
  name: api-queue
  service: api
  stack: test-stack
spec:
  type: standard
---
apiVersion: components.panka.io/v1
kind: S3
metadata:
  name: api-bucket
  service: api
  stack: test-stack
spec:
  versioning:
    enabled: true
`
	require.NoError(t, os.WriteFile(filepath.Join(apiDir, "resources.yaml"), []byte(apiResourcesYAML), 0644))

	// Parse the stack folder
	fp := NewFolderParser()
	result, err := fp.ParseStackFolder(tmpDir)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check stack
	assert.Equal(t, "test-stack", result.Stack.Metadata.Name)
	assert.Equal(t, "test-tenant", result.Stack.Metadata.Tenant)

	// Check services
	assert.Len(t, result.Services, 1)
	assert.Contains(t, result.Services, "api")

	// Check api service
	apiSvc := result.Services["api"]
	require.NotNil(t, apiSvc.Service)
	assert.Equal(t, "api", apiSvc.Service.Metadata.Name)
	assert.Len(t, apiSvc.Components, 2)

	// Check total components
	assert.Len(t, result.AllComponents, 2)

	// Check component types
	var foundSQS, foundS3 bool
	for _, comp := range result.AllComponents {
		switch comp.GetKind() {
		case schema.KindSQS:
			foundSQS = true
			assert.Equal(t, "api-queue", comp.GetMetadata().Name)
		case schema.KindS3:
			foundS3 = true
			assert.Equal(t, "api-bucket", comp.GetMetadata().Name)
		}
	}
	assert.True(t, foundSQS, "SQS component not found")
	assert.True(t, foundS3, "S3 component not found")
}

func TestFolderParser_InvalidFolder(t *testing.T) {
	fp := NewFolderParser()

	// Test with non-existent folder
	_, err := fp.ParseStackFolder("/non/existent/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestFolderParser_MissingStackYAML(t *testing.T) {
	tmpDir := t.TempDir()
	fp := NewFolderParser()

	// Test folder without stack.yaml
	_, err := fp.ParseStackFolder(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stack.yaml not found")
}

func TestFolderParser_NoServices(t *testing.T) {
	tmpDir := t.TempDir()

	// Create stack.yaml only
	stackYAML := `apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: empty-stack
spec:
  provider:
    name: aws
    region: us-east-1
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "stack.yaml"), []byte(stackYAML), 0644))

	fp := NewFolderParser()
	result, err := fp.ParseStackFolder(tmpDir)

	// Should succeed but with no services
	require.NoError(t, err)
	assert.Equal(t, "empty-stack", result.Stack.Metadata.Name)
	assert.Len(t, result.Services, 0)
	assert.Len(t, result.AllComponents, 0)
}

func TestFolderParser_LambdaComponents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create stack.yaml
	stackYAML := `apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: lambda-stack
spec:
  provider:
    name: aws
    region: us-east-1
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "stack.yaml"), []byte(stackYAML), 0644))

	// Create services folder
	servicesDir := filepath.Join(tmpDir, "services", "worker")
	require.NoError(t, os.MkdirAll(servicesDir, 0755))

	// Create Lambda component
	lambdaYAML := `apiVersion: components.panka.io/v1
kind: Lambda
metadata:
  name: processor
  service: worker
  stack: lambda-stack
spec:
  runtime: nodejs18.x
  handler: index.handler
  code:
    s3Bucket: my-bucket
    s3Key: code.zip
`
	require.NoError(t, os.WriteFile(filepath.Join(servicesDir, "lambda.yaml"), []byte(lambdaYAML), 0644))

	fp := NewFolderParser()
	result, err := fp.ParseStackFolder(tmpDir)

	require.NoError(t, err)
	assert.Len(t, result.AllComponents, 1)

	// Check Lambda was parsed
	lambda := result.AllComponents[0]
	assert.Equal(t, schema.KindLambda, lambda.GetKind())
	assert.Equal(t, "processor", lambda.GetMetadata().Name)
}

func TestStackParseResult_GetComponentByName(t *testing.T) {
	result := &StackParseResult{
		AllComponents: []schema.Resource{
			&schema.SQS{
				ResourceBase: schema.ResourceBase{
					Kind:     schema.KindSQS,
					Metadata: schema.Metadata{Name: "my-queue"},
				},
			},
			&schema.S3{
				ResourceBase: schema.ResourceBase{
					Kind:     schema.KindS3,
					Metadata: schema.Metadata{Name: "my-bucket"},
				},
			},
		},
	}

	// Find existing component
	comp := result.GetComponentByName("my-queue")
	require.NotNil(t, comp)
	assert.Equal(t, "my-queue", comp.GetMetadata().Name)

	// Find non-existent component
	comp = result.GetComponentByName("nonexistent")
	assert.Nil(t, comp)
}

