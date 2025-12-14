package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetKubeconfigPath(t *testing.T) {
	tests := []struct {
		name           string
		kubeconfigEnv  string
		expectedPrefix string
		expectedSuffix string
	}{
		{
			name:           "with KUBECONFIG env set",
			kubeconfigEnv:  "/custom/path/config",
			expectedPrefix: "/custom/path/config",
			expectedSuffix: "",
		},
		{
			name:           "without KUBECONFIG env",
			kubeconfigEnv:  "",
			expectedSuffix: ".kube/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			originalKubeconfig := os.Getenv("KUBECONFIG")

			// Set test value
			if tt.kubeconfigEnv != "" {
				os.Setenv("KUBECONFIG", tt.kubeconfigEnv)
			} else {
				os.Unsetenv("KUBECONFIG")
			}

			// Clean up
			defer func() {
				if originalKubeconfig != "" {
					os.Setenv("KUBECONFIG", originalKubeconfig)
				} else {
					os.Unsetenv("KUBECONFIG")
				}
			}()

			result := getKubeconfigPath()

			if tt.expectedPrefix != "" {
				if result != tt.expectedPrefix {
					t.Errorf("getKubeconfigPath() = %q, want %q", result, tt.expectedPrefix)
				}
			} else {
				// Check that it ends with the expected suffix
				if !filepath.IsAbs(result) {
					t.Errorf("getKubeconfigPath() = %q, want absolute path", result)
				}
				if filepath.Base(result) != "config" {
					// Check that it's in a .kube directory
					dir := filepath.Dir(result)
					if filepath.Base(dir) != ".kube" {
						t.Errorf("getKubeconfigPath() = %q, want path ending in .kube/config", result)
					}
				}
			}
		})
	}
}
