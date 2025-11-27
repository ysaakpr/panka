package schema

// SQS represents an SQS queue component
type SQS struct {
	ResourceBase `yaml:",inline"`
	Spec         SQSSpec `yaml:"spec" validate:"required"`
}

// SQSSpec defines the SQS specification
type SQSSpec struct {
	Type                   string           `yaml:"type" validate:"required,oneof=standard fifo"`
	MessageRetentionPeriod int              `yaml:"messageRetentionPeriod,omitempty" validate:"omitempty,min=60,max=1209600"` // 1 min to 14 days
	VisibilityTimeout      int              `yaml:"visibilityTimeout,omitempty" validate:"omitempty,min=0,max=43200"` // 0 to 12 hours
	MaxMessageSize         int              `yaml:"maxMessageSize,omitempty" validate:"omitempty,min=1024,max=262144"` // 1KB to 256KB
	ReceiveWaitTime        int              `yaml:"receiveWaitTime,omitempty" validate:"omitempty,min=0,max=20"` // Long polling
	DelaySeconds           int              `yaml:"delaySeconds,omitempty" validate:"omitempty,min=0,max=900"` // 0 to 15 min
	
	// Dead Letter Queue
	DeadLetterQueue *DeadLetterQueueConfig `yaml:"deadLetterQueue,omitempty"`
	
	// FIFO-specific
	ContentBasedDeduplication bool `yaml:"contentBasedDeduplication,omitempty"`
	DeduplicationScope        string `yaml:"deduplicationScope,omitempty" validate:"omitempty,oneof=messageGroup queue"`
	FifoThroughputLimit       string `yaml:"fifoThroughputLimit,omitempty" validate:"omitempty,oneof=perQueue perMessageGroupId"`
	
	DependsOn []string `yaml:"dependsOn,omitempty"`
}

// DeadLetterQueueConfig defines DLQ configuration
type DeadLetterQueueConfig struct {
	Enabled         bool `yaml:"enabled"`
	MaxReceiveCount int  `yaml:"maxReceiveCount" validate:"required_if=Enabled true,omitempty,min=1"`
}

// Validate validates the SQS configuration
func (s *SQS) Validate() error {
	// TODO: Implement validation logic
	return nil
}

// NewSQS creates a new SQS resource with defaults
func NewSQS(name, service, stack string) *SQS {
	return &SQS{
		ResourceBase: ResourceBase{
			APIVersion: ComponentsAPIVersion,
			Kind:       KindSQS,
			Metadata: Metadata{
				Name:    name,
				Service: service,
				Stack:   stack,
				Labels:  make(map[string]string),
			},
		},
		Spec: SQSSpec{
			Type:                   "standard",
			MessageRetentionPeriod: 345600, // 4 days
			VisibilityTimeout:      30,
			MaxMessageSize:         262144, // 256KB
		},
	}
}

// SNS represents an SNS topic component
type SNS struct {
	ResourceBase `yaml:",inline"`
	Spec         SNSSpec `yaml:"spec" validate:"required"`
}

// SNSSpec defines the SNS specification
type SNSSpec struct {
	DisplayName       string            `yaml:"displayName,omitempty"`
	DeliveryPolicy    string            `yaml:"deliveryPolicy,omitempty"`
	FifoTopic         bool              `yaml:"fifoTopic,omitempty"`
	ContentBasedDeduplication bool      `yaml:"contentBasedDeduplication,omitempty"`
	
	// Subscriptions
	Subscriptions []SNSSubscription `yaml:"subscriptions,omitempty"`
	
	DependsOn []string `yaml:"dependsOn,omitempty"`
}

// SNSSubscription defines an SNS subscription
type SNSSubscription struct {
	Protocol string `yaml:"protocol" validate:"required,oneof=http https email email-json sms sqs lambda application"`
	Endpoint string `yaml:"endpoint" validate:"required"`
	FilterPolicy string `yaml:"filterPolicy,omitempty"`
}

// Validate validates the SNS configuration
func (s *SNS) Validate() error {
	// TODO: Implement validation logic
	return nil
}

// NewSNS creates a new SNS resource with defaults
func NewSNS(name, service, stack string) *SNS {
	return &SNS{
		ResourceBase: ResourceBase{
			APIVersion: ComponentsAPIVersion,
			Kind:       KindSNS,
			Metadata: Metadata{
				Name:    name,
				Service: service,
				Stack:   stack,
				Labels:  make(map[string]string),
			},
		},
		Spec: SNSSpec{},
	}
}

