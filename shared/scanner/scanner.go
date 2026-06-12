package scanner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type DetectedPort struct {
	Port     int
	Source   string
	File     string
	Line     int
	Priority int
}

type ScanResult struct {
	ServiceName  string
	Language     string
	Subdirectory string
	Ports        []DetectedPort
	Env          map[string]string
	DependsOn    []string
	Image        string
	Dockerfile   string
	BuildCommand string
	RunCommand   string
}

type Scanner struct {
	MaxDepth int
}

func New() *Scanner {
	return NewWithDepth(2)
}

func NewWithDepth(depth int) *Scanner {
	if depth < 1 {
		depth = 2
	}
	return &Scanner{MaxDepth: depth}
}

var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, ".venv": true,
	"__pycache__": true, ".next": true, "dist": true,
	"build": true, "target": true, "vendor": true,
	".terraform": true, ".serverless": true, ".cache": true,
}

func (s *Scanner) Scan(dir string) ([]ScanResult, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory %s does not exist", dir)
	}

	var results []ScanResult

	rootResult, _ := s.scanDir(dir, "")
	if rootResult != nil {
		results = append(results, *rootResult)
	}

	groups := s.findProjectGroups(dir)
	for _, sub := range groups {
		rel, _ := filepath.Rel(dir, sub)
		result, err := s.scanDir(sub, rel)
		if err != nil {
			continue
		}
		if result != nil {
			results = append(results, *result)
		}
	}

	if len(results) == 0 {
		result := s.scanGeneric(dir, "")
		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

func (s *Scanner) findProjectGroups(dir string) []string {
	var groups []string
	s.findProjectGroupsRecursive(dir, 0, &groups)
	return groups
}

func (s *Scanner) findProjectGroupsRecursive(dir string, depth int, groups *[]string) {
	if depth >= s.MaxDepth {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || skipDirs[e.Name()] || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		sub := filepath.Join(dir, e.Name())
		if hasProjectIndicator(sub) {
			*groups = append(*groups, sub)
		}
		s.findProjectGroupsRecursive(sub, depth+1, groups)
	}
}

func hasProjectIndicator(dir string) bool {
	indicators := []string{
		"package.json", "go.mod", "Cargo.toml",
		"requirements.txt", "pyproject.toml", "setup.py",
		"Dockerfile", "docker-compose.yml", ".env",
		"vite.config.js", "vite.config.ts",
		"next.config.js", "nuxt.config.js",
		"angular.json", "vue.config.js",
		"pom.xml", "build.gradle",
		"Gemfile", "config.ru",
		"composer.json", "artisan",
	}
	for _, ind := range indicators {
		if _, err := os.Stat(filepath.Join(dir, ind)); err == nil {
			return true
		}
	}
	return false
}

func (s *Scanner) scanDir(dir string, rel string) (*ScanResult, error) {
	result := &ScanResult{
		ServiceName:  deriveServiceName(dir, rel),
		Subdirectory: rel,
		Ports:        []DetectedPort{},
		Env:          make(map[string]string),
	}

	detected := false

	if hasFile(dir, "package.json") {
		detected = true
		result.Language = "node"
		result.RunCommand = "npm run dev"
		r := scanNodeJS(dir)
		result.Ports = append(result.Ports, r...)
	}

	if hasFile(dir, "go.mod") {
		detected = true
		result.Language = "go"
		result.RunCommand = "go run ."
		r := scanGo(dir)
		result.Ports = append(result.Ports, r...)
	}

	if hasFile(dir, "Cargo.toml") {
		detected = true
		result.Language = "rust"
		result.RunCommand = "cargo run"
		r := scanRust(dir)
		result.Ports = append(result.Ports, r...)
	}

	if hasFile(dir, "requirements.txt") || hasFile(dir, "pyproject.toml") || hasFile(dir, "setup.py") {
		detected = true
		result.Language = "python"
		result.RunCommand = "python -m app"
		r := scanPython(dir)
		result.Ports = append(result.Ports, r...)
	}

	if hasFile(dir, "pom.xml") || hasFile(dir, "build.gradle") || hasFile(dir, "build.gradle.kts") {
		detected = true
		if result.Language == "" {
			result.Language = "java"
		}
		if result.RunCommand == "" {
			if hasFile(dir, "gradlew") {
				result.RunCommand = "./gradlew bootRun"
			} else if hasFile(dir, "mvnw") {
				result.RunCommand = "./mvnw spring-boot:run"
			} else {
				result.RunCommand = "mvn spring-boot:run"
			}
		}
		r := scanJava(dir)
		result.Ports = append(result.Ports, r...)
	}

	if hasFile(dir, "Gemfile") || hasFile(dir, "config.ru") {
		detected = true
		if result.Language == "" {
			result.Language = "ruby"
		}
		if result.RunCommand == "" {
			result.RunCommand = "bundle exec rails s"
		}
		r := scanRuby(dir)
		result.Ports = append(result.Ports, r...)
	}

	if hasFile(dir, "composer.json") || hasFile(dir, "artisan") {
		detected = true
		if result.Language == "" {
			result.Language = "php"
		}
		if result.RunCommand == "" {
			result.RunCommand = "php artisan serve"
		}
		r := scanPHP(dir)
		result.Ports = append(result.Ports, r...)
	}

	if hasFile(dir, "Dockerfile") {
		detected = true
		if result.Language == "" {
			result.Language = "docker"
		}
		result.BuildCommand = "."
		result.Dockerfile = "Dockerfile"
		r := scanDockerfile(filepath.Join(dir, "Dockerfile"))
		result.Ports = append(result.Ports, r...)
	}

	if hasFile(dir, "docker-compose.yml") {
		detected = true
		composeSvc := importDockerCompose(filepath.Join(dir, "docker-compose.yml"))
		for name, ports := range composeSvc {
			if name == result.ServiceName || len(result.Ports) == 0 {
				for _, p := range ports {
					result.Ports = append(result.Ports, p)
				}
			}
		}
	}

	if hasFile(dir, ".env") || hasFile(dir, ".env.local") || hasFile(dir, ".env.development") {
		envPorts, envVars := scanEnvFile(dir)
		result.Ports = append(result.Ports, envPorts...)
		for k, v := range envVars {
			result.Env[k] = v
		}
	}

	if !detected {
		return nil, nil
	}

	result.Ports = dedupePorts(result.Ports)
	return result, nil
}

func (s *Scanner) scanGeneric(dir string, rel string) *ScanResult {
	result := &ScanResult{
		ServiceName:  deriveServiceName(dir, rel),
		Subdirectory: rel,
		Ports:        []DetectedPort{},
	}

	if hasFile(dir, ".env") {
		envPorts, _ := scanEnvFile(dir)
		result.Ports = append(result.Ports, envPorts...)
	}

	r := scanConfigFiles(dir)
	result.Ports = append(result.Ports, r...)

	if hasFile(dir, "Dockerfile") {
		r := scanDockerfile(filepath.Join(dir, "Dockerfile"))
		result.Ports = append(result.Ports, r...)
		result.Language = "docker"
		result.BuildCommand = "."
		result.Dockerfile = "Dockerfile"
	}

	if len(result.Ports) > 0 {
		result.Ports = dedupePorts(result.Ports)
		return result
	}
	return nil
}

func hasFile(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

var DefaultPorts = map[string]int{
	"node":   3000,
	"go":     8080,
	"rust":   8080,
	"python": 8000,
	"docker": 8080,
	"java":   8080,
	"ruby":   3000,
	"php":    8000,
}

func KnownDefault(language string) (int, bool) {
	p, ok := DefaultPorts[language]
	return p, ok
}

var (
	nodeListenRe = regexp.MustCompile(`(?:app|server|http|express|fastify|restana|polka)\.(?:listen|run)\((\d+)`)

	goListenRe   = regexp.MustCompile(`ListenAndServe(?:TLS)?\([^)]*:(\d+)`)
	goGinRe      = regexp.MustCompile(`\.Run\([^)]*:(\d+)`)
	goEchoRe     = regexp.MustCompile(`\.Start\([^)]*:(\d+)`)

	rustBindRe   = regexp.MustCompile(`(?:bind|TcpListener::bind)\(["']([\d.]+):(\d+)["']`)

	pythonRunRe  = regexp.MustCompile(`(?:app|uvicorn|flask|fastapi|sanic|aiohttp)\.run\([^)]*port\s*=\s*(\d+)`)
	pythonCLIRe  = regexp.MustCompile(`runserver[^0-9]*(\d+)`)
	pythonPortRe = regexp.MustCompile(`(?:--port|-p)\s+(\d+)`)

	exposeRe     = regexp.MustCompile(`EXPOSE\s+(\d+)`)

	envPortRe    = regexp.MustCompile(`^(?:PORT|SERVER_PORT|APP_PORT|API_PORT|HTTP_PORT|BACKEND_PORT|FRONTEND_PORT|VITE_PORT|NEXT_PUBLIC_PORT)\s*=\s*(\d+)$`)

	configPortRe = regexp.MustCompile(`[^a-zA-Z]port[:\s]+(\d+)`)
)

func scanNodeJS(dir string) []DetectedPort {
	var ports []DetectedPort
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(d.Name())
		if ext != ".js" && ext != ".ts" && ext != ".mjs" && ext != ".json" && ext != ".mts" {
			return nil
		}
		if filepath.Base(path) == "package.json" {
			p := scanPackageJSON(path)
			ports = append(ports, p...)
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if m := nodeListenRe.FindStringSubmatch(line); len(m) > 1 {
				if p, err := strconv.Atoi(m[1]); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "listener", File: path, Line: lineNum, Priority: 10})
				}
			}
		}
		return nil
	})

	ports = append(ports, scanViteConfig(dir)...)
	ports = append(ports, scanNextConfig(dir)...)

	return ports
}

