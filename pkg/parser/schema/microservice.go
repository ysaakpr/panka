package schema

// MicroService represents a containerized microservice component
type MicroService struct {
	ResourceBase `yaml:",inline"`
	Spec         MicroServiceSpec `yaml:"spec" validate:"required"`
}

// MicroServiceSpec defines the microservice specification
type MicroServiceSpec struct {
	// Container image configuration
	Image ImageConfig `yaml:"image" validate:"required"`
	
	// Runtime configuration
	Runtime RuntimeConfig `yaml:"runtime,omitempty"`
	
	// Networking
	Ports []Port `yaml:"ports,omitempty" validate:"dive"`
	
	// Environment and secrets
	Environment []EnvironmentVariable `yaml:"environment,omitempty" validate:"dive"`
	Secrets     []Secret              `yaml:"secrets,omitempty" validate:"dive"`
	
	// Configuration files
	Configs *ConfigsMount `yaml:"configs,omitempty"`
	
	// Health checks
	HealthCheck *HealthCheck `yaml:"healthCheck,omitempty"`
	
	// Dependencies
	DependsOn []string `yaml:"dependsOn,omitempty"`
	
	// Command override
	Command []string `yaml:"command,omitempty"`
	Args    []string `yaml:"args,omitempty"`
}

// ImageConfig defines container image configuration
type ImageConfig struct {
	Repository string `yaml:"repository" validate:"required"`
	Tag        string `yaml:"tag" validate:"required"`
	PullPolicy string `yaml:"pullPolicy,omitempty" validate:"omitempty,oneof=Always IfNotPresent Never"`
}

// RuntimeConfig defines runtime-specific configuration
type RuntimeConfig struct {
	Platform string `yaml:"platform" validate:"required,oneof=fargate ec2 lambda"`
	
	// For Fargate/ECS
	LaunchType       string `yaml:"launchType,omitempty"`
	NetworkMode      string `yaml:"networkMode,omitempty"`
	RequiresGPU      bool   `yaml:"requiresGPU,omitempty"`
	
	// For Lambda
	Handler string `yaml:"handler,omitempty"`
	Runtime string `yaml:"runtime,omitempty"`
	Timeout int    `yaml:"timeout,omitempty" validate:"omitempty,min=1,max=900"`
}

// ConfigsMount defines configuration file mounting
type ConfigsMount struct {
	MountPath string   `yaml:"mountPath" validate:"required"`
	Files     []string `yaml:"files" validate:"required,min=1"`
}

// Validate validates the microservice
func (m *MicroService) Validate() error {
	// TODO: Implement comprehensive validation
	return nil
}

// NewMicroService creates a new microservice with defaults
func NewMicroService(name, service, stack string) *MicroService {
	return &MicroService{
		ResourceBase: ResourceBase{
			APIVersion: ComponentsAPIVersion,
			Kind:       KindMicroService,
			Metadata: Metadata{
				Name:    name,
				Service: service,
				Stack:   stack,
				Labels:  make(map[string]string),
			},
		},
		Spec: MicroServiceSpec{
			Runtime: RuntimeConfig{
				Platform: "fargate",
			},
		},
	}
}

