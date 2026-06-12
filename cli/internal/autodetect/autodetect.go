package autodetect

import (
	"fmt"

	"github.com/devboxos/devboxos/shared/scanner"
	"github.com/devboxos/devboxos/shared/types"
)

func dedupeInts(slice []int) []int {
	seen := make(map[int]bool)
	result := make([]int, 0, len(slice))
	for _, v := range slice {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func AutoDetect(dir string) (*types.Config, error) {
	return AutoDetectWithDepth(dir, 2)
}

func AutoDetectWithDepth(dir string, maxDepth int) (*types.Config, error) {
	s := scanner.NewWithDepth(maxDepth)
	results, err := s.Scan(dir)
	if err != nil {
		return nil, fmt.Errorf("scan project: %w", err)
	}

	cfg := &types.Config{
		Services: make(map[string]types.Service),
		Runtimes: make(map[string]string),
	}

	cfg.Version = "1.0"

	resolved, warnings := scanner.ResolveConflicts(results)

	if msg := scanner.FormatWarnings(warnings); msg != "" {
		fmt.Print(msg)
	}

	for name, rs := range resolved {
		svc := types.Service{}

		if rs.RunCommand != "" {
			svc.Command = rs.RunCommand
		}
		if rs.BuildCommand != "" {
			svc.Build = &types.BuildConfig{
				Context:    rs.BuildCommand,
				Dockerfile: "Dockerfile",
			}
		}
		if len(rs.Env) > 0 {
			svc.Env = rs.Env
		}
		if len(rs.DependsOn) > 0 {
			svc.DependsOn = rs.DependsOn
		}

		svc.Port = rs.ResolvedPort

		if len(rs.AllPorts) > 1 {
			svc.Ports = rs.AllPorts
		}

		switch rs.Language {
		case "node":
			cfg.Runtimes["node"] = "18"
		case "java":
			cfg.Runtimes["java"] = "21"
		case "ruby":
			cfg.Runtimes["ruby"] = "3.3"
		case "php":
			cfg.Runtimes["php"] = "8.3"
		case "postgres":
			svc.Image = "postgres:16-alpine"
		case "mysql":
			svc.Image = "mysql:8"
		case "redis":
			svc.Image = "redis:7-alpine"
		case "mongo":
			svc.Image = "mongo:7"
		}

		cfg.Services[name] = svc
	}

	if len(cfg.Services) == 0 {
		return nil, fmt.Errorf("no services detected in project")
	}

	cfg.Networking = types.Networking{
		Discovery: true,
		Egress:    "default-deny",
	}

	for _, svc := range cfg.Services {
		if svc.Port != "" {
			var port int
			fmt.Sscanf(svc.Port, "%d", &port)
			if port > 0 {
				cfg.Networking.Expose = append(cfg.Networking.Expose, port)
			}
		}
		for _, p := range svc.Ports {
			var port int
			fmt.Sscanf(p, "%d", &port)
			if port > 0 {
				cfg.Networking.Expose = append(cfg.Networking.Expose, port)
			}
		}
	}
	cfg.Networking.Expose = dedupeInts(cfg.Networking.Expose)

	return cfg, nil
}
