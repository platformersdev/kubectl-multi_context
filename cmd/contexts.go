package cmd

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/tools/clientcmd"
)

// Kubeconfig represents the minimal structure needed to read contexts from a kubeconfig file
type Kubeconfig struct {
	Contexts []ContextEntry `yaml:"contexts"`
}

// ContextEntry represents a single context entry in the kubeconfig
type ContextEntry struct {
	Name string `yaml:"name"`
}

func getContexts() ([]string, error) {
	kubeconfigPath := getKubeconfigPath()
	if kubeconfigPath == "" {
		return nil, fmt.Errorf("could not determine kubeconfig path")
	}

	file, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	var config Kubeconfig
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	var contexts []string
	for _, entry := range config.Contexts {
		if entry.Name != "" {
			contexts = append(contexts, entry.Name)
		}
	}

	if len(contexts) == 0 {
		// Fallback to clientcmd if YAML parsing doesn't find contexts
		kubeconfig, err := clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}

		for name := range kubeconfig.Contexts {
			contexts = append(contexts, name)
		}
	}

	if len(contexts) == 0 {
		return nil, fmt.Errorf("no contexts found in kubeconfig")
	}

	// Apply filter if specified
	if filterPattern != "" {
		contexts = filterContexts(contexts, filterPattern)
		if len(contexts) == 0 {
			return nil, fmt.Errorf("no contexts match filter pattern: %s", filterPattern)
		}
	}

	return contexts, nil
}

// filterContexts filters contexts by substring match (case-insensitive)
func filterContexts(contexts []string, pattern string) []string {
	if pattern == "" {
		return contexts
	}

	var filtered []string
	patternLower := strings.ToLower(pattern)
	for _, ctx := range contexts {
		if strings.Contains(strings.ToLower(ctx), patternLower) {
			filtered = append(filtered, ctx)
		}
	}
	return filtered
}

func getKubeconfigPath() string {
	path := os.Getenv("KUBECONFIG")
	if path != "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s/.kube/config", home)
}
