package kubectl

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/RaymondAkachi/custom-kub-cli/internal/config"
)

// Executor handles kubectl command execution
type Executor struct {
	cluster *config.ClusterInfo
	timeout time.Duration
}

// NewExecutor creates a new kubectl executor for a cluster
func NewExecutor(cluster *config.ClusterInfo) *Executor {
	return &Executor{
		cluster: cluster,
		timeout: 30 * time.Second, // Default timeout
	}
}

// SetTimeout sets the command execution timeout
func (e *Executor) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}

// Execute runs a kubectl command and returns the output
func (e *Executor) Execute(args ...string) (string, error) {
	if e.cluster == nil {
		return "", fmt.Errorf("no cluster configured")
	}

	// Prepare kubectl command with kubeconfig
	cmdArgs := []string{"--kubeconfig", e.cluster.ConfigPath}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("kubectl", cmdArgs...)
	
	// Set timeout
	if e.timeout > 0 {
		go func() {
			time.Sleep(e.timeout)
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("kubectl command failed: %v", err)
	}

	return string(output), nil
}

// ExecuteCommand parses a command string and executes it
func (e *Executor) ExecuteCommand(command string) (string, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	return e.Execute(parts...)
}

// TestConnection tests connectivity to the cluster
func (e *Executor) TestConnection() error {
	_, err := e.Execute("cluster-info", "--request-timeout=10s")
	return err
}

// GetClusterInfo returns cluster information
func (e *Executor) GetClusterInfo() (string, error) {
	return e.Execute("cluster-info")
}

// GetVersion returns kubectl and cluster version information
func (e *Executor) GetVersion() (string, error) {
	return e.Execute("version", "--output=yaml")
}

// GetNodes returns cluster nodes
func (e *Executor) GetNodes() (string, error) {
	return e.Execute("get", "nodes", "-o", "wide")
}

// GetPods returns pods in all namespaces
func (e *Executor) GetPods() (string, error) {
	return e.Execute("get", "pods", "--all-namespaces")
}

// GetNamespaces returns all namespaces
func (e *Executor) GetNamespaces() (string, error) {
	return e.Execute("get", "namespaces")
}

// GetServices returns services in all namespaces
func (e *Executor) GetServices() (string, error) {
	return e.Execute("get", "services", "--all-namespaces")
}

// GetDeployments returns deployments in all namespaces
func (e *Executor) GetDeployments() (string, error) {
	return e.Execute("get", "deployments", "--all-namespaces")
}

// Apply applies a configuration file or resource
func (e *Executor) Apply(args ...string) (string, error) {
	cmdArgs := append([]string{"apply"}, args...)
	return e.Execute(cmdArgs...)
}

// Create creates resources
func (e *Executor) Create(args ...string) (string, error) {
	cmdArgs := append([]string{"create"}, args...)
	return e.Execute(cmdArgs...)
}

// Delete deletes resources
func (e *Executor) Delete(args ...string) (string, error) {
	cmdArgs := append([]string{"delete"}, args...)
	return e.Execute(cmdArgs...)
}

// Scale scales a deployment
func (e *Executor) Scale(deployment string, replicas int) (string, error) {
	return e.Execute("scale", "deployment", deployment, fmt.Sprintf("--replicas=%d", replicas))
}

// Logs gets logs from a pod
func (e *Executor) Logs(podName string, args ...string) (string, error) {
	cmdArgs := append([]string{"logs", podName}, args...)
	return e.Execute(cmdArgs...)
}

// Describe describes a resource
func (e *Executor) Describe(resourceType, resourceName string) (string, error) {
	return e.Execute("describe", resourceType, resourceName)
}

// IsModifyingCommand checks if a command modifies cluster state
func IsModifyingCommand(command string) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	modifyingCommands := map[string]bool{
		"create":    true,
		"apply":     true,
		"patch":     true,
		"replace":   true,
		"delete":    true,
		"scale":     true,
		"annotate":  true,
		"label":     true,
		"expose":    true,
		"rollout":   true,
		"set":       true,
		"edit":      true,
		"cp":        true,
		"drain":     true,
		"cordon":    true,
		"uncordon":  true,
		"taint":     true,
	}

	return modifyingCommands[parts[0]]
}

// GetResourcesForExport returns a list of resource types suitable for GitOps export
func GetResourcesForExport() []string {
	return []string{
		"namespaces",
		"deployments",
		"services",
		"configmaps",
		"secrets",
		"ingresses",
		"persistentvolumes",
		"persistentvolumeclaims",
		"serviceaccounts",
		"roles",
		"rolebindings",
		"clusterroles",
		"clusterrolebindings",
		"networkpolicies",
		"poddisruptionbudgets",
		"horizontalpodautoscalers",
		"verticalPodAutoscalers",
		"jobs",
		"cronjobs",
		"daemonsets",
		"statefulsets",
		"replicasets",
	}
}