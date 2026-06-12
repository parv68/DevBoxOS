package config

import (
	"fmt"
	"strings"

	"github.com/devboxos/devboxos/shared/scanner"
	"github.com/devboxos/devboxos/shared/types"
)

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
		Networking: types.Networking{
			Discovery: true,
			Egress:    "default-deny",
		},
	}

	cfg.Version = "1.0"

	resolved, warnings := scanner.ResolveConflicts(results)

	if msg := scanner.FormatWarnings(warnings); msg != "" {
		fmt.Print(msg)
	}

	for name, rs := range resolved {
		svc := types.Service{
			DependsOn: rs.DependsOn,
		}

		switch rs.Language {
		case "node":
			svc.Image = "node:18"
			if v, ok := rs.Env["NODE_VERSION"]; ok {
				svc.Image = fmt.Sprintf("node:%s", v)
			}
		case "python":
			svc.Image = "python:3.11-slim"
		case "go":
			svc.Image = "golang:1.22"
		case "rust":
			svc.Image = "rust:1.75"
		case "java":
			svc.Image = "eclipse-temurin:21"
		case "ruby":
			svc.Image = "ruby:3.3-slim"
		case "php":
			svc.Image = "php:8.3-cli"
		case "docker":
			if rs.Image != "" {
				svc.Image = rs.Image
			}
		}

		if rs.BuildCommand != "" {
			svc.Build = &types.BuildConfig{
				Context:    rs.BuildCommand,
				Dockerfile: "Dockerfile",
			}
			svc.Image = ""
		}

		if len(rs.Env) > 0 {
			if svc.Env == nil {
				svc.Env = make(map[string]string)
			}
			for k, v := range rs.Env {
				if strings.EqualFold(k, "NODE_VERSION") {
					continue
				}
				svc.Env[k] = v
			}
		}

		svc.Port = rs.ResolvedPort

		if len(rs.AllPorts) > 1 {
			svc.Ports = rs.AllPorts
		}

		cfg.Services[name] = svc

		switch rs.Language {
		case "node":
			cfg.Runtimes["node"] = "18"
		case "java":
			cfg.Runtimes["java"] = "21"
		case "ruby":
			cfg.Runtimes["ruby"] = "3.3"
		case "php":
			cfg.Runtimes["php"] = "8.3"
		}
	}

	if len(cfg.Services) == 0 {
		return nil, fmt.Errorf("no services detected in project")
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

	return cfg, nil
}