func scanPackageJSON(path string) []DetectedPort {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
		Port    int               `json:"port"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	var ports []DetectedPort
	if pkg.Port > 0 {
		ports = append(ports, DetectedPort{Port: pkg.Port, Source: "package.json:port", File: path, Priority: 5})
	}

	scriptPort := regexp.MustCompile(`(?:PORT|PORT_NUMBER|SERVER_PORT)=(\d+)`)
	for _, script := range pkg.Scripts {
		if m := scriptPort.FindStringSubmatch(script); len(m) > 1 {
			if p, err := strconv.Atoi(m[1]); err == nil {
				ports = append(ports, DetectedPort{Port: p, Source: "package.json:script", File: path, Priority: 5})
			}
		}
	}
	return ports
}

func scanViteConfig(dir string) []DetectedPort {
	candidates := []string{"vite.config.js", "vite.config.ts", "vite.config.mjs", "vite.config.mts"}
	for _, name := range candidates {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		re := regexp.MustCompile(`port\s*:\s*(\d+)`)
		if m := re.FindSubmatch(data); len(m) > 1 {
			if p, err := strconv.Atoi(string(m[1])); err == nil {
				return []DetectedPort{{Port: p, Source: "vite:config", File: name, Priority: 8}}
			}
		}
	}
	return nil
}

func scanNextConfig(dir string) []DetectedPort {
	candidates := []string{"next.config.js", "next.config.mjs"}
	for _, name := range candidates {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		re := regexp.MustCompile(`port\s*:\s*(\d+)`)
		if m := re.FindSubmatch(data); len(m) > 1 {
			if p, err := strconv.Atoi(string(m[1])); err == nil {
				return []DetectedPort{{Port: p, Source: "next:config", File: name, Priority: 8}}
			}
		}
	}
	return nil
}

func scanGo(dir string) []DetectedPort {
	var ports []DetectedPort
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || (d != nil && d.IsDir() && skipDirs[d.Name()]) {
			return nil
		}
		if filepath.Ext(d.Name()) != ".go" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if m := goListenRe.FindStringSubmatch(line); len(m) > 1 {
				if p, err := strconv.Atoi(m[1]); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "go:ListenAndServe", File: path, Line: lineNum, Priority: 10})
				}
			}
			if m := goGinRe.FindStringSubmatch(line); len(m) > 1 {
				if p, err := strconv.Atoi(m[1]); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "go:Gin", File: path, Line: lineNum, Priority: 10})
				}
			}
			if m := goEchoRe.FindStringSubmatch(line); len(m) > 1 {
				if p, err := strconv.Atoi(m[1]); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "go:Echo", File: path, Line: lineNum, Priority: 10})
				}
			}
		}
		return nil
	})
	return ports
}

func scanRust(dir string) []DetectedPort {
	var ports []DetectedPort
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || (d != nil && d.IsDir() && skipDirs[d.Name()]) {
			return nil
		}
		if filepath.Ext(d.Name()) != ".rs" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if m := rustBindRe.FindStringSubmatch(line); len(m) > 2 {
				if p, err := strconv.Atoi(m[2]); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "rust:bind", File: path, Line: lineNum, Priority: 10})
				}
			}
		}
		return nil
	})

	if len(ports) == 0 {
		re := regexp.MustCompile(`:(\d{4,5})`)
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || (d != nil && d.IsDir() && skipDirs[d.Name()]) {
				return nil
			}
			if filepath.Ext(d.Name()) != ".rs" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if m := re.FindSubmatch(data); len(m) > 1 {
				if p, err := strconv.Atoi(string(m[1])); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "rust:port-pattern", File: path, Priority: 5})
				}
			}
			return nil
		})
	}

	return ports
}

func scanPython(dir string) []DetectedPort {
	var ports []DetectedPort
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || (d != nil && d.IsDir() && skipDirs[d.Name()]) {
			return nil
		}
		if filepath.Ext(d.Name()) != ".py" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if m := pythonRunRe.FindStringSubmatch(line); len(m) > 1 {
				if p, err := strconv.Atoi(m[1]); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "python:run", File: path, Line: lineNum, Priority: 10})
				}
			}
			if m := pythonCLIRe.FindStringSubmatch(line); len(m) > 1 {
				if p, err := strconv.Atoi(m[1]); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "python:cli", File: path, Line: lineNum, Priority: 7})
				}
			}
			if m := pythonPortRe.FindStringSubmatch(line); len(m) > 1 {
				if p, err := strconv.Atoi(m[1]); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "python:port-flag", File: path, Line: lineNum, Priority: 7})
				}
			}
		}
		return nil
	})
	return ports
}

func scanJava(dir string) []DetectedPort {
	var ports []DetectedPort

	candidates := []string{
		"application.properties",
		"application-dev.properties",
		"application-prod.properties",
		"src/main/resources/application.properties",
		"src/main/resources/application-dev.properties",
	}
	propRe := regexp.MustCompile(`server\.port\s*[=:]\s*(\d+)`)
	for _, name := range candidates {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		if m := propRe.FindSubmatch(data); len(m) > 1 {
			if p, err := strconv.Atoi(string(m[1])); err == nil {
				ports = append(ports, DetectedPort{Port: p, Source: "java:server.port", File: name, Priority: 10})
			}
		}
	}

	ymlCandidates := []string{
		"application.yml", "application.yaml",
		"src/main/resources/application.yml", "src/main/resources/application.yaml",
	}
	ymlRe := regexp.MustCompile(`port\s*:\s*(\d+)`)
	for _, name := range ymlCandidates {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		if m := ymlRe.FindSubmatch(data); len(m) > 1 {
			if p, err := strconv.Atoi(string(m[1])); err == nil {
				ports = append(ports, DetectedPort{Port: p, Source: "java:yml:port", File: name, Priority: 10})
			}
		}
	}

	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || (d != nil && d.IsDir() && skipDirs[d.Name()]) {
			return nil
		}
		if filepath.Ext(d.Name()) != ".java" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		setPortRe := regexp.MustCompile(`setPort\s*\(\s*(\d+)\s*\)`)
		for _, m := range setPortRe.FindAllSubmatch(data, -1) {
			if len(m) > 1 {
				if p, err := strconv.Atoi(string(m[1])); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "java:setPort", File: path, Priority: 8})
				}
			}
		}
		return nil
	})

	return ports
}

func scanRuby(dir string) []DetectedPort {
	var ports []DetectedPort

	pumaFiles := []string{"config/puma.rb", "config/puma.rb.example"}
	pumaPortRe := regexp.MustCompile(`port\s+(?:.*?\{\s*)?(\d+)`)
	for _, name := range pumaFiles {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		for _, m := range pumaPortRe.FindAllSubmatch(data, -1) {
			if len(m) > 1 {
				if p, err := strconv.Atoi(string(m[1])); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "ruby:puma", File: name, Priority: 10})
				}
			}
		}
	}

	configRe := regexp.MustCompile(`port\s*[=:>]\s*(\d+)`)
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || (d != nil && d.IsDir() && skipDirs[d.Name()]) {
			return nil
		}
		ext := filepath.Ext(d.Name())
		if ext != ".rb" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, m := range configRe.FindAllSubmatch(data, -1) {
			if len(m) > 1 {
				if p, err := strconv.Atoi(string(m[1])); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "ruby:config", File: path, Priority: 7})
				}
			}
		}
		return nil
	})

	return ports
}

func scanPHP(dir string) []DetectedPort {
	var ports []DetectedPort

	if hasFile(dir, "composer.json") {
		data, err := os.ReadFile(filepath.Join(dir, "composer.json"))
		if err == nil {
			re := regexp.MustCompile(`(?:php\s+(?:-\S+\s+)*?-S\s+\S+:)(\d+)|--port[= ](\d+)|serve[^a-z]*--port[= ](\d+)`)
			for _, m := range re.FindAllSubmatch(data, -1) {
				for i := 1; i < len(m); i++ {
					if len(m[i]) > 0 {
						if p, err := strconv.Atoi(string(m[i])); err == nil {
							ports = append(ports, DetectedPort{Port: p, Source: "php:composer", File: "composer.json", Priority: 8})
						}
					}
				}
			}
		}
	}

	envFileRe := regexp.MustCompile(`APP_PORT\s*=\s*(\d+)`)
	envFiles := []string{".env", ".env.local", ".env.production"}
	for _, name := range envFiles {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		if m := envFileRe.FindSubmatch(data); len(m) > 1 {
			if p, err := strconv.Atoi(string(m[1])); err == nil {
				ports = append(ports, DetectedPort{Port: p, Source: "php:APP_PORT", File: name, Priority: 10})
			}
		}
	}

	return ports
}

func scanDockerfile(path string) []DetectedPort {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var ports []DetectedPort
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if m := exposeRe.FindStringSubmatch(line); len(m) > 1 {
			if p, err := strconv.Atoi(m[1]); err == nil {
				ports = append(ports, DetectedPort{Port: p, Source: "Dockerfile:EXPOSE", File: path, Line: i + 1, Priority: 10})
			}
		}
	}
	return ports
}

func importDockerCompose(path string) map[string][]DetectedPort {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var compose struct {
		Services map[string]struct {
			Ports []string `yaml:"ports"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil
	}
	result := make(map[string][]DetectedPort)
	for name, svc := range compose.Services {
		for _, p := range svc.Ports {
			hostPort := strings.Split(p, ":")[0]
			if idx := strings.Index(hostPort, "-"); idx > 0 {
				if start, err := strconv.Atoi(hostPort[:idx]); err == nil {
					result[name] = append(result[name], DetectedPort{Port: start, Source: "compose:port-range-start", Priority: 7})
				}
				continue
			}
			portNum, err := strconv.Atoi(strings.TrimSpace(hostPort))
			if err != nil {
				continue
			}
			result[name] = append(result[name], DetectedPort{Port: portNum, Source: "compose:ports", Priority: 8})
		}
		if len(svc.Ports) == 0 {
			result[name] = append(result[name], DetectedPort{Port: 8080, Source: "compose:default", Priority: 1})
		}
	}
	return result
}

