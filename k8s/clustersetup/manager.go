// Package k8shard provides automation for setting up Kubernetes clusters.
// manager.go orchestrates the cluster setup and destruction.
package clustersetup

import (
	"context"
	"fmt"
	"os"
)

// SetupCluster sets up the Kubernetes cluster.
func (cm *ClusterManager) SetupCluster(ctx context.Context) error {
	totalSteps := 6
	cm.progress.ReportProgress(1, totalSteps, "Checking Prerequisites")
	if err := cm.ValidateK8sPrerequisites(); err != nil {
		return fmt.Errorf("prerequisites check failed: %w", err)
	}

	workDir := cm.config.WorkDir
	cm.progress.ReportProgress(2, totalSteps, "Generating Certificates")
	if err := cm.generateCertificates(ctx, workDir); err != nil {
		return fmt.Errorf("failed to generate certificates: %w", err)
	}

	cm.progress.ReportProgress(3, totalSteps, "Creating Configurations")
	if err := cm.createConfigurations(ctx, workDir); err != nil {
		return fmt.Errorf("failed to create configurations: %w", err)
	}

	cm.progress.ReportProgress(4, totalSteps, "Setting Up Control Plane")
	if err := cm.setupControlPlane(ctx, workDir); err != nil {
		return fmt.Errorf("failed to setup control plane: %w", err)
	}

	cm.progress.ReportProgress(5, totalSteps, "Setting Up Worker Nodes")
	if err := cm.setupWorkerNodes(ctx, workDir); err != nil {
		return fmt.Errorf("failed to setup worker nodes: %w", err)
	}

	cm.progress.ReportProgress(6, totalSteps, "Setting Up Networking")
	if err := cm.setupNetworking(ctx); err != nil {
		return fmt.Errorf("failed to setup networking: %w", err)
	}

	cm.progress.ReportProgress(6, totalSteps, "Validating Cluster")
	if err := cm.validateCluster(ctx); err != nil {
		return fmt.Errorf("failed to validate cluster: %w", err)
	}

	return nil
}

// ValidateK8sPrerequisites checks SSH connectivity and working directory.
func (cm *ClusterManager) ValidateK8sPrerequisites() error {
	cm.logger.Info("Checking prerequisites...")

	nodes := append([]Node{cm.config.Controller}, cm.config.Workers...)
	for _, node := range nodes {
		if _, err := cm.sshClient.ExecuteCommand(context.Background(), node.IPAddress, "echo 'SSH test'"); err != nil {
			return fmt.Errorf("SSH connection to %s failed: %w", node.Name, err)
		}
		cm.logger.Info(fmt.Sprintf("SSH connection verified: %s", node.Name))
	}

	if err := os.MkdirAll(cm.config.WorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory %s: %w", cm.config.WorkDir, err)
	}

	return nil
}

// GetClusterStatus retrieves the current cluster status.
func (cm *ClusterManager) GetClusterStatus(ctx context.Context) (ClusterStatus, error) {
	var status ClusterStatus
	controller := cm.config.Controller

	nodeStatus, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress,
		"kubectl get nodes --kubeconfig /var/lib/kubernetes/admin.kubeconfig")
	if err != nil {
		return status, fmt.Errorf("failed to get node status: %w", err)
	}
	status.Nodes = nodeStatus

	podStatus, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress,
		"kubectl get pods -n kube-system --kubeconfig /var/lib/kubernetes/admin.kubeconfig")
	if err != nil {
		return status, fmt.Errorf("failed to get system pods: %w", err)
	}
	status.PodStatus = podStatus

	testStatus, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress,
		"kubectl get deployment test-deployment --kubeconfig /var/lib/kubernetes/admin.kubeconfig")
	if err != nil {
		return status, fmt.Errorf("failed to get test app status: %w", err)
	}
	status.TestStatus = testStatus

	return status, nil
}

// DestroyCluster cleans up the cluster.
func (cm *ClusterManager) DestroyCluster(ctx context.Context) error {
	cm.logger.Info("Destroying cluster...")

	nodes := append([]Node{cm.config.Controller}, cm.config.Workers...)
	for _, node := range nodes {
		cm.logger.Info(fmt.Sprintf("Cleaning up node: %s", node.Name))
		commands := []string{
			"sudo systemctl stop etcd kube-apiserver kube-controller-manager kube-scheduler containerd kubelet kube-proxy || true",
			"sudo systemctl disable etcd kube-apiserver kube-controller-manager kube-scheduler containerd kubelet kube-proxy || true",
			"sudo rm -rf /etc/etcd /var/lib/etcd /etc/kubernetes /var/lib/kubernetes /var/lib/kubelet /var/lib/kube-proxy /etc/cni /opt/cni /var/run/kubernetes",
			"sudo rm -f /usr/local/bin/etcd* /usr/local/bin/kube* /usr/local/bin/runc /bin/containerd*",
			"sudo rm -f /etc/systemd/system/etcd.service /etc/systemd/system/kube*.service /etc/systemd/system/containerd.service",
			"sudo systemctl daemon-reload",
			"sudo systemctl reset-failed",
		}
		for _, cmd := range commands {
			if _, err := cm.sshClient.ExecuteCommand(ctx, node.IPAddress, cmd); err != nil {
				return fmt.Errorf("failed to execute cleanup command '%s' on %s: %w", cmd, node.Name, err)
			}
		}
	}

	if err := os.RemoveAll(cm.config.WorkDir); err != nil {
		return fmt.Errorf("failed to remove work directory %s: %w", cm.config.WorkDir, err)
	}

	cm.logger.Info("Cluster destroyed successfully")
	return nil
}