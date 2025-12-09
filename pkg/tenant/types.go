package tenant

import (
	"time"
)

// Tenant represents a tenant in the multi-tenant system
type Tenant struct {
	// Identity
	ID          string    `yaml:"id" json:"id"`
	DisplayName string    `yaml:"displayName" json:"displayName"`
	Email       string    `yaml:"email" json:"email"`
	Status      Status    `yaml:"status" json:"status"`
	Created     time.Time `yaml:"created" json:"created"`
	Updated     time.Time `yaml:"updated" json:"updated"`

	// Credentials
	Credentials Credentials `yaml:"credentials" json:"credentials"`

	// Storage configuration
	Storage StorageConfig `yaml:"storage" json:"storage"`

	// Lock configuration
	Locks LockConfig `yaml:"locks" json:"locks"`

	// AWS configuration
	AWS AWSConfig `yaml:"aws,omitempty" json:"aws,omitempty"`

	// Networking configuration - shared by all stacks in tenant
	Networking NetworkingConfig `yaml:"networking,omitempty" json:"networking,omitempty"`

	// Limits and quotas
	Limits Limits `yaml:"limits" json:"limits"`

	// Default tags applied to all resources
	DefaultTags map[string]string `yaml:"defaultTags,omitempty" json:"defaultTags,omitempty"`

	// Allowed resource types
	AllowedResources []string `yaml:"allowedResources,omitempty" json:"allowedResources,omitempty"`

	// Metadata
	Metadata map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// =============================================================================
// NETWORKING CONFIGURATION
// =============================================================================

// NetworkingConfig defines tenant-level networking (shared by all stacks)
type NetworkingConfig struct {
	// VPC configuration
	VPC VPCConfig `yaml:"vpc" json:"vpc"`

	// Subnet configuration
	Subnets SubnetsConfig `yaml:"subnets" json:"subnets"`

	// NAT Gateway configuration
	NATGateway NATGatewayConfig `yaml:"natGateway,omitempty" json:"natGateway,omitempty"`

	// Internet Gateway configuration
	InternetGateway InternetGatewayConfig `yaml:"internetGateway,omitempty" json:"internetGateway,omitempty"`

	// Default Security Group configuration
	DefaultSecurityGroup SecurityGroupConfig `yaml:"defaultSecurityGroup,omitempty" json:"defaultSecurityGroup,omitempty"`

	// Resource IDs (populated after creation)
	ResourceIDs *NetworkingResourceIDs `yaml:"resourceIds,omitempty" json:"resourceIds,omitempty"`
}

// VPCConfig defines VPC settings
type VPCConfig struct {
	CidrBlock          string `yaml:"cidrBlock" json:"cidrBlock"`                   // e.g., "10.0.0.0/16"
	EnableDNSHostnames bool   `yaml:"enableDNSHostnames" json:"enableDNSHostnames"` // Default: true
	EnableDNSSupport   bool   `yaml:"enableDNSSupport" json:"enableDNSSupport"`     // Default: true
	Name               string `yaml:"name,omitempty" json:"name,omitempty"`         // Auto-generated if empty
}

// SubnetsConfig defines subnet configuration
type SubnetsConfig struct {
	Public  []SubnetConfig `yaml:"public,omitempty" json:"public,omitempty"`
	Private []SubnetConfig `yaml:"private,omitempty" json:"private,omitempty"`
}

// SubnetConfig defines a single subnet
type SubnetConfig struct {
	CidrBlock        string `yaml:"cidrBlock" json:"cidrBlock"`               // e.g., "10.0.1.0/24"
	AvailabilityZone string `yaml:"availabilityZone" json:"availabilityZone"` // e.g., "us-east-1a"
	Name             string `yaml:"name,omitempty" json:"name,omitempty"`     // Auto-generated if empty
	// ResourceID is populated after creation
	ResourceID string `yaml:"resourceId,omitempty" json:"resourceId,omitempty"`
}

// NATGatewayConfig defines NAT Gateway settings
type NATGatewayConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Type    string `yaml:"type,omitempty" json:"type,omitempty"` // "single" or "per-az", default: "single"
}

// InternetGatewayConfig defines Internet Gateway settings
type InternetGatewayConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// SecurityGroupConfig defines security group settings
type SecurityGroupConfig struct {
	Name                 string         `yaml:"name,omitempty" json:"name,omitempty"`
	Description          string         `yaml:"description,omitempty" json:"description,omitempty"`
	AllowInternalTraffic bool           `yaml:"allowInternalTraffic" json:"allowInternalTraffic"` // Services can talk to each other
	Ingress              []SecurityRule `yaml:"ingress,omitempty" json:"ingress,omitempty"`
	Egress               []SecurityRule `yaml:"egress,omitempty" json:"egress,omitempty"`
}

// SecurityRule defines a security group rule
type SecurityRule struct {
	Protocol    string   `yaml:"protocol" json:"protocol"`                       // "tcp", "udp", "icmp", "-1" (all)
	Port        int      `yaml:"port,omitempty" json:"port,omitempty"`           // Single port
	FromPort    int      `yaml:"fromPort,omitempty" json:"fromPort,omitempty"`   // Port range start
	ToPort      int      `yaml:"toPort,omitempty" json:"toPort,omitempty"`       // Port range end
	CidrBlocks  []string `yaml:"cidrBlocks,omitempty" json:"cidrBlocks,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
}

// NetworkingResourceIDs stores AWS resource IDs after creation
type NetworkingResourceIDs struct {
	VPCID              string   `yaml:"vpcId,omitempty" json:"vpcId,omitempty"`
	InternetGatewayID  string   `yaml:"internetGatewayId,omitempty" json:"internetGatewayId,omitempty"`
	NATGatewayIDs      []string `yaml:"natGatewayIds,omitempty" json:"natGatewayIds,omitempty"`
	PublicSubnetIDs    []string `yaml:"publicSubnetIds,omitempty" json:"publicSubnetIds,omitempty"`
	PrivateSubnetIDs   []string `yaml:"privateSubnetIds,omitempty" json:"privateSubnetIds,omitempty"`
	SecurityGroupID    string   `yaml:"securityGroupId,omitempty" json:"securityGroupId,omitempty"`
	PublicRouteTableID string   `yaml:"publicRouteTableId,omitempty" json:"publicRouteTableId,omitempty"`
	PrivateRouteTableIDs []string `yaml:"privateRouteTableIds,omitempty" json:"privateRouteTableIds,omitempty"`
}

// Status represents the tenant status
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDeleted   Status = "deleted"
)

// Credentials stores the tenant's authentication credentials
type Credentials struct {
	Hash         string    `yaml:"hash" json:"hash"`
	Algorithm    string    `yaml:"algorithm" json:"algorithm"`
	Rotations    int       `yaml:"rotations" json:"rotations"`
	LastRotated  *time.Time `yaml:"lastRotated,omitempty" json:"lastRotated,omitempty"`
}

// StorageConfig defines where tenant state is stored
type StorageConfig struct {
	Prefix  string `yaml:"prefix" json:"prefix"`
	Version string `yaml:"version" json:"version"`
	Path    string `yaml:"path" json:"path"`
}

// LockConfig defines how tenant locks are namespaced
type LockConfig struct {
	Prefix string `yaml:"prefix" json:"prefix"`
}

// AWSConfig stores AWS-specific configuration
type AWSConfig struct {
	AccountID     string `yaml:"accountId,omitempty" json:"accountId,omitempty"`
	Region        string `yaml:"region,omitempty" json:"region,omitempty"`
	AssumeRoleArn string `yaml:"assumeRoleArn,omitempty" json:"assumeRoleArn,omitempty"`
}

// Limits defines resource limits for the tenant
type Limits struct {
	CostTracking          bool `yaml:"costTracking" json:"costTracking"`
	MonthlyCostLimit      int  `yaml:"monthlyCostLimit" json:"monthlyCostLimit"` // USD
	MaxStacks             int  `yaml:"maxStacks" json:"maxStacks"`
	MaxServicesPerStack   int  `yaml:"maxServicesPerStack" json:"maxServicesPerStack"`
	MaxResourcesPerService int `yaml:"maxResourcesPerService" json:"maxResourcesPerService"`
}

// Registry represents the tenants.yaml file structure
type Registry struct {
	Version  string           `yaml:"version" json:"version"`
	Metadata RegistryMetadata `yaml:"metadata" json:"metadata"`
	Config   RegistryConfig   `yaml:"config" json:"config"`
	Tenants  []Tenant         `yaml:"tenants" json:"tenants"`
}

// RegistryMetadata contains registry-level metadata
type RegistryMetadata struct {
	Created time.Time `yaml:"created" json:"created"`
	Updated time.Time `yaml:"updated" json:"updated"`
	Bucket  string    `yaml:"bucket" json:"bucket"`
	Region  string    `yaml:"region" json:"region"`
}

// RegistryConfig contains registry-level configuration
type RegistryConfig struct {
	LockTable      string `yaml:"lockTable" json:"lockTable"`
	DefaultVersion string `yaml:"defaultVersion" json:"defaultVersion"`
}

// Session represents an authenticated session
type Session struct {
	Mode          SessionMode `yaml:"mode" json:"mode"`
	Tenant        *TenantInfo `yaml:"tenant,omitempty" json:"tenant,omitempty"`
	Backend       *BackendConfig `yaml:"backend,omitempty" json:"backend,omitempty"`
	Locks         *LocksConfig `yaml:"locks,omitempty" json:"locks,omitempty"`
	AWS           *AWSConfig `yaml:"aws,omitempty" json:"aws,omitempty"`
	Authenticated time.Time `yaml:"authenticated" json:"authenticated"`
	Expires       time.Time `yaml:"expires" json:"expires"`
}

// SessionMode defines the type of session
type SessionMode string

const (
	ModeAdmin  SessionMode = "admin"
	ModeTenant SessionMode = "tenant"
)

// TenantInfo contains basic tenant information for sessions
type TenantInfo struct {
	ID          string `yaml:"id" json:"id"`
	DisplayName string `yaml:"displayName" json:"displayName"`
	Version     string `yaml:"version" json:"version"`
}

// BackendConfig contains S3 backend configuration
type BackendConfig struct {
	Type   string `yaml:"type" json:"type"`
	Bucket string `yaml:"bucket" json:"bucket"`
	Region string `yaml:"region" json:"region"`
	Prefix string `yaml:"prefix" json:"prefix"`
}

// LocksConfig contains DynamoDB lock configuration
type LocksConfig struct {
	Type   string `yaml:"type" json:"type"`
	Table  string `yaml:"table" json:"table"`
	Region string `yaml:"region" json:"region"`
	Prefix string `yaml:"prefix" json:"prefix"`
}

// CreateTenantRequest represents a request to create a new tenant
type CreateTenantRequest struct {
	Name             string
	DisplayName      string
	Email            string
	AWSAccountID     string
	AWSRegion        string
	AWSAssumeRoleArn string
	Version          string

	// Networking
	VPCCidr              string
	EnableNATGateway     bool
	NATGatewayType       string // "single" or "per-az"
	AvailabilityZones    []string

	// Limits
	CostTracking           bool
	MonthlyCostLimit       int
	MaxStacks              int
	MaxServicesPerStack    int
	MaxResourcesPerService int

	// Tags and metadata
	DefaultTags      map[string]string
	AllowedResources []string
	Metadata         map[string]string
}

// DefaultNetworkingConfig returns default networking configuration
func DefaultNetworkingConfig(region string, vpcCidr string, azs []string) NetworkingConfig {
	if vpcCidr == "" {
		vpcCidr = "10.0.0.0/16"
	}
	if len(azs) == 0 {
		azs = []string{region + "a", region + "b"}
	}

	// Generate subnet CIDRs based on VPC CIDR
	// For 10.0.0.0/16:
	//   Public:  10.0.1.0/24, 10.0.2.0/24
	//   Private: 10.0.10.0/24, 10.0.20.0/24
	publicSubnets := make([]SubnetConfig, len(azs))
	privateSubnets := make([]SubnetConfig, len(azs))

	for i, az := range azs {
		publicSubnets[i] = SubnetConfig{
			CidrBlock:        generateSubnetCidr(vpcCidr, i+1),
			AvailabilityZone: az,
		}
		privateSubnets[i] = SubnetConfig{
			CidrBlock:        generateSubnetCidr(vpcCidr, (i+1)*10),
			AvailabilityZone: az,
		}
	}

	return NetworkingConfig{
		VPC: VPCConfig{
			CidrBlock:          vpcCidr,
			EnableDNSHostnames: true,
			EnableDNSSupport:   true,
		},
		Subnets: SubnetsConfig{
			Public:  publicSubnets,
			Private: privateSubnets,
		},
		NATGateway: NATGatewayConfig{
			Enabled: true,
			Type:    "single",
		},
		InternetGateway: InternetGatewayConfig{
			Enabled: true,
		},
		DefaultSecurityGroup: SecurityGroupConfig{
			AllowInternalTraffic: true,
			Egress: []SecurityRule{
				{
					Protocol:    "-1",
					CidrBlocks:  []string{"0.0.0.0/0"},
					Description: "Allow all outbound traffic",
				},
			},
		},
	}
}

// generateSubnetCidr generates a /24 subnet CIDR from a /16 VPC CIDR
func generateSubnetCidr(vpcCidr string, thirdOctet int) string {
	// Simple implementation: replace third octet
	// "10.0.0.0/16" + 1 -> "10.0.1.0/24"
	// This is a simplified version; production code should properly parse CIDR
	parts := []byte(vpcCidr)
	// Find second dot
	dotCount := 0
	secondDotIdx := 0
	for i, c := range parts {
		if c == '.' {
			dotCount++
			if dotCount == 2 {
				secondDotIdx = i
				break
			}
		}
	}
	if secondDotIdx > 0 {
		// Extract base (e.g., "10.0")
		base := string(parts[:secondDotIdx])
		return base + "." + itoa(thirdOctet) + ".0/24"
	}
	return vpcCidr // fallback
}

// itoa converts int to string (simple implementation)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

// TenantCredentials represents generated tenant credentials
type TenantCredentials struct {
	TenantID string
	Secret   string // Plain text, shown once
	Hash     string // Bcrypt hash, stored
}