func scanEnvFile(dir string) ([]DetectedPort, map[string]string) {
	var ports []DetectedPort
	vars := make(map[string]string)
	envFiles := []string{".env", ".env.local", ".env.development", ".env.production"}
	for _, name := range envFiles {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		used := false
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if m := envPortRe.FindStringSubmatch(line); len(m) > 1 {
				if p, err := strconv.Atoi(m[1]); err == nil {
					ports = append(ports, DetectedPort{Port: p, Source: "env:" + name, File: name, Line: i + 1, Priority: 10})
					used = true
				}
			}
			if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
				vars[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
		if !used {
			for k, v := range vars {
				if strings.Contains(strings.ToUpper(k), "PORT") {
					if p, err := strconv.Atoi(v); err == nil && p > 0 {
						ports = append(ports, DetectedPort{Port: p, Source: "env:" + name + ":" + k, File: name, Priority: 7})
					}
				}
			}
		}
	}
	return ports, vars
}

func scanConfigFiles(dir string) []DetectedPort {
	var ports []DetectedPort
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || (d != nil && d.IsDir() && skipDirs[d.Name()]) {
			return nil
		}
		ext := filepath.Ext(d.Name())
		if ext != ".yml" && ext != ".yaml" && ext != ".json" && ext != ".toml" && ext != ".ini" && ext != ".cfg" {
			return nil
		}
		if strings.Contains(d.Name(), "node_modules") || strings.Contains(d.Name(), "package-lock") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, m := range configPortRe.FindAllSubmatch(data, -1) {
			if len(m) > 1 {
				if p, err := strconv.Atoi(string(m[1])); err == nil && p > 0 && p < 65536 {
					ports = append(ports, DetectedPort{Port: p, Source: "config-file", File: path, Priority: 4})
				}
			}
		}
		return nil
	})
	return ports
}

func deriveServiceName(dir string, rel string) string {
	if rel == "" || rel == "." {
		name := filepath.Base(dir)
		name = strings.TrimSuffix(name, ".git")
		return strings.ToLower(name)
	}
	name := filepath.Base(dir)
	name = strings.TrimSuffix(name, ".git")
	return strings.ToLower(name)
}

func dedupePorts(ports []DetectedPort) []DetectedPort {
	seen := make(map[int]DetectedPort)
	for _, p := range ports {
		if existing, ok := seen[p.Port]; ok {
			if p.Priority > existing.Priority {
				seen[p.Port] = p
			}
		} else {
			seen[p.Port] = p
		}
	}
	var result []DetectedPort
	for _, p := range seen {
		result = append(result, p)
	}
	return result
}
