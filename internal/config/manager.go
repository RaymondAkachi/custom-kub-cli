package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

// ClusterInfo holds cluster information
type ClusterInfo struct {
	Name         string    `json:"name"`
	ConfigPath   string    `json:"config_path"`
	Server       string    `json:"server"`
	PublicIP     string    `json:"public_ip"`
	DNS          string    `json:"dns"`
	CreatedAt    time.Time `json:"created_at"`
	HasPrometheus bool     `json:"has_prometheus"`
	HasArgoCD    bool     `json:"has_argocd"`
	GitRepo      string   `json:"git_repo"`
	GitRepoPath  string   `json:"git_repo_path"`
}

// ClusterRegistry manages cluster configurations
type ClusterRegistry struct {
	Clusters []ClusterInfo `json:"clusters"`
}

// KubeConfig represents a simplified kubeconfig structure
type KubeConfig struct {
	Clusters []struct {
		Name    string `yaml:"name"`
		Cluster struct {
			Server string `yaml:"server"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
	Contexts []struct {
		Name    string `yaml:"name"`
		Context struct {
			Cluster string `yaml:"cluster"`
		} `yaml:"context"`
	} `yaml:"contexts"`
	CurrentContext string `yaml:"current-context"`
}

// Manager handles configuration management
type Manager struct {
	ConfigDir    string
	RegistryPath string
	Registry     *ClusterRegistry
}

// Initialize creates and initializes the configuration manager
func Initialize() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %v", err)
	}

	configDir := filepath.Join(homeDir, ".kube-orchestrator", "configs")
	registryPath := filepath.Join(homeDir, ".kube-orchestrator", "registry.json")

	// Create directories
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(registryPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create registry directory: %v", err)
	}

	manager := &Manager{
		ConfigDir:    configDir,
		RegistryPath: registryPath,
		Registry:     &ClusterRegistry{},
	}

	if err := manager.LoadRegistry(); err != nil {
		return nil, fmt.Errorf("failed to load registry: %v", err)
	}

	return manager, nil
}

// LoadRegistry loads the cluster registry from disk
func (m *Manager) LoadRegistry() error {
	if _, err := os.Stat(m.RegistryPath); os.IsNotExist(err) {
		// Create empty registry if it doesn't exist
		return m.SaveRegistry()
	}

	data, err := ioutil.ReadFile(m.RegistryPath)
	if err != nil {
		return fmt.Errorf("failed to read registry file: %v", err)
	}

	if err := json.Unmarshal(data, m.Registry); err != nil {
		return fmt.Errorf("failed to parse registry file: %v", err)
	}

	return nil
}

// SaveRegistry saves the cluster registry to disk
func (m *Manager) SaveRegistry() error {
	data, err := json.MarshalIndent(m.Registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %v", err)
	}

	if err := ioutil.WriteFile(m.RegistryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %v", err)
	}

	return nil
}

// AddCluster adds a new cluster to the registry
func (m *Manager) AddCluster(cluster ClusterInfo) error {
	// Check if cluster already exists
	for _, existing := range m.Registry.Clusters {
		if existing.Name == cluster.Name {
			return fmt.Errorf("cluster with name '%s' already exists", cluster.Name)
		}
	}

	m.Registry.Clusters = append(m.Registry.Clusters, cluster)
	return m.SaveRegistry()
}

// UpdateCluster updates an existing cluster in the registry
func (m *Manager) UpdateCluster(cluster ClusterInfo) error {
	for i, existing := range m.Registry.Clusters {
		if existing.Name == cluster.Name {
			m.Registry.Clusters[i] = cluster
			return m.SaveRegistry()
		}
	}
	return fmt.Errorf("cluster with name '%s' not found", cluster.Name)
}

// RemoveCluster removes a cluster from the registry
func (m *Manager) RemoveCluster(name string) error {
	for i, cluster := range m.Registry.Clusters {
		if cluster.Name == name {
			// Remove config file
			if err := os.Remove(cluster.ConfigPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove config file: %v", err)
			}

			// Remove from registry
			m.Registry.Clusters = append(m.Registry.Clusters[:i], m.Registry.Clusters[i+1:]...)
			return m.SaveRegistry()
		}
	}
	return fmt.Errorf("cluster with name '%s' not found", name)
}

// GetCluster retrieves a cluster by name
func (m *Manager) GetCluster(name string) (*ClusterInfo, error) {
	for _, cluster := range m.Registry.Clusters {
		if cluster.Name == name {
			return &cluster, nil
		}
	}
	return nil, fmt.Errorf("cluster with name '%s' not found", name)
}

// GetAllClusters returns all registered clusters
func (m *Manager) GetAllClusters() []ClusterInfo {
	return m.Registry.Clusters
}

// ParseKubeConfig parses a kubeconfig file and extracts cluster information
func (m *Manager) ParseKubeConfig(configPath string) (*KubeConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig file: %v", err)
	}

	var config KubeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig file: %v", err)
	}

	return &config, nil
}

// CopyKubeConfig copies a kubeconfig file to the managed directory
func (m *Manager) CopyKubeConfig(srcPath, clusterName string) (string, error) {
	destPath := filepath.Join(m.ConfigDir, fmt.Sprintf("%s.yaml", clusterName))
	
	data, err := ioutil.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to read source kubeconfig: %v", err)
	}

	if err := ioutil.WriteFile(destPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write kubeconfig to managed directory: %v", err)
	}

	return destPath, nil
}

// ValidateClusterConfig validates cluster configuration
func (m *Manager) ValidateClusterConfig(cluster *ClusterInfo) error {
	if cluster.Name == "" {
		return fmt.Errorf("cluster name cannot be empty")
	}

	if cluster.ConfigPath == "" {
		return fmt.Errorf("kubeconfig path cannot be empty")
	}

	if _, err := os.Stat(cluster.ConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("kubeconfig file does not exist: %s", cluster.ConfigPath)
	}

	// Validate kubeconfig format
	_, err := m.ParseKubeConfig(cluster.ConfigPath)
	if err != nil {
		return fmt.Errorf("invalid kubeconfig file: %v", err)
	}

	return nil
}