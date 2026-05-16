package types

import "time"

// Config represents the parsed devbox.yml configuration.
type Config struct {
	Name      string             `yaml:"name" json:"name"`
	Version   string             `yaml:"version" json:"version"`
	Runtimes  map[string]string  `yaml:"runtimes,omitempty" json:"runtimes,omitempty"`
	Services  map[string]Service `yaml:"services" json:"services"`
	Networking Networking        `yaml:"networking,omitempty" json:"networking,omitempty"`
	Security  Security           `yaml:"security,omitempty" json:"security,omitempty"`
	Secrets   Secrets            `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Plugins   []Plugin           `yaml:"plugins,omitempty" json:"plugins,omitempty"`
	Telemetry *TelemetryConfig   `yaml:"telemetry,omitempty" json:"telemetry,omitempty"`
}

// SecretRef represents a reference to a secret
type SecretRef struct {
	Name   string `yaml:"name" json:"name"`
	Source string `yaml:"source" json:"source"`
}

// Service represents a single service definition.
type Service struct {
	Image         string            `yaml:"image,omitempty" json:"image,omitempty"`
	Runtime       string            `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Build         *BuildConfig      `yaml:"build,omitempty" json:"build,omitempty"`
	Command       string            `yaml:"command,omitempty" json:"command,omitempty"`
	Args          []string          `yaml:"args,omitempty" json:"args,omitempty"`
	WorkingDir    string            `yaml:"working_dir,omitempty" json:"working_dir,omitempty"`
	Port          string            `yaml:"port,omitempty" json:"port,omitempty"`
	Ports         []string          `yaml:"ports,omitempty" json:"ports,omitempty"`
	Protocol      string            `yaml:"protocol,omitempty" json:"protocol,omitempty"`
	DependsOn     []string          `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Env           map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	EnvFile       string            `yaml:"env_file,omitempty" json:"env_file,omitempty"`
	Data          string            `yaml:"data,omitempty" json:"data,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Healthcheck   *Healthcheck      `yaml:"healthcheck,omitempty" json:"healthcheck,omitempty"`
	Resources     *Resources        `yaml:"resources,omitempty" json:"resources,omitempty"`
	RestartPolicy *RestartPolicy    `yaml:"restart_policy,omitempty" json:"restart_policy,omitempty"`
	Security      *ServiceSecurity  `yaml:"security,omitempty" json:"security,omitempty"`
	Secrets       []SecretRef       `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

// BuildConfig represents Docker build configuration.
type BuildConfig struct {
	Context    string `yaml:"context" json:"context"`
	Dockerfile string `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`
}

// Healthcheck represents service health check configuration.
type Healthcheck struct {
	Type        string `yaml:"type,omitempty" json:"type,omitempty"`
	Path        string `yaml:"path,omitempty" json:"path,omitempty"`
	Command     string `yaml:"command,omitempty" json:"command,omitempty"`
	Interval    string `yaml:"interval,omitempty" json:"interval,omitempty"`
	Timeout     string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retries     int    `yaml:"retries,omitempty" json:"retries,omitempty"`
	StartPeriod string `yaml:"start_period,omitempty" json:"start_period,omitempty"`
}

// Resources represents service resource limits.
type Resources struct {
	Memory string `yaml:"memory,omitempty" json:"memory,omitempty"`
	CPU    string `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Disk   string `yaml:"disk,omitempty" json:"disk,omitempty"`
}

// RestartPolicy represents service restart configuration.
type RestartPolicy struct {
	OnFailure  bool   `yaml:"on_failure,omitempty" json:"on_failure,omitempty"`
	Always     bool   `yaml:"always,omitempty" json:"always,omitempty"`
	MaxRetries int    `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
	Backoff    string `yaml:"backoff,omitempty" json:"backoff,omitempty"`
}

// ServiceSecurity represents per-service security configuration.
type ServiceSecurity struct {
	TLS          bool     `yaml:"tls,omitempty" json:"tls,omitempty"`
	Capabilities []string `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	ReadOnly     bool     `yaml:"read_only,omitempty" json:"read_only,omitempty"`
}

// Networking represents network configuration.
type Networking struct {
	Discovery bool     `yaml:"discovery,omitempty" json:"discovery,omitempty"`
	Expose    []int    `yaml:"expose,omitempty" json:"expose,omitempty"`
	Egress    string   `yaml:"egress,omitempty" json:"egress,omitempty"`
}

// Security represents global security configuration.
type Security struct {
	TLS          string `yaml:"tls,omitempty" json:"tls,omitempty"`
	Capabilities string `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
}

// Secrets represents secrets configuration.
type Secrets struct {
	Source      string            `yaml:"source,omitempty" json:"source,omitempty"`
	Vault       *VaultConfig      `yaml:"vault,omitempty" json:"vault,omitempty"`
	OnePassword *OnePasswordConfig `yaml:"onepassword,omitempty" json:"onepassword,omitempty"`
	AWS         *AWSConfig        `yaml:"aws,omitempty" json:"aws,omitempty"`
}

// VaultConfig represents HashiCorp Vault configuration.
type VaultConfig struct {
	Address string `yaml:"address" json:"address"`
	Path    string `yaml:"path" json:"path"`
}

// OnePasswordConfig represents 1Password configuration.
type OnePasswordConfig struct {
	Vault string `yaml:"vault" json:"vault"`
}

// AWSConfig represents AWS Secrets Manager configuration.
type AWSConfig struct {
	Region string `yaml:"region" json:"region"`
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
}

// Plugin represents a plugin configuration.
type Plugin struct {
	Name    string                 `yaml:"name" json:"name"`
	Version string                 `yaml:"version,omitempty" json:"version,omitempty"`
	Config  map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// TelemetryConfig represents telemetry configuration.
type TelemetryConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// ServiceStatus represents the runtime status of a service.
type ServiceStatus struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	Health       string    `json:"health"`
	Port         int       `json:"port,omitempty"`
	ContainerID  string    `json:"container_id,omitempty"`
	RestartCount int       `json:"restart_count"`
	StartedAt    time.Time `json:"started_at,omitempty"`
}

// EnvironmentStatus represents the overall environment status.
type EnvironmentStatus struct {
	Name     string          `json:"name"`
	Path     string          `json:"path"`
	Status   string          `json:"status"`
	Services []ServiceStatus `json:"services"`
}
