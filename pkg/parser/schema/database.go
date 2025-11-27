package schema

// RDS represents an RDS database component
type RDS struct {
	ResourceBase `yaml:",inline"`
	Spec         RDSSpec `yaml:"spec" validate:"required"`
}

// RDSSpec defines the RDS specification
type RDSSpec struct {
	Engine   EngineConfig   `yaml:"engine" validate:"required"`
	Instance InstanceConfig `yaml:"instance" validate:"required"`
	Database DatabaseConfig `yaml:"database" validate:"required"`
	Backup   BackupConfig   `yaml:"backup,omitempty"`
	DependsOn []string      `yaml:"dependsOn,omitempty"`
}

// EngineConfig defines the database engine configuration
type EngineConfig struct {
	Type    string `yaml:"type" validate:"required,oneof=postgres mysql mariadb aurora-postgresql aurora-mysql"`
	Version string `yaml:"version" validate:"required"`
	
	// Engine-specific parameters
	ParameterGroup string            `yaml:"parameterGroup,omitempty"`
	Parameters     map[string]string `yaml:"parameters,omitempty"`
}

// InstanceConfig defines the RDS instance configuration
type InstanceConfig struct {
	Class   string        `yaml:"class" validate:"required"`
	Storage StorageSpec   `yaml:"storage" validate:"required"`
	
	// High Availability
	MultiAZ           bool   `yaml:"multiAZ,omitempty"`
	AvailabilityZones []string `yaml:"availabilityZones,omitempty"`
	
	// Performance
	IOPS int `yaml:"iops,omitempty" validate:"omitempty,min=1000"`
}

// StorageSpec defines storage configuration for RDS
type StorageSpec struct {
	Type        string `yaml:"type" validate:"required,oneof=gp2 gp3 io1 io2"`
	AllocatedGB int    `yaml:"allocatedGB" validate:"required,min=20"`
	MaxAllocatedGB int `yaml:"maxAllocatedGB,omitempty" validate:"omitempty,min=20"`
}

// DatabaseConfig defines database-specific configuration
type DatabaseConfig struct {
	Name           string    `yaml:"name" validate:"required"`
	Username       string    `yaml:"username" validate:"required"`
	PasswordSecret SecretRef `yaml:"passwordSecret" validate:"required"`
	Port           int       `yaml:"port,omitempty" validate:"omitempty,min=1,max=65535"`
}

// BackupConfig defines backup configuration
type BackupConfig struct {
	Enabled            bool   `yaml:"enabled"`
	RetentionDays      int    `yaml:"retentionDays,omitempty" validate:"omitempty,min=1,max=35"`
	PreferredWindow    string `yaml:"preferredWindow,omitempty"`
	MaintenanceWindow  string `yaml:"maintenanceWindow,omitempty"`
}

// Validate validates the RDS configuration
func (r *RDS) Validate() error {
	// TODO: Implement comprehensive validation
	return nil
}

// NewRDS creates a new RDS resource with defaults
func NewRDS(name, service, stack string) *RDS {
	return &RDS{
		ResourceBase: ResourceBase{
			APIVersion: ComponentsAPIVersion,
			Kind:       KindRDS,
			Metadata: Metadata{
				Name:    name,
				Service: service,
				Stack:   stack,
				Labels:  make(map[string]string),
			},
		},
		Spec: RDSSpec{
			Backup: BackupConfig{
				Enabled:       true,
				RetentionDays: 7,
			},
		},
	}
}

// DynamoDB represents a DynamoDB table component
type DynamoDB struct {
	ResourceBase `yaml:",inline"`
	Spec         DynamoDBSpec `yaml:"spec" validate:"required"`
}

// DynamoDBSpec defines the DynamoDB specification
type DynamoDBSpec struct {
	BillingMode string              `yaml:"billingMode" validate:"required,oneof=PAY_PER_REQUEST PROVISIONED"`
	TableName   string              `yaml:"tableName,omitempty"`
	HashKey     AttributeDefinition `yaml:"hashKey" validate:"required"`
	RangeKey    *AttributeDefinition `yaml:"rangeKey,omitempty"`
	
	// Provisioned throughput (for PROVISIONED billing mode)
	ReadCapacity  int `yaml:"readCapacity,omitempty" validate:"omitempty,min=1"`
	WriteCapacity int `yaml:"writeCapacity,omitempty" validate:"omitempty,min=1"`
	
	// Global Secondary Indexes
	GlobalSecondaryIndexes []GlobalSecondaryIndex `yaml:"globalSecondaryIndexes,omitempty"`
	
	// TTL configuration
	TTL *TTLConfig `yaml:"ttl,omitempty"`
	
	// Encryption
	Encryption *EncryptionConfig `yaml:"encryption,omitempty"`
	
	// Point-in-time recovery
	PointInTimeRecovery bool `yaml:"pointInTimeRecovery,omitempty"`
	
	DependsOn []string `yaml:"dependsOn,omitempty"`
}

// AttributeDefinition defines a DynamoDB attribute
type AttributeDefinition struct {
	Name string `yaml:"name" validate:"required"`
	Type string `yaml:"type" validate:"required,oneof=S N B"` // String, Number, Binary
}

// GlobalSecondaryIndex defines a GSI
type GlobalSecondaryIndex struct {
	Name          string              `yaml:"name" validate:"required"`
	HashKey       AttributeDefinition `yaml:"hashKey" validate:"required"`
	RangeKey      *AttributeDefinition `yaml:"rangeKey,omitempty"`
	Projection    string              `yaml:"projection" validate:"required,oneof=ALL KEYS_ONLY INCLUDE"`
	ReadCapacity  int                 `yaml:"readCapacity,omitempty" validate:"omitempty,min=1"`
	WriteCapacity int                 `yaml:"writeCapacity,omitempty" validate:"omitempty,min=1"`
}

// TTLConfig defines TTL configuration
type TTLConfig struct {
	Enabled       bool   `yaml:"enabled"`
	AttributeName string `yaml:"attributeName" validate:"required"`
}

// EncryptionConfig defines encryption configuration
type EncryptionConfig struct {
	Enabled bool   `yaml:"enabled"`
	KMSKey  string `yaml:"kmsKey,omitempty"`
}

// Validate validates the DynamoDB configuration
func (d *DynamoDB) Validate() error {
	// TODO: Implement comprehensive validation
	return nil
}

// NewDynamoDB creates a new DynamoDB resource with defaults
func NewDynamoDB(name, service, stack string) *DynamoDB {
	return &DynamoDB{
		ResourceBase: ResourceBase{
			APIVersion: ComponentsAPIVersion,
			Kind:       KindDynamoDB,
			Metadata: Metadata{
				Name:    name,
				Service: service,
				Stack:   stack,
				Labels:  make(map[string]string),
			},
		},
		Spec: DynamoDBSpec{
			BillingMode: "PAY_PER_REQUEST",
			Encryption: &EncryptionConfig{
				Enabled: true,
			},
		},
	}
}

