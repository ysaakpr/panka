package schema

// S3 represents an S3 bucket component
type S3 struct {
	ResourceBase `yaml:",inline"`
	Spec         S3Spec `yaml:"spec" validate:"required"`
}

// S3Spec defines the S3 bucket specification
type S3Spec struct {
	Bucket      BucketConfig      `yaml:"bucket" validate:"required"`
	Versioning  *VersioningConfig `yaml:"versioning,omitempty"`
	Encryption  *S3Encryption     `yaml:"encryption,omitempty"`
	Lifecycle   []LifecycleRule   `yaml:"lifecycle,omitempty"`
	CORS        *CORSConfig       `yaml:"cors,omitempty"`
	Website     *WebsiteConfig    `yaml:"website,omitempty"`
	Replication *ReplicationConfig `yaml:"replication,omitempty"`
	DependsOn   []string          `yaml:"dependsOn,omitempty"`
}

// BucketConfig defines bucket configuration
type BucketConfig struct {
	Name         string   `yaml:"name,omitempty"` // If empty, will be auto-generated
	ACL          string   `yaml:"acl,omitempty" validate:"omitempty,oneof=private public-read public-read-write authenticated-read"`
	ForceDestroy bool     `yaml:"forceDestroy,omitempty"`
	Tags         map[string]string `yaml:"tags,omitempty"`
}

// VersioningConfig defines versioning configuration
type VersioningConfig struct {
	Enabled bool `yaml:"enabled"`
}

// S3Encryption defines S3 encryption configuration
type S3Encryption struct {
	Enabled   bool   `yaml:"enabled"`
	Algorithm string `yaml:"algorithm,omitempty" validate:"omitempty,oneof=AES256 aws:kms"`
	KMSKeyID  string `yaml:"kmsKeyId,omitempty"`
}

// LifecycleRule defines a lifecycle rule
type LifecycleRule struct {
	ID         string                  `yaml:"id" validate:"required"`
	Enabled    bool                    `yaml:"enabled"`
	Prefix     string                  `yaml:"prefix,omitempty"`
	Expiration *ExpirationConfig       `yaml:"expiration,omitempty"`
	Transition []TransitionConfig      `yaml:"transitions,omitempty"`
	NoncurrentVersionExpiration *NoncurrentVersionExpirationConfig `yaml:"noncurrentVersionExpiration,omitempty"`
}

// ExpirationConfig defines expiration configuration
type ExpirationConfig struct {
	Days int `yaml:"days" validate:"required,min=1"`
}

// TransitionConfig defines transition to different storage class
type TransitionConfig struct {
	Days         int    `yaml:"days" validate:"required,min=0"`
	StorageClass string `yaml:"storageClass" validate:"required,oneof=STANDARD_IA ONEZONE_IA INTELLIGENT_TIERING GLACIER DEEP_ARCHIVE"`
}

// NoncurrentVersionExpirationConfig defines expiration for noncurrent versions
type NoncurrentVersionExpirationConfig struct {
	Days int `yaml:"days" validate:"required,min=1"`
}

// CORSConfig defines CORS configuration
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowedOrigins" validate:"required,min=1"`
	AllowedMethods []string `yaml:"allowedMethods" validate:"required,min=1"`
	AllowedHeaders []string `yaml:"allowedHeaders,omitempty"`
	ExposeHeaders  []string `yaml:"exposeHeaders,omitempty"`
	MaxAgeSeconds  int      `yaml:"maxAgeSeconds,omitempty" validate:"omitempty,min=0"`
}

// WebsiteConfig defines static website hosting configuration
type WebsiteConfig struct {
	Enabled           bool   `yaml:"enabled"`
	IndexDocument     string `yaml:"indexDocument" validate:"required_if=Enabled true"`
	ErrorDocument     string `yaml:"errorDocument,omitempty"`
	RoutingRules      string `yaml:"routingRules,omitempty"`
}

// ReplicationConfig defines cross-region replication
type ReplicationConfig struct {
	Enabled        bool   `yaml:"enabled"`
	DestinationBucket string `yaml:"destinationBucket" validate:"required_if=Enabled true"`
	RoleARN        string `yaml:"roleArn" validate:"required_if=Enabled true"`
}

// Validate validates the S3 configuration
func (s *S3) Validate() error {
	// TODO: Implement comprehensive validation
	return nil
}

// NewS3 creates a new S3 resource with defaults
func NewS3(name, service, stack string) *S3 {
	return &S3{
		ResourceBase: ResourceBase{
			APIVersion: ComponentsAPIVersion,
			Kind:       KindS3,
			Metadata: Metadata{
				Name:    name,
				Service: service,
				Stack:   stack,
				Labels:  make(map[string]string),
			},
		},
		Spec: S3Spec{
			Bucket: BucketConfig{
				ACL: "private",
			},
			Versioning: &VersioningConfig{
				Enabled: false,
			},
			Encryption: &S3Encryption{
				Enabled:   true,
				Algorithm: "AES256",
			},
		},
	}
}

