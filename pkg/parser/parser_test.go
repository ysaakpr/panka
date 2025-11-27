package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/panka/pkg/parser/schema"
)

func TestParser_Parse_SimpleStack(t *testing.T) {
	yaml := `
apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: my-stack
spec:
  provider:
    name: aws
    region: us-east-1
    accountId: "123456789012"
  variables:
    VERSION: "1.0.0"
    ENVIRONMENT: "dev"
---
apiVersion: core.panka.io/v1
kind: Service
metadata:
  name: api-service
  stack: my-stack
spec:
  variables:
    PORT: "8080"
`
	
	parser := NewParser()
	result, err := parser.Parse([]byte(yaml))
	
	require.NoError(t, err)
	require.NotNil(t, result.Stack)
	assert.Equal(t, "my-stack", result.Stack.Metadata.Name)
	assert.Equal(t, "aws", result.Stack.Spec.Provider.Name)
	assert.Equal(t, "us-east-1", result.Stack.Spec.Provider.Region)
	assert.Equal(t, "1.0.0", result.Stack.Spec.Variables["VERSION"])
	
	require.Len(t, result.Services, 1)
	assert.Equal(t, "api-service", result.Services[0].Metadata.Name)
	assert.Equal(t, "my-stack", result.Services[0].Metadata.Stack)
}

func TestParser_Parse_MicroService(t *testing.T) {
	yaml := `
apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: test-stack
spec:
  provider:
    name: aws
    region: us-west-2
---
apiVersion: core.panka.io/v1
kind: Service
metadata:
  name: backend
  stack: test-stack
---
apiVersion: components.panka.io/v1
kind: MicroService
metadata:
  name: api
  service: backend
  stack: test-stack
spec:
  image:
    repository: myrepo/api
    tag: "v1.0.0"
    pullPolicy: Always
  runtime:
    platform: fargate
  ports:
    - name: http
      port: 8080
      protocol: tcp
  environment:
    - name: DATABASE_URL
      value: "postgres://localhost/db"
    - name: LOG_LEVEL
      value: "info"
`
	
	parser := NewParser()
	result, err := parser.Parse([]byte(yaml))
	
	require.NoError(t, err)
	require.Len(t, result.Components, 1)
	
	ms, ok := result.Components[0].(*schema.MicroService)
	require.True(t, ok)
	assert.Equal(t, "api", ms.Metadata.Name)
	assert.Equal(t, "backend", ms.Metadata.Service)
	assert.Equal(t, "myrepo/api", ms.Spec.Image.Repository)
	assert.Equal(t, "v1.0.0", ms.Spec.Image.Tag)
	assert.Equal(t, "fargate", ms.Spec.Runtime.Platform)
	assert.Len(t, ms.Spec.Ports, 1)
	assert.Len(t, ms.Spec.Environment, 2)
}

func TestParser_Parse_RDS(t *testing.T) {
	yaml := `
apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: test-stack
spec:
  provider:
    name: aws
    region: us-west-2
---
apiVersion: core.panka.io/v1
kind: Service
metadata:
  name: data
  stack: test-stack
---
apiVersion: components.panka.io/v1
kind: RDS
metadata:
  name: main-db
  service: data
  stack: test-stack
spec:
  engine:
    type: postgres
    version: "14.7"
  instance:
    class: db.t3.medium
    storage:
      type: gp3
      allocatedGB: 100
    multiAZ: true
  database:
    name: appdb
    username: dbadmin
    passwordSecret:
      ref: db-password
  backup:
    enabled: true
    retentionDays: 7
`
	
	parser := NewParser()
	result, err := parser.Parse([]byte(yaml))
	
	require.NoError(t, err)
	require.Len(t, result.Components, 1)
	
	rds, ok := result.Components[0].(*schema.RDS)
	require.True(t, ok)
	assert.Equal(t, "main-db", rds.Metadata.Name)
	assert.Equal(t, "data", rds.Metadata.Service)
	assert.Equal(t, "postgres", rds.Spec.Engine.Type)
	assert.Equal(t, "14.7", rds.Spec.Engine.Version)
	assert.Equal(t, "db.t3.medium", rds.Spec.Instance.Class)
	assert.Equal(t, 100, rds.Spec.Instance.Storage.AllocatedGB)
	assert.True(t, rds.Spec.Instance.MultiAZ)
	assert.True(t, rds.Spec.Backup.Enabled)
	assert.Equal(t, 7, rds.Spec.Backup.RetentionDays)
}

func TestParser_VariableInterpolation(t *testing.T) {
	yaml := `
apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: test-stack
spec:
  provider:
    name: aws
    region: us-west-2
  variables:
    VERSION: "2.5.0"
---
apiVersion: core.panka.io/v1
kind: Service
metadata:
  name: backend
  stack: test-stack
spec:
  variables:
    IMAGE_REPO: "myregistry/backend"
---
apiVersion: components.panka.io/v1
kind: MicroService
metadata:
  name: api
  service: backend
  stack: test-stack
spec:
  image:
    repository: ${backend.IMAGE_REPO}
    tag: ${VERSION}
  runtime:
    platform: fargate
`
	
	parser := NewParser()
	result, err := parser.Parse([]byte(yaml))
	
	require.NoError(t, err)
	require.Len(t, result.Components, 1)
	
	ms, ok := result.Components[0].(*schema.MicroService)
	require.True(t, ok)
	// Variables should be interpolated
	assert.Equal(t, "myregistry/backend", ms.Spec.Image.Repository)
	assert.Equal(t, "2.5.0", ms.Spec.Image.Tag)
}

func TestParser_MultipleServices(t *testing.T) {
	yaml := `
apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: multi-stack
spec:
  provider:
    name: aws
    region: us-east-1
---
apiVersion: core.panka.io/v1
kind: Service
metadata:
  name: frontend
  stack: multi-stack
---
apiVersion: core.panka.io/v1
kind: Service
metadata:
  name: backend
  stack: multi-stack
---
apiVersion: core.panka.io/v1
kind: Service
metadata:
  name: data
  stack: multi-stack
`
	
	parser := NewParser()
	result, err := parser.Parse([]byte(yaml))
	
	require.NoError(t, err)
	require.NotNil(t, result.Stack)
	require.Len(t, result.Services, 3)
	
	serviceNames := make([]string, len(result.Services))
	for i, svc := range result.Services {
		serviceNames[i] = svc.Metadata.Name
	}
	
	assert.Contains(t, serviceNames, "frontend")
	assert.Contains(t, serviceNames, "backend")
	assert.Contains(t, serviceNames, "data")
}

func TestParser_MissingStack(t *testing.T) {
	yaml := `
apiVersion: core.panka.io/v1
kind: Service
metadata:
  name: orphan-service
  stack: missing-stack
`
	
	parser := NewParser()
	_, err := parser.Parse([]byte(yaml))
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Stack definition found")
}

func TestParser_MultipleStacks(t *testing.T) {
	yaml := `
apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: stack-1
spec:
  provider:
    name: aws
    region: us-east-1
---
apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: stack-2
spec:
  provider:
    name: aws
    region: us-west-2
`
	
	parser := NewParser()
	_, err := parser.Parse([]byte(yaml))
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple Stack definitions found")
}

func TestParser_InvalidKind(t *testing.T) {
	yaml := `
apiVersion: core.panka.io/v1
kind: InvalidKind
metadata:
  name: test
`
	
	parser := NewParser()
	_, err := parser.Parse([]byte(yaml))
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource kind")
}

func TestParser_SetVariable(t *testing.T) {
	parser := NewParser()
	parser.SetVariable("MY_VAR", "my_value")
	
	assert.Equal(t, "my_value", parser.variables["MY_VAR"])
}

func TestParser_SetComponentOutput(t *testing.T) {
	parser := NewParser()
	parser.SetComponentOutput("db", "endpoint", "db.example.com:5432")
	
	assert.Equal(t, "db.example.com:5432", parser.componentOutputs["db"]["endpoint"])
}

func TestParser_SplitDocuments(t *testing.T) {
	yaml := `
doc1: value1
---
doc2: value2
---
# Comment only document
---
doc3: value3
`
	
	parser := NewParser()
	docs := parser.splitDocuments([]byte(yaml))
	
	assert.Len(t, docs, 3)
	assert.Contains(t, string(docs[0]), "doc1")
	assert.Contains(t, string(docs[1]), "doc2")
	assert.Contains(t, string(docs[2]), "doc3")
}

func TestParser_DynamoDB(t *testing.T) {
	yaml := `
apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: test-stack
spec:
  provider:
    name: aws
    region: us-west-2
---
apiVersion: core.panka.io/v1
kind: Service
metadata:
  name: data
  stack: test-stack
---
apiVersion: components.panka.io/v1
kind: DynamoDB
metadata:
  name: sessions
  service: data
  stack: test-stack
spec:
  billingMode: PAY_PER_REQUEST
  hashKey:
    name: id
    type: S
  rangeKey:
    name: timestamp
    type: N
  ttl:
    enabled: true
    attributeName: expiresAt
  encryption:
    enabled: true
  pointInTimeRecovery: true
`
	
	parser := NewParser()
	result, err := parser.Parse([]byte(yaml))
	
	require.NoError(t, err)
	require.Len(t, result.Components, 1)
	
	dynamo, ok := result.Components[0].(*schema.DynamoDB)
	require.True(t, ok)
	assert.Equal(t, "sessions", dynamo.Metadata.Name)
	assert.Equal(t, "PAY_PER_REQUEST", dynamo.Spec.BillingMode)
	assert.Equal(t, "id", dynamo.Spec.HashKey.Name)
	assert.Equal(t, "S", dynamo.Spec.HashKey.Type)
	assert.NotNil(t, dynamo.Spec.RangeKey)
	assert.Equal(t, "timestamp", dynamo.Spec.RangeKey.Name)
	assert.True(t, dynamo.Spec.TTL.Enabled)
	assert.True(t, dynamo.Spec.PointInTimeRecovery)
}

