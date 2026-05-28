package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devboxos/devboxos/shared/types"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Validator validates devbox.yml configurations against the JSON schema.
type Validator struct {
	schema *jsonschema.Schema
}

// NewValidator creates a new config validator.
func NewValidator() (*Validator, error) {
	// Load the embedded schema
	schemaPath := filepath.Join("shared", "schemas", "devbox.schema.json")

	// Try loading from the repo root first, then from a relative path
	var schemaData []byte
	var err error

	schemaData, err = os.ReadFile(schemaPath)
	if err != nil {
		// Try relative to the engine binary location
		schemaData, err = os.ReadFile(filepath.Join("..", "..", schemaPath))
		if err != nil {
			// If schema file not found, skip validation (not fatal for development)
			return &Validator{schema: nil}, nil
		}
	}

	var schemaDoc interface{}
	if err := json.Unmarshal(schemaData, &schemaDoc); err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("devbox.schema.json", schemaDoc); err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}

	schema, err := compiler.Compile("devbox.schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}

	return &Validator{schema: schema}, nil
}

// Validate validates a configuration against the schema.
func (v *Validator) Validate(cfg *types.Config) []error {
	var errs []error

	// Validate required fields
	if cfg.Name == "" {
		errs = append(errs, fmt.Errorf("field 'name' is required"))
	}
	if cfg.Version == "" {
		errs = append(errs, fmt.Errorf("field 'version' is required"))
	}
	if len(cfg.Services) == 0 {
		errs = append(errs, fmt.Errorf("at least one service must be defined"))
	}

	// Validate service definitions
	for name, svc := range cfg.Services {
		if svc.Image == "" && svc.Runtime == "" && svc.Build == nil {
			errs = append(errs, fmt.Errorf("service '%s' must define image, runtime, or build", name))
		}
	}

	// Skip JSON schema validation if schema not loaded
	if v.schema == nil {
		return errs
	}

	// Convert config to map for JSON schema validation
	cfgMap := configToMap(cfg)
	if err := v.schema.Validate(cfgMap); err != nil {
		errs = append(errs, fmt.Errorf("schema validation: %w", err))
	}

	return errs
}

