package scanner

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type ResolvedService struct {
	Name         string
	OriginalPort string
	ResolvedPort string
	AllPorts     []string
	Language     string
	Subdirectory string
	Env          map[string]string
	DependsOn    []string
	Image        string
	BuildCommand string
	RunCommand   string
}

type ResolutionWarnings struct {
	Conflicts []string
	Used      []string
}

func ResolveConflicts(results []ScanResult) (map[string]ResolvedService, ResolutionWarnings) {
	services := make(map[string]ResolvedService)
	var warnings ResolutionWarnings

	portUsage := make(map[int]string)
	assigned := make(map[string]int)

	conflictCounter := make(map[int]int)

	serviceNames := make([]string, 0, len(results))
	for _, r := range results {
		serviceNames = append(serviceNames, r.ServiceName)
	}

	sort.Strings(serviceNames)

	for _, name := range serviceNames {
		result := findResult(name, results)
		if result == nil {
			continue
		}

		rs := ResolvedService{
			Name:         result.ServiceName,
			Language:     result.Language,
			Subdirectory: result.Subdirectory,
			Env:          result.Env,
			DependsOn:    result.DependsOn,
			Image:        result.Image,
			BuildCommand: result.BuildCommand,
			RunCommand:   result.RunCommand,
		}

		if len(result.Ports) == 0 {
			if def, ok := KnownDefault(result.Language); ok {
				rs.OriginalPort = strconv.Itoa(def)
				rs.AllPorts = []string{rs.OriginalPort}
			}
		} else {
			bestPort := bestPort(result.Ports)
			rs.OriginalPort = strconv.Itoa(bestPort)

			var allPorts []int
			seen := make(map[int]bool)
			for _, dp := range result.Ports {
				if !seen[dp.Port] {
					allPorts = append(allPorts, dp.Port)
					seen[dp.Port] = true
				}
			}
			sort.Ints(allPorts)
			for _, p := range allPorts {
				rs.AllPorts = append(rs.AllPorts, strconv.Itoa(p))
			}
		}

		services[name] = rs
	}

	for _, name := range serviceNames {
		rs := services[name]
		if rs.OriginalPort == "" {
			continue
		}

		portNum, err := strconv.Atoi(strings.TrimSuffix(rs.OriginalPort, "/tcp"))
		if err != nil {
			rs.ResolvedPort = rs.OriginalPort
			services[name] = rs
			continue
		}

		if existing, ok := portUsage[portNum]; ok {
			conflictCounter[portNum]++
			newPort := portNum + conflictCounter[portNum]
			for portUsage[newPort] != "" {
				conflictCounter[portNum]++
				newPort = portNum + conflictCounter[portNum]
			}
			rs.ResolvedPort = strconv.Itoa(newPort)
			portUsage[newPort] = name
			assigned[name] = newPort

			warnings.Conflicts = append(warnings.Conflicts,
				fmt.Sprintf("%s → %d (was %d, conflicted with %s)", name, newPort, portNum, existing))
		} else {
			rs.ResolvedPort = rs.OriginalPort
			portUsage[portNum] = name
			assigned[name] = portNum
		}

		services[name] = rs
	}

	var usedPorts []string
	for _, name := range serviceNames {
		rs := services[name]
		if rs.ResolvedPort != "" {
			usedPorts = append(usedPorts, fmt.Sprintf("%s=%s", rs.Name, rs.ResolvedPort))
		}
	}
	warnings.Used = usedPorts

	return services, warnings
}

func bestPort(ports []DetectedPort) int {
	if len(ports) == 0 {
		return 0
	}
	sort.Slice(ports, func(i, j int) bool {
		if ports[i].Priority != ports[j].Priority {
			return ports[i].Priority > ports[j].Priority
		}
		return ports[i].Port < ports[j].Port
	})
	return ports[0].Port
}

func findResult(name string, results []ScanResult) *ScanResult {
	for i := range results {
		if results[i].ServiceName == name {
			return &results[i]
		}
	}
	return nil
}

func FormatWarnings(warnings ResolutionWarnings) string {
	if len(warnings.Conflicts) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Port conflicts resolved:\n")
	for _, w := range warnings.Conflicts {
		sb.WriteString(fmt.Sprintf("  • %s\n", w))
	}
	return sb.String()
}
