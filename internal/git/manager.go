package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/RaymondAkachi/custom-kub-cli/internal/config"
	"github.com/RaymondAkachi/custom-kub-cli/internal/kubectl"
)

// Manager handles Git operations for GitOps workflow
type Manager struct {
	cluster     *config.ClusterInfo
	repoPath    string
	clusterPath string
	executor    *kubectl.Executor
}

// NewManager creates a new Git manager for a cluster
func NewManager(cluster *config.ClusterInfo, executor *kubectl.Executor) (*Manager, error) {
	if cluster.GitRepo == "" {
		return nil, fmt.Errorf("git repository not configured for cluster %s", cluster.Name)
	}

	repoPath := cluster.GitRepoPath
	if repoPath == "" {
		// Default to temp directory if not specified
		repoPath = filepath.Join(os.TempDir(), fmt.Sprintf("k8s-configs-%s", cluster.Name))
	}

	clusterPath := filepath.Join(repoPath, cluster.Name)

	manager := &Manager{
		cluster:     cluster,
		repoPath:    repoPath,
		clusterPath: clusterPath,
		executor:    executor,
	}

	return manager, nil
}

// Initialize sets up the Git repository (clone if needed)
func (gm *Manager) Initialize() error {
	// Check if repository already exists
	if _, err := os.Stat(filepath.Join(gm.repoPath, ".git")); err == nil {
		// Repository exists, just pull latest changes
		return gm.pullLatest()
	}

	// Clone the repository
	return gm.cloneRepository()
}

// cloneRepository clones the Git repository
func (gm *Manager) cloneRepository() error {
	// Remove existing directory if it exists but is not a git repo
	if _, err := os.Stat(gm.repoPath); err == nil {
		if err := os.RemoveAll(gm.repoPath); err != nil {
			return fmt.Errorf("failed to remove existing directory: %v", err)
		}
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(gm.repoPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %v", err)
	}

	// Clone repository
	cmd := exec.Command("git", "clone", gm.cluster.GitRepo, gm.repoPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone repository: %v\nOutput: %s", err, string(output))
	}

	return nil
}

// pullLatest pulls the latest changes from the remote repository
func (gm *Manager) pullLatest() error {
	cmd := exec.Command("git", "-C", gm.repoPath, "pull", "origin", "main")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try master branch if main fails
		cmd = exec.Command("git", "-C", gm.repoPath, "pull", "origin", "master")
		if output2, err2 := cmd.CombinedOutput(); err2 != nil {
			return fmt.Errorf("failed to pull from both main and master branches:\nMain: %v (%s)\nMaster: %v (%s)", 
				err, string(output), err2, string(output2))
		}
	}

	return nil
}

// ExportClusterResources exports current cluster resources to the Git repository
func (gm *Manager) ExportClusterResources() error {
	// Ensure cluster directory exists
	if err := os.MkdirAll(gm.clusterPath, 0755); err != nil {
		return fmt.Errorf("failed to create cluster directory: %v", err)
	}

	// Export each resource type
	resources := kubectl.GetResourcesForExport()
	exportedCount := 0

	for _, resource := range resources {
		if err := gm.exportResource(resource); err != nil {
			// Log warning but continue with other resources
			fmt.Printf("Warning: Failed to export %s: %v\n", resource, err)
			continue
		}
		exportedCount++
	}

	if exportedCount == 0 {
		return fmt.Errorf("no resources were successfully exported")
	}

	return nil
}

// exportResource exports a specific resource type
func (gm *Manager) exportResource(resourceType string) error {
	output, err := gm.executor.Execute("get", resourceType, "--all-namespaces", "-o", "yaml")
	if err != nil {
		return fmt.Errorf("failed to get %s: %v", resourceType, err)
	}

	// Skip if no resources found
	if strings.Contains(output, "No resources found") {
		return nil
	}

	// Write to file
	resourceFile := filepath.Join(gm.clusterPath, fmt.Sprintf("%s.yaml", resourceType))
	if err := ioutil.WriteFile(resourceFile, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write %s file: %v", resourceType, err)
	}

	return nil
}

// CommitAndPush commits changes and pushes to the remote repository
func (gm *Manager) CommitAndPush(message string) error {
	// Change to repository directory
	oldDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(gm.repoPath); err != nil {
		return fmt.Errorf("failed to change to repository directory: %v", err)
	}

	// Add all changes
	if err := gm.runGitCommand("add", "."); err != nil {
		return fmt.Errorf("failed to add changes: %v", err)
	}

	// Check if there are changes to commit
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	if err := cmd.Run(); err == nil {
		// No changes to commit
		return nil
	}

	// Commit changes
	if message == "" {
		message = fmt.Sprintf("Update %s cluster resources - %s", 
			gm.cluster.Name, time.Now().Format("2006-01-02 15:04:05"))
	}

	if err := gm.runGitCommand("commit", "-m", message); err != nil {
		return fmt.Errorf("failed to commit changes: %v", err)
	}

	// Push changes
	if err := gm.runGitCommand("push"); err != nil {
		return fmt.Errorf("failed to push changes: %v", err)
	}

	return nil
}

// runGitCommand runs a git command in the repository directory
func (gm *Manager) runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = gm.repoPath
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s failed: %v\nOutput: %s", 
			strings.Join(args, " "), err, string(output))
	}

	return nil
}

// SyncChanges performs a complete sync operation (export + commit + push)
func (gm *Manager) SyncChanges(commitMessage string) error {
	// Pull latest changes first
	if err := gm.pullLatest(); err != nil {
		return fmt.Errorf("failed to pull latest changes: %v", err)
	}

	// Export current cluster resources
	if err := gm.ExportClusterResources(); err != nil {
		return fmt.Errorf("failed to export cluster resources: %v", err)
	}

	// Commit and push changes
	if err := gm.CommitAndPush(commitMessage); err != nil {
		return fmt.Errorf("failed to commit and push changes: %v", err)
	}

	return nil
}

// GetRepositoryStatus returns the current Git repository status
func (gm *Manager) GetRepositoryStatus() (string, error) {
	cmd := exec.Command("git", "-C", gm.repoPath, "status", "--porcelain")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get repository status: %v", err)
	}

	return string(output), nil
}

// GetLastCommit returns information about the last commit
func (gm *Manager) GetLastCommit() (string, error) {
	cmd := exec.Command("git", "-C", gm.repoPath, "log", "-1", "--pretty=format:%h - %s (%cr) <%an>")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get last commit: %v", err)
	}

	return string(output), nil
}

// ValidateRepository checks if the repository is properly configured
func (gm *Manager) ValidateRepository() error {
	// Check if repository directory exists
	if _, err := os.Stat(gm.repoPath); os.IsNotExist(err) {
		return fmt.Errorf("repository directory does not exist: %s", gm.repoPath)
	}

	// Check if it's a valid git repository
	if _, err := os.Stat(filepath.Join(gm.repoPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", gm.repoPath)
	}

	// Check if remote origin is configured correctly
	cmd := exec.Command("git", "-C", gm.repoPath, "remote", "get-url", "origin")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get remote origin URL: %v", err)
	}

	remoteURL := strings.TrimSpace(string(output))
	if remoteURL != gm.cluster.GitRepo {
		return fmt.Errorf("remote origin URL mismatch: expected %s, got %s", gm.cluster.GitRepo, remoteURL)
	}

	return nil
}