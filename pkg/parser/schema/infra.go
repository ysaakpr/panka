package schema

// ComponentInfra defines infrastructure requirements for a component
type ComponentInfra struct {
	ResourceBase `yaml:",inline"`
	Spec         ComponentInfraSpec `yaml:"spec" validate:"required"`
}

// ComponentInfraSpec defines the component infrastructure specification
type ComponentInfraSpec struct {
	Resources  ResourceRequirements `yaml:"resources" validate:"required"`
	Scaling    ScalingConfig        `yaml:"scaling,omitempty"`
	Networking NetworkingConfig     `yaml:"networking,omitempty"`
	Storage    StorageConfig        `yaml:"storage,omitempty"`
}

// ScalingConfig defines scaling configuration
type ScalingConfig struct {
	Replicas    int          `yaml:"replicas" validate:"required,min=0"`
	AutoScaling *AutoScaling `yaml:"autoscaling,omitempty"`
}

// NetworkingConfig defines networking configuration
type NetworkingConfig struct {
	LoadBalancer *LoadBalancerConfig `yaml:"loadBalancer,omitempty"`
	Ingress      *IngressConfig      `yaml:"ingress,omitempty"`
	ServiceMesh  *ServiceMeshConfig  `yaml:"serviceMesh,omitempty"`
}

// LoadBalancerConfig defines load balancer configuration
type LoadBalancerConfig struct {
	Enabled          bool     `yaml:"enabled"`
	Type             string   `yaml:"type,omitempty" validate:"omitempty,oneof=application network"`
	Internal         bool     `yaml:"internal,omitempty"`
	HealthCheckPath  string   `yaml:"healthCheckPath,omitempty"`
	CertificateARN   string   `yaml:"certificateArn,omitempty"`
	SSLPolicy        string   `yaml:"sslPolicy,omitempty"`
	AllowedCIDRs     []string `yaml:"allowedCIDRs,omitempty"`
}

// IngressConfig defines ingress configuration
type IngressConfig struct {
	Enabled     bool              `yaml:"enabled"`
	Hostname    string            `yaml:"hostname,omitempty"`
	Path        string            `yaml:"path,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// ServiceMeshConfig defines service mesh configuration
type ServiceMeshConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Type     string `yaml:"type,omitempty" validate:"omitempty,oneof=istio linkerd appmesh"`
	MtlsMode string `yaml:"mtlsMode,omitempty" validate:"omitempty,oneof=strict permissive disabled"`
}

// StorageConfig defines storage configuration
type StorageConfig struct {
	Volumes []VolumeMount `yaml:"volumes,omitempty"`
}

// VolumeMount defines a volume mount
type VolumeMount struct {
	Name      string `yaml:"name" validate:"required"`
	MountPath string `yaml:"mountPath" validate:"required"`
	Type      string `yaml:"type" validate:"required,oneof=emptyDir hostPath configMap secret pvc efs"`
	Source    string `yaml:"source,omitempty"`
	ReadOnly  bool   `yaml:"readOnly,omitempty"`
}

// Validate validates the component infrastructure
func (c *ComponentInfra) Validate() error {
	// TODO: Implement validation logic
	return nil
}

// NewComponentInfra creates new component infrastructure with defaults
func NewComponentInfra(name, service, stack string) *ComponentInfra {
	return &ComponentInfra{
		ResourceBase: ResourceBase{
			APIVersion: InfraAPIVersion,
			Kind:       KindComponentInfra,
			Metadata: Metadata{
				Name:    name,
				Service: service,
				Stack:   stack,
				Labels:  make(map[string]string),
			},
		},
		Spec: ComponentInfraSpec{
			Resources: ResourceRequirements{
				CPU:    256,
				Memory: 512,
			},
			Scaling: ScalingConfig{
				Replicas: 1,
			},
		},
	}
}

