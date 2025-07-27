package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// Dependency represents a system dependency
type Dependency struct {
	Name        string   // Name of the dependency
	Command     string   // Command to execute the dependency
	Description string   // Brief description of the dependency
	InstallURL  string   // URL for installation instructions
	VersionCmd  []string // Command to check the version
}

// DependencyChecker manages system dependency verification
type DependencyChecker struct {
	dependencies []Dependency // List of dependencies to check
}

// NewDependencyChecker creates a new dependency checker instance
func NewDependencyChecker() *DependencyChecker {
	return &DependencyChecker{
		dependencies: []Dependency{
			{
				Name:        "kubectl",
				Command:     "kubectl",
				Description: "Kubernetes command-line tool",
				InstallURL:  "https://kubernetes.io/docs/tasks/tools/",
				VersionCmd:  []string{"kubectl", "version", "--client"},
			},
			{
				Name:        "git",
				Command:     "git",
				Description: "Version control system",
				InstallURL:  "https://git-scm.com/downloads",
				VersionCmd:  []string{"git", "--version"},
			},
		},
	}
}

// CheckAll verifies all required dependencies
func (dc *DependencyChecker) CheckAll() error {
	var missingDeps []string
	var errors []string

	for _, dep := range dc.dependencies {
		if err := dc.checkDependency(dep); err != nil {
			missingDeps = append(missingDeps, dep.Name)
			errors = append(errors, fmt.Sprintf("  • %s: %v", dep.Name, err))
		}
	}

	if len(missingDeps) > 0 {
		return fmt.Errorf("missing required dependencies:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// checkDependency verifies a single dependency
func (dc *DependencyChecker) checkDependency(dep Dependency) error {
	// Check if the command exists in PATH
	fmt.Printf("About to check if the dependency: %s exists on the host system\n", dep.Name)
	if _, err := exec.LookPath(dep.Command); err != nil {
		return fmt.Errorf("command '%s' not found in PATH", dep.Command)
	}

	// Execute the version command to ensure the tool is functional
	cmd := exec.Command(dep.VersionCmd[0], dep.VersionCmd[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command '%s' exists but failed to execute: %v", dep.Command, err)
	}

	// Validate the output for specific tools
	switch dep.Name {
	case "kubectl":
		if !strings.Contains(strings.ToLower(string(output)), "client version") {
			return fmt.Errorf("kubectl exists but may be corrupted (unexpected version output)")
		}
	case "git":
		if !strings.Contains(strings.ToLower(string(output)), "git version") {
			return fmt.Errorf("git exists but may be corrupted (unexpected version output)")
		}
	}
	fmt.Printf("Dependency %s verified to exist\n", dep.Name)
	return nil
}

// GetDependencyInfo returns information about a specific dependency
func (dc *DependencyChecker) GetDependencyInfo(name string) (*Dependency, error) {
	for _, dep := range dc.dependencies {
		if dep.Name == name {
			return &dep, nil
		}
	}
	return nil, fmt.Errorf("dependency '%s' not found", name)
}

// GetVersion returns the version of a dependency
func (dc *DependencyChecker) GetVersion(name string) (string, error) {
	dep, err := dc.GetDependencyInfo(name)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(dep.VersionCmd[0], dep.VersionCmd[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get version for %s: %v", name, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// VerifyKubectlConnection tests kubectl connectivity with a cluster
func (dc *DependencyChecker) VerifyKubectlConnection(kubeconfigPath string) error {
	if kubeconfigPath == "" {
		return fmt.Errorf("kubeconfig path cannot be empty")
	}

	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfigPath, "cluster-info", "--request-timeout=10s")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to connect to cluster using kubeconfig: %v", err)
	}

	return nil
}

// VerifyGitRepository tests git connectivity with a repository
func (dc *DependencyChecker) VerifyGitRepository(repoURL string) error {
	if repoURL == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}

	// Test git ls-remote to verify repository access without cloning
	cmd := exec.Command("git", "ls-remote", "--heads", repoURL)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to access git repository '%s': %v", repoURL, err)
	}

	return nil
}