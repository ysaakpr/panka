package schema

// Stack represents the top-level deployment unit
type Stack struct {
	ResourceBase `yaml:",inline"`
	Spec         StackSpec `yaml:"spec" validate:"required"`
}

// StackSpec defines the stack specification
type StackSpec struct {
	Provider       ProviderConfig       `yaml:"provider" validate:"required"`
	Infrastructure InfrastructureConfig `yaml:"infrastructure,omitempty"`
	Variables      map[string]string    `yaml:"variables,omitempty"`
}

// ProviderConfig defines the cloud provider configuration
type ProviderConfig struct {
	Name   string `yaml:"name" validate:"required,oneof=aws azure gcp"`
	Region string `yaml:"region" validate:"required"`
	
	// AWS-specific
	AccountID string `yaml:"accountId,omitempty"`
	Profile   string `yaml:"profile,omitempty"`
	
	// Additional provider configs can be added
	Config map[string]interface{} `yaml:"config,omitempty"`
}

// InfrastructureConfig references infrastructure configuration files
type InfrastructureConfig struct {
	Defaults      string `yaml:"defaults,omitempty"`
	Networking    string `yaml:"networking,omitempty"`
	Security      string `yaml:"security,omitempty"`
	Observability string `yaml:"observability,omitempty"`
}

// Validate validates the stack
func (s *Stack) Validate() error {
	// TODO: Implement validation logic
	return nil
}

// NewStack creates a new stack with defaults
func NewStack(name string) *Stack {
	return &Stack{
		ResourceBase: ResourceBase{
			APIVersion: CoreAPIVersion,
			Kind:       KindStack,
			Metadata: Metadata{
				Name:   name,
				Labels: make(map[string]string),
			},
		},
		Spec: StackSpec{
			Variables: make(map[string]string),
		},
	}
}

