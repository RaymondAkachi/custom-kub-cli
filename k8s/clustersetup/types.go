// Package k8shard provides automation for setting up Kubernetes clusters.
// types.go defines the data structures and interfaces used throughout the package.
package clustersetup

import (
	"context"
)

// ClusterConfig defines the configuration for the Kubernetes cluster.
type ClusterConfig struct {
	ClusterName       string            `yaml:"cluster_name"`
	KubernetesVersion string            `yaml:"kubernetes_version"`
	EtcdVersion       string            `yaml:"etcd_version"`
	ContainerdVersion string            `yaml:"containerd_version"`
	CNIVersion        string            `yaml:"cni_version"`
	CoreDNSVersion    string            `yaml:"coredns_version"`
	PodCIDR           string            `yaml:"pod_cidr"`
	ServiceCIDR       string            `yaml:"service_cidr"`
	ClusterDNS        string            `yaml:"cluster_dns"`
	WorkDir           string            `yaml:"work_dir"`
	SSHKey            string            `yaml:"ssh_key"`
	SSHUser           string            `yaml:"ssh_user"`
	Controller        Node              `yaml:"controller"`
	Workers           []Node            `yaml:"workers"`
	Certificates      CertificateConfig `yaml:"certificates"`
}

// Node represents a node in the cluster.
type Node struct {
	Name      string `yaml:"name"`
	IPAddress string `yaml:"ip_address"`
	Hostname  string `yaml:"hostname"`
	PodCIDR   string `yaml:"pod_cidr,omitempty"`
}

// CertificateConfig defines certificate generation parameters.
type CertificateConfig struct {
	Country            string `yaml:"country"`
	State              string `yaml:"state"`
	City               string `yaml:"city"`
	Organization       string `yaml:"organization"`
	OrganizationalUnit string `yaml:"organizational_unit"`
	ValidityDays       int    `yaml:"validity_days"`
}

// ClusterStatus holds the status of the cluster.
type ClusterStatus struct {
	Nodes      string
	PodStatus  string
	TestStatus string
}

// SSHClient defines the interface for SSH operations.
type SSHClient interface {
	ExecuteCommand(ctx context.Context, host, command string) (string, error)
	CopyFile(ctx context.Context, host, localPath, remotePath string) error
	CopyContent(ctx context.Context, host, content, remotePath string) error
}

// CertificateManager defines the interface for certificate operations.
type CertificateManager interface {
	GenerateCA(workDir string, config CertificateConfig) error
	GenerateClientCert(workDir, name string, config CertificateConfig) error
	GenerateServerCert(workDir, name string, hosts []string, config CertificateConfig) error
}

// Logger defines the logging interface.
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// ProgressReporter defines the interface for reporting progress.
type ProgressReporter interface {
	ReportProgress(step, totalSteps int, phase string)
	Start(total int, description string)
	Update(current int, status string)
	Finish(success bool, message string)
}

// ClusterManager manages the Kubernetes cluster setup.
type ClusterManager struct {
	config      ClusterConfig
	logger      Logger
	sshClient   SSHClient
	certManager CertificateManager
	progress    ProgressReporter
}

// NewClusterManager creates a new ClusterManager.
func NewClusterManager(config ClusterConfig, logger Logger, sshClient SSHClient, certManager CertificateManager, progress ProgressReporter) *ClusterManager {
	return &ClusterManager{
		config:      config,
		logger:      logger,
		sshClient:   sshClient,
		certManager: certManager,
		progress:    progress,
	}
}