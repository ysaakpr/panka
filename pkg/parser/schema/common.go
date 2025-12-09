package schema

// APIVersion represents the API version of a resource
type APIVersion string

const (
	// CoreAPIVersion is the API version for core resources (Stack, Service)
	CoreAPIVersion APIVersion = "core.panka.io/v1"
	
	// InfraAPIVersion is the API version for infrastructure resources
	InfraAPIVersion APIVersion = "infra.panka.io/v1"
	
	// ComponentsAPIVersion is the API version for component resources
	ComponentsAPIVersion APIVersion = "components.panka.io/v1"
)

// Kind represents the kind of resource
type Kind string

const (
	// Core resource kinds
	KindStack   Kind = "Stack"
	KindService Kind = "Service"
	
	// Infrastructure kinds
	KindComponentInfra Kind = "ComponentInfra"
	KindInfraDefaults  Kind = "InfraDefaults"
	KindNetworking     Kind = "Networking"
	KindSecurity       Kind = "Security"
	KindObservability  Kind = "Observability"
	
	// Component kinds - Compute
	KindMicroService Kind = "MicroService"
	KindWorker       Kind = "Worker"
	KindCronJob      Kind = "CronJob"
	KindLambda       Kind = "Lambda"
	
	// Component kinds - Database
	KindRDS         Kind = "RDS"
	KindDynamoDB    Kind = "DynamoDB"
	KindDocumentDB  Kind = "DocumentDB"
	
	// Component kinds - Cache
	KindElastiCacheRedis      Kind = "ElastiCacheRedis"
	KindElastiCacheMemcached  Kind = "ElastiCacheMemcached"
	KindMemoryDB              Kind = "MemoryDB"
	
	// Component kinds - Storage
	KindS3  Kind = "S3"
	KindEFS Kind = "EFS"
	KindEBS Kind = "EBS"
	
	// Component kinds - Messaging
	KindSQS         Kind = "SQS"
	KindSNS         Kind = "SNS"
	KindKafka       Kind = "Kafka"
	KindMSK         Kind = "MSK"
	KindEventBridge Kind = "EventBridge"
	
	// Component kinds - Networking
	KindALB        Kind = "ALB"
	KindNLB        Kind = "NLB"
	KindCloudFront Kind = "CloudFront"
	KindAPIGateway Kind = "APIGateway"
)

// Metadata contains common metadata for all resources
type Metadata struct {
	Name        string            `yaml:"name" validate:"required"`
	Description string            `yaml:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`

	// Hierarchical references
	Tenant  string `yaml:"tenant,omitempty"`
	Stack   string `yaml:"stack,omitempty"`
	Service string `yaml:"service,omitempty"`
}

// Resource is the base interface that all resources implement
type Resource interface {
	GetAPIVersion() APIVersion
	GetKind() Kind
	GetMetadata() *Metadata
	Validate() error
}

// ResourceBase provides common fields for all resources
type ResourceBase struct {
	APIVersion APIVersion `yaml:"apiVersion" validate:"required"`
	Kind       Kind       `yaml:"kind" validate:"required"`
	Metadata   Metadata   `yaml:"metadata" validate:"required"`
}

// GetAPIVersion returns the API version
func (r *ResourceBase) GetAPIVersion() APIVersion {
	return r.APIVersion
}

// GetKind returns the resource kind
func (r *ResourceBase) GetKind() Kind {
	return r.Kind
}

// GetMetadata returns the resource metadata
func (r *ResourceBase) GetMetadata() *Metadata {
	return &r.Metadata
}

// ValueFrom represents a reference to another component's output
type ValueFrom struct {
	Component string `yaml:"component" validate:"required"`
	Output    string `yaml:"output" validate:"required"`
}

// SecretRef represents a reference to a secret
type SecretRef struct {
	Ref    string `yaml:"ref" validate:"required"`
	EnvVar string `yaml:"envVar,omitempty"`
}

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	Name      string     `yaml:"name" validate:"required"`
	Value     string     `yaml:"value,omitempty"`
	ValueFrom *ValueFrom `yaml:"valueFrom,omitempty"`
}

// Secret represents a secret to be injected
type Secret struct {
	Name      string    `yaml:"name" validate:"required"`
	SecretRef string    `yaml:"secretRef" validate:"required"`
	EnvVar    string    `yaml:"envVar,omitempty"`
}

// ResourceRequirements defines compute resources
type ResourceRequirements struct {
	CPU    int `yaml:"cpu" validate:"required,min=128"`       // CPU units (256 = 0.25 vCPU)
	Memory int `yaml:"memory" validate:"required,min=128"`    // Memory in MB
}

// AutoScaling defines autoscaling configuration
type AutoScaling struct {
	Enabled     bool `yaml:"enabled"`
	MinReplicas int  `yaml:"minReplicas" validate:"required,min=1"`
	MaxReplicas int  `yaml:"maxReplicas" validate:"required,min=1"`
	
	// Scaling policies
	TargetCPUPercent    int `yaml:"targetCPUPercent,omitempty" validate:"omitempty,min=1,max=100"`
	TargetMemoryPercent int `yaml:"targetMemoryPercent,omitempty" validate:"omitempty,min=1,max=100"`
}

// HealthCheck defines health check configuration
type HealthCheck struct {
	Readiness *HealthCheckProbe `yaml:"readiness,omitempty"`
	Liveness  *HealthCheckProbe `yaml:"liveness,omitempty"`
}

// HealthCheckProbe defines a specific health check probe
type HealthCheckProbe struct {
	HTTP *HTTPHealthCheck `yaml:"http,omitempty"`
	TCP  *TCPHealthCheck  `yaml:"tcp,omitempty"`
	Exec *ExecHealthCheck `yaml:"exec,omitempty"`
	
	InitialDelaySeconds int `yaml:"initialDelaySeconds,omitempty" validate:"omitempty,min=0"`
	PeriodSeconds       int `yaml:"periodSeconds,omitempty" validate:"omitempty,min=1"`
	TimeoutSeconds      int `yaml:"timeoutSeconds,omitempty" validate:"omitempty,min=1"`
	SuccessThreshold    int `yaml:"successThreshold,omitempty" validate:"omitempty,min=1"`
	FailureThreshold    int `yaml:"failureThreshold,omitempty" validate:"omitempty,min=1"`
}

// HTTPHealthCheck defines an HTTP health check
type HTTPHealthCheck struct {
	Path   string `yaml:"path" validate:"required"`
	Port   int    `yaml:"port" validate:"required,min=1,max=65535"`
	Scheme string `yaml:"scheme,omitempty"` // http or https
}

// TCPHealthCheck defines a TCP health check
type TCPHealthCheck struct {
	Port int `yaml:"port" validate:"required,min=1,max=65535"`
}

// ExecHealthCheck defines a command execution health check
type ExecHealthCheck struct {
	Command []string `yaml:"command" validate:"required,min=1"`
}

// Port represents a container port
type Port struct {
	Name     string `yaml:"name" validate:"required"`
	Port     int    `yaml:"port" validate:"required,min=1,max=65535"`
	Protocol string `yaml:"protocol,omitempty"` // tcp, udp
}

