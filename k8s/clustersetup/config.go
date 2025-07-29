// Package k8shard provides automation for setting up Kubernetes clusters.
// config.go handles loading and validating the cluster configuration.
package clustersetup

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadClusterConfig loads the cluster configuration from a YAML file.
func LoadClusterConfig(configPath string) (ClusterConfig, error) {
	var config ClusterConfig
	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if config.ClusterName == "" {
		return config, fmt.Errorf("cluster_name is required")
	}
	if config.KubernetesVersion == "" {
		return config, fmt.Errorf("kubernetes_version is required")
	}
	if config.EtcdVersion == "" {
		return config, fmt.Errorf("etcd_version is required")
	}
	if config.ContainerdVersion == "" {
		return config, fmt.Errorf("containerd_version is required")
	}
	if config.CNIVersion == "" {
		return config, fmt.Errorf("cni_version is required")
	}
	if config.CoreDNSVersion == "" {
		return config, fmt.Errorf("coredns_version is required")
	}
	if config.PodCIDR == "" {
		return config, fmt.Errorf("pod_cidr is required")
	}
	if config.ServiceCIDR == "" {
		return config, fmt.Errorf("service_cidr is required")
	}
	if config.ClusterDNS == "" {
		return config, fmt.Errorf("cluster_dns is required")
	}
	if config.WorkDir == "" {
		return config, fmt.Errorf("work_dir is required")
	}
	if config.SSHKey == "" {
		return config, fmt.Errorf("ssh_key is required")
	}
	if config.SSHUser == "" {
		return config, fmt.Errorf("ssh_user is required")
	}
	if config.Controller.IPAddress == "" || config.Controller.Name == "" {
		return config, fmt.Errorf("controller configuration is incomplete")
	}
	if len(config.Workers) == 0 {
		return config, fmt.Errorf("at least one worker node is required")
	}
	for _, worker := range config.Workers {
		if worker.IPAddress == "" || worker.Name == "" || worker.PodCIDR == "" {
			return config, fmt.Errorf("worker %s configuration is incomplete", worker.Name)
		}
	}
	if config.Certificates.Country == "" || config.Certificates.ValidityDays <= 0 {
		return config, fmt.Errorf("certificate configuration is incomplete")
	}

	// Ensure WorkDir exists
	if err := os.MkdirAll(config.WorkDir, 0755); err != nil {
		return config, fmt.Errorf("failed to create work directory %s: %w", config.WorkDir, err)
	}

	return config, nil
}

// GenerateDefaultConfig generates a default cluster configuration.
func GenerateDefaultConfig() ClusterConfig {
	return ClusterConfig{
		ClusterName:       "my-cluster",
		KubernetesVersion: "v1.26.0",
		EtcdVersion:       "v3.5.9",
		ContainerdVersion:  "v1.7.2",
		CNIVersion:        "v1.3.0",
		CoreDNSVersion:    "1.10.1",
		PodCIDR:           "10.200.0.0/16",
		ServiceCIDR:       "10.32.0.0/24",
		ClusterDNS:        "10.32.0.10",
		WorkDir:           "/tmp/k8s-hard-way",
		SSHKey:            "~/.ssh/id_rsa",
		SSHUser:           "ubuntu",
		Controller: Node{
			Name:      "controller-0",
			IPAddress: "10.240.0.10",
			Hostname:  "controller-0",
		},
		Workers: []Node{
			{
				Name:      "worker-0",
				IPAddress: "10.240.0.20",
				Hostname:  "worker-0",
				PodCIDR:   "10.200.0.0/24",
			},
			{
				Name:      "worker-1",
				IPAddress: "10.240.0.21",
				Hostname:  "worker-1",
				PodCIDR:   "10.200.1.0/24",
			},
			{
				Name:      "worker-2",
				IPAddress: "10.240.0.22",
				Hostname:  "worker-2",
				PodCIDR:   "10.200.2.0/24",
			},
		},
		Certificates: CertificateConfig{
			Country:            "US",
			State:              "California",
			City:               "San Francisco",
			Organization:       "ExampleOrg",
			OrganizationalUnit: "IT",
			ValidityDays:       365,
		},
	}
}

// SaveConfig saves the configuration to a YAML file.
func SaveConfig(config ClusterConfig, outputPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", outputPath, err)
	}
	return nil
}