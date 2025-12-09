package schema

// Lambda represents an AWS Lambda function
type Lambda struct {
	ResourceBase `yaml:",inline"`
	Spec         LambdaSpec `yaml:"spec" validate:"required"`
}

// LambdaSpec defines the Lambda function specification
type LambdaSpec struct {
	// Runtime configuration
	Runtime string `yaml:"runtime" validate:"required"` // e.g., nodejs18.x, python3.11, go1.x
	Handler string `yaml:"handler" validate:"required"` // e.g., index.handler

	// Code location
	Code LambdaCode `yaml:"code" validate:"required"`

	// Resource allocation
	Memory  string `yaml:"memory,omitempty"`  // Memory in MB (default: 128)
	Timeout string `yaml:"timeout,omitempty"` // Timeout in seconds (default: 3)

	// Environment variables
	Environment map[string]interface{} `yaml:"environment,omitempty"`

	// Triggers (SQS, EventBridge, API Gateway, etc.)
	Triggers []LambdaTrigger `yaml:"triggers,omitempty"`

	// VPC configuration
	VPC LambdaVPC `yaml:"vpc,omitempty"`

	// Concurrency
	ReservedConcurrentExecutions int `yaml:"reservedConcurrentExecutions,omitempty"`

	// IAM
	RoleArn string `yaml:"roleArn,omitempty"` // Custom IAM role (auto-generated if empty)

	// Dependencies
	DependsOn []string `yaml:"dependsOn,omitempty"`

	// Layers
	Layers []string `yaml:"layers,omitempty"`

	// Tags
	Tags map[string]string `yaml:"tags,omitempty"`
}

// LambdaCode defines the Lambda code location
type LambdaCode struct {
	// S3 deployment
	S3Bucket string `yaml:"s3Bucket,omitempty"`
	S3Key    string `yaml:"s3Key,omitempty"`

	// Inline code (for simple functions)
	ZipFile string `yaml:"zipFile,omitempty"`

	// Container image
	ImageUri string `yaml:"imageUri,omitempty"`
}

// LambdaTrigger defines a Lambda trigger
type LambdaTrigger struct {
	Type string `yaml:"type" validate:"required"` // sqs, eventbridge, apigateway, s3, sns, kinesis, dynamodb

	// SQS trigger
	Source    *TriggerSource `yaml:"source,omitempty"`
	BatchSize string         `yaml:"batchSize,omitempty"`

	// EventBridge/CloudWatch Events trigger
	Schedule    string `yaml:"schedule,omitempty"`    // cron or rate expression
	Description string `yaml:"description,omitempty"`

	// API Gateway trigger
	HTTP *HTTPTrigger `yaml:"http,omitempty"`

	// S3 trigger
	S3 *S3Trigger `yaml:"s3,omitempty"`
}

// TriggerSource defines the source component for a trigger
type TriggerSource struct {
	Component string `yaml:"component,omitempty"` // Reference to another component
	Arn       string `yaml:"arn,omitempty"`       // Direct ARN
}

// HTTPTrigger defines API Gateway HTTP trigger
type HTTPTrigger struct {
	Method string `yaml:"method,omitempty"` // GET, POST, etc.
	Path   string `yaml:"path,omitempty"`   // /api/users
}

// S3Trigger defines S3 event trigger
type S3Trigger struct {
	Bucket string   `yaml:"bucket,omitempty"` // Bucket name or component reference
	Events []string `yaml:"events,omitempty"` // s3:ObjectCreated:*, etc.
	Prefix string   `yaml:"prefix,omitempty"`
	Suffix string   `yaml:"suffix,omitempty"`
}

// LambdaVPC defines VPC configuration for Lambda
type LambdaVPC struct {
	Enabled bool `yaml:"enabled"` // If true, uses tenant VPC configuration
	// SubnetIds and SecurityGroupIds are inherited from tenant if enabled
	SubnetIds        []string `yaml:"subnetIds,omitempty"`
	SecurityGroupIds []string `yaml:"securityGroupIds,omitempty"`
}

// Validate validates the Lambda configuration
func (l *Lambda) Validate() error {
	// TODO: Implement validation
	return nil
}

// NewLambda creates a new Lambda with defaults
func NewLambda(name, service, stack string) *Lambda {
	return &Lambda{
		ResourceBase: ResourceBase{
			APIVersion: ComponentsAPIVersion,
			Kind:       KindLambda,
			Metadata: Metadata{
				Name:    name,
				Service: service,
				Stack:   stack,
				Labels:  make(map[string]string),
			},
		},
		Spec: LambdaSpec{
			Runtime: "nodejs18.x",
			Handler: "index.handler",
			Memory:  "128",
			Timeout: "30",
		},
	}
}