// configToMap converts a Config struct to a map for JSON schema validation.
func configToMap(cfg *types.Config) map[string]interface{} {
	result := make(map[string]interface{})
	result["name"] = cfg.Name
	result["version"] = cfg.Version

	if len(cfg.Runtimes) > 0 {
		rtMap := make(map[string]interface{})
		for k, v := range cfg.Runtimes {
			rtMap[k] = v
		}
		result["runtimes"] = rtMap
	}

	services := make(map[string]interface{})
	for name, svc := range cfg.Services {
		svcMap := make(map[string]interface{})
		if svc.Image != "" {
			svcMap["image"] = svc.Image
		}
		if svc.Runtime != "" {
			svcMap["runtime"] = svc.Runtime
		}
		if svc.Build != nil {
			buildMap := make(map[string]interface{})
			buildMap["context"] = svc.Build.Context
			if svc.Build.Dockerfile != "" {
				buildMap["dockerfile"] = svc.Build.Dockerfile
			}
			svcMap["build"] = buildMap
		}
		if svc.Command != "" {
			svcMap["command"] = svc.Command
		}
		if len(svc.Args) > 0 {
			argList := make([]interface{}, len(svc.Args))
			for i, v := range svc.Args {
				argList[i] = v
			}
			svcMap["args"] = argList
		}
		if svc.WorkingDir != "" {
			svcMap["working_dir"] = svc.WorkingDir
		}
		if svc.Port != "" {
			svcMap["port"] = svc.Port
		}
		if len(svc.Ports) > 0 {
			portList := make([]interface{}, len(svc.Ports))
			for i, v := range svc.Ports {
				portList[i] = v
			}
			svcMap["ports"] = portList
		}
		if svc.Protocol != "" {
			svcMap["protocol"] = svc.Protocol
		}
		if len(svc.DependsOn) > 0 {
			depList := make([]interface{}, len(svc.DependsOn))
			for i, v := range svc.DependsOn {
				depList[i] = v
			}
			svcMap["depends_on"] = depList
		}
		if len(svc.Env) > 0 {
			envMap := make(map[string]interface{})
			for k, v := range svc.Env {
				envMap[k] = v
			}
			svcMap["env"] = envMap
		}
		if svc.EnvFile != "" {
			svcMap["env_file"] = svc.EnvFile
		}
		if svc.Data != "" {
			svcMap["data"] = svc.Data
		}
		if len(svc.Volumes) > 0 {
			volList := make([]interface{}, len(svc.Volumes))
			for i, v := range svc.Volumes {
				volList[i] = v
			}
			svcMap["volumes"] = volList
		}
		if svc.Healthcheck != nil {
			hcMap := make(map[string]interface{})
			if svc.Healthcheck.Type != "" {
				hcMap["type"] = svc.Healthcheck.Type
			}
			if svc.Healthcheck.Path != "" {
				hcMap["path"] = svc.Healthcheck.Path
			}
			if svc.Healthcheck.Command != "" {
				hcMap["command"] = svc.Healthcheck.Command
			}
			if svc.Healthcheck.Interval != "" {
				hcMap["interval"] = svc.Healthcheck.Interval
			}
			if svc.Healthcheck.Timeout != "" {
				hcMap["timeout"] = svc.Healthcheck.Timeout
			}
			if svc.Healthcheck.Retries > 0 {
				hcMap["retries"] = svc.Healthcheck.Retries
			}
			if svc.Healthcheck.StartPeriod != "" {
				hcMap["start_period"] = svc.Healthcheck.StartPeriod
			}
			svcMap["healthcheck"] = hcMap
		}
		if svc.Resources != nil {
			resMap := make(map[string]interface{})
			if svc.Resources.Memory != "" {
				resMap["memory"] = svc.Resources.Memory
			}
			if svc.Resources.CPU != "" {
				resMap["cpu"] = svc.Resources.CPU
			}
			if svc.Resources.Disk != "" {
				resMap["disk"] = svc.Resources.Disk
			}
			svcMap["resources"] = resMap
		}
		if svc.RestartPolicy != nil {
			rpMap := make(map[string]interface{})
			if svc.RestartPolicy.OnFailure {
				rpMap["on_failure"] = true
			}
			if svc.RestartPolicy.Always {
				rpMap["always"] = true
			}
			if svc.RestartPolicy.MaxRetries > 0 {
				rpMap["max_retries"] = svc.RestartPolicy.MaxRetries
			}
			if svc.RestartPolicy.Backoff != "" {
				rpMap["backoff"] = svc.RestartPolicy.Backoff
			}
			svcMap["restart_policy"] = rpMap
		}
		if svc.Security != nil {
			secMap := make(map[string]interface{})
			secMap["tls"] = svc.Security.TLS
			if len(svc.Security.Capabilities) > 0 {
				capList := make([]interface{}, len(svc.Security.Capabilities))
				for i, v := range svc.Security.Capabilities {
					capList[i] = v
				}
				secMap["capabilities"] = capList
			}
			secMap["read_only"] = svc.Security.ReadOnly
			svcMap["security"] = secMap
		}
		services[name] = svcMap
	}
	result["services"] = services

	// Networking
	if cfg.Networking.Discovery || len(cfg.Networking.Expose) > 0 || cfg.Networking.Egress != "" {
		netMap := make(map[string]interface{})
		netMap["discovery"] = cfg.Networking.Discovery
		if len(cfg.Networking.Expose) > 0 {
			exposeList := make([]interface{}, len(cfg.Networking.Expose))
			for i, v := range cfg.Networking.Expose {
				exposeList[i] = v
			}
			netMap["expose"] = exposeList
		}
		if cfg.Networking.Egress != "" {
			netMap["egress"] = cfg.Networking.Egress
		}
		result["networking"] = netMap
	}

	// Security
	if cfg.Security.TLS != "" || cfg.Security.Capabilities != "" {
		secMap := make(map[string]interface{})
		if cfg.Security.TLS != "" {
			secMap["tls"] = cfg.Security.TLS
		}
		if cfg.Security.Capabilities != "" {
			secMap["capabilities"] = cfg.Security.Capabilities
		}
		result["security"] = secMap
	}

	// Secrets
	if cfg.Secrets.Source != "" {
		secMap := make(map[string]interface{})
		secMap["source"] = cfg.Secrets.Source
		result["secrets"] = secMap
	}

	return result
}
