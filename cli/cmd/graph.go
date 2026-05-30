package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/devboxos/devboxos/shared/config"
	"github.com/devboxos/devboxos/shared/types"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Visualize service dependency tree",
	Long: `Display the service dependency graph in ASCII format.

Shows which services depend on each other and the startup order.

Example:
  devbox graph
    web ───► api ───► db
                  └──► redis`,
	RunE: runGraph,
}

func init() {
	rootCmd.AddCommand(graphCmd)
}

func runGraph(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	parser := config.NewParser()
	cfg, err := parser.Parse(dir)
	if err != nil {
		return fmt.Errorf("parse devbox config: %w", err)
	}

	if len(cfg.Services) == 0 {
		fmt.Println("No services defined in devbox.yml")
		return nil
	}

	names := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	services := cfg.Services

	type node struct {
		name     string
		children []string
		depth    int
	}

	nodes := make(map[string]*node)
	for _, name := range names {
		nodes[name] = &node{name: name}
	}

	rootNames := make([]string, 0)
	for _, name := range names {
		svc := services[name]
		if len(svc.DependsOn) == 0 {
			rootNames = append(rootNames, name)
		}
		for _, dep := range svc.DependsOn {
			if n, ok := nodes[dep]; ok {
				n.children = append(n.children, name)
			}
		}
	}

	if len(rootNames) == 0 && len(names) > 0 {
		rootNames = names
	}

	fmt.Println()
	for _, rootName := range rootNames {
		printTree(rootName, services, "", true, true)
	}
	fmt.Println()

	return nil
}

func printTree(name string, services map[string]types.Service, prefix string, isLast bool, isRoot bool) {
	svc := services[name]

	connector := "├──"
	if isLast {
		connector = "└──"
	}
	if isRoot {
		connector = ""
	}

	label := name
	if svc.Port != "" || len(svc.Ports) > 0 {
		port := svc.Port
		if port == "" {
			port = svc.Ports[0]
		}
		label = fmt.Sprintf("%s [%s]", name, port)
	}

	if isRoot {
		fmt.Printf("  %s\n", label)
	} else {
		fmt.Printf("%s%s %s\n", prefix, connector, label)
	}

	children := svc.DependsOn
	if len(children) == 0 && !isRoot {
		return
	}

	childPrefix := prefix
	if !isRoot {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	for i, child := range children {
		isLastChild := i == len(children)-1
		printTree(child, services, childPrefix, isLastChild, false)
	}
}
