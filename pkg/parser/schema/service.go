package schema

// Service represents a logical grouping of related components
type Service struct {
	ResourceBase `yaml:",inline"`
	Spec         ServiceSpec `yaml:"spec" validate:"required"`
}

// ServiceSpec defines the service specification
type ServiceSpec struct {
	Infrastructure InfrastructureConfig `yaml:"infrastructure,omitempty"`
	Variables      map[string]string    `yaml:"variables,omitempty"`
	DependsOn      []string             `yaml:"dependsOn,omitempty"`
}

// Validate validates the service
func (s *Service) Validate() error {
	// TODO: Implement validation logic
	return nil
}

// NewService creates a new service with defaults
func NewService(name, stack string) *Service {
	return &Service{
		ResourceBase: ResourceBase{
			APIVersion: CoreAPIVersion,
			Kind:       KindService,
			Metadata: Metadata{
				Name:   name,
				Stack:  stack,
				Labels: make(map[string]string),
			},
		},
		Spec: ServiceSpec{
			Variables: make(map[string]string),
		},
	}
}

