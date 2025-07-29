// Package k8shard provides automation for setting up Kubernetes clusters.
// setup.go contains the core logic for setting up the control plane, worker nodes, networking, and validating the cluster.
package clustersetup

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// generateCertificates generates all required certificates for the Kubernetes cluster.
func (cm *ClusterManager) generateCertificates(ctx context.Context, workDir string) error {
	cm.logger.Info("Generating certificates...")
	if err := cm.certManager.GenerateCA(workDir, cm.config.Certificates); err != nil {
		return fmt.Errorf("failed to generate CA: %w", err)
	}

	clientCerts := []string{
		"admin",
		"kube-controller-manager",
		"kube-proxy",
		"kube-scheduler",
		"service-account",
	}
	for _, worker := range cm.config.Workers {
		clientCerts = append(clientCerts, worker.Name)
	}
	for _, name := range clientCerts {
		if err := cm.certManager.GenerateClientCert(workDir, name, cm.config.Certificates); err != nil {
			return fmt.Errorf("failed to generate client certificate for %s: %w", name, err)
		}
	}

	serverHosts := []string{
		"127.0.0.1",
		"10.32.0.1",
		cm.config.Controller.IPAddress,
		cm.config.Controller.Hostname,
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		"kubernetes.default.svc.cluster",
		"kubernetes.default.svc.cluster.local",
	}
	if err := cm.certManager.GenerateServerCert(workDir, "kubernetes", serverHosts, cm.config.Certificates); err != nil {
		return fmt.Errorf("failed to generate server certificate: %w", err)
	}

	cm.logger.Info("All certificates generated successfully")
	return nil
}

// createConfigurations generates all required configuration files for the cluster.
func (cm *ClusterManager) createConfigurations(ctx context.Context, workDir string) error {
	cm.logger.Info("Creating configurations...")
	if err := cm.generateEncryptionConfig(workDir); err != nil {
		return fmt.Errorf("failed to create encryption config: %w", err)
	}

	for _, worker := range cm.config.Workers {
		if err := cm.generateKubeconfig(workDir, worker.Name, worker.IPAddress); err != nil {
			return fmt.Errorf("failed to generate kubeconfig for %s: %w", worker.Name, err)
		}
	}

	for _, name := range []string{"kube-proxy", "kube-controller-manager", "kube-scheduler", "admin"} {
		var ip string
		if name == "kube-controller-manager" || name == "kube-scheduler" || name == "admin" {
			ip = cm.config.Controller.IPAddress
		} else {
			ip = ""
		}
		if err := cm.generateKubeconfig(workDir, name, ip); err != nil {
			return fmt.Errorf("failed to generate kubeconfig for %s: %w", name, err)
		}
	}

	cm.logger.Info("All configurations created successfully")
	return nil
}

// setupControlPlane sets up the Kubernetes control plane on the controller node.
func (cm *ClusterManager) setupControlPlane(ctx context.Context, workDir string) error {
	cm.logger.Info("Setting up control plane...")
	controller := cm.config.Controller

	// Setup etcd
	etcdCommands := []string{
		"sudo mkdir -p /etc/etcd /var/lib/etcd",
		"sudo groupadd -f etcd",
		"sudo useradd -g etcd -d /var/lib/etcd -s /sbin/nologin -c 'etcd user' etcd || true",
		"sudo chown -R etcd:etcd /var/lib/etcd",
		fmt.Sprintf("wget -q --show-progress --https-only --timestamping 'https://github.com/etcd-io/etcd/releases/download/%s/etcd-%s-linux-amd64.tar.gz'", cm.config.EtcdVersion, cm.config.EtcdVersion),
		fmt.Sprintf("tar -xzf etcd-%s-linux-amd64.tar.gz", cm.config.EtcdVersion),
		fmt.Sprintf("sudo mv etcd-%s-linux-amd64/etcd* /usr/local/bin/", cm.config.EtcdVersion),
		fmt.Sprintf("rm -f etcd-%s-linux-amd64.tar.gz", cm.config.EtcdVersion),
	}
	for _, cmd := range etcdCommands {
		if _, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress, cmd); err != nil {
			return fmt.Errorf("failed to execute etcd setup command '%s' on controller: %w", cmd, err)
		}
	}

	// Copy etcd certificates
	certFiles := []string{"ca.pem", "kubernetes-key.pem", "kubernetes.pem"}
	for _, file := range certFiles {
		localPath := filepath.Join(workDir, file)
		remotePath := "/etc/etcd/" + file
		if err := cm.sshClient.CopyFile(ctx, controller.IPAddress, localPath, remotePath); err != nil {
			return fmt.Errorf("failed to copy %s to controller: %w", file, err)
		}
		if _, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress, fmt.Sprintf("sudo chown etcd:etcd %s", remotePath)); err != nil {
			return fmt.Errorf("failed to set ownership for %s: %w", remotePath, err)
		}
	}

	// Setup etcd service
	etcdService := cm.generateEtcdService(controller)
	if err := cm.sshClient.CopyContent(ctx, controller.IPAddress, etcdService, "/etc/systemd/system/etcd.service"); err != nil {
		return fmt.Errorf("failed to upload etcd service: %w", err)
	}

	// Setup Kubernetes control plane components
	k8sCommands := []string{
		"sudo mkdir -p /etc/kubernetes/config /var/lib/kubernetes",
		fmt.Sprintf("wget -q --show-progress --https-only --timestamping "+
			"'https://storage.googleapis.com/kubernetes-release/release/%s/bin/linux/amd64/kube-apiserver' "+
			"'https://storage.googleapis.com/kubernetes-release/release/%s/bin/linux/amd64/kube-controller-manager' "+
			"'https://storage.googleapis.com/kubernetes-release/release/%s/bin/linux/amd64/kube-scheduler' "+
			"'https://storage.googleapis.com/kubernetes-release/release/%s/bin/linux/amd64/kubectl'",
			cm.config.KubernetesVersion, cm.config.KubernetesVersion, cm.config.KubernetesVersion, cm.config.KubernetesVersion),
		"chmod +x kube-apiserver kube-controller-manager kube-scheduler kubectl",
		"sudo mv kube-apiserver kube-controller-manager kube-scheduler kubectl /usr/local/bin/",
	}
	for _, cmd := range k8sCommands {
		if _, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress, cmd); err != nil {
			return fmt.Errorf("failed to execute kubernetes setup command '%s' on controller: %w", cmd, err)
		}
	}

	// Copy additional Kubernetes files
	additionalFiles := []string{
		"ca-key.pem", "service-account-key.pem", "service-account.pem",
		"encryption-config.yaml", "kube-controller-manager.kubeconfig", "kube-scheduler.kubeconfig",
	}
	for _, file := range additionalFiles {
		localPath := filepath.Join(workDir, file)
		remotePath := "/var/lib/kubernetes/" + file
		if err := cm.sshClient.CopyFile(ctx, controller.IPAddress, localPath, remotePath); err != nil {
			return fmt.Errorf("failed to copy %s to controller: %w", file, err)
		}
	}

	// Setup Kubernetes services
	services := map[string]string{
		"kube-apiserver":         cm.generateAPIServerService(),
		"kube-controller-manager": cm.generateControllerManagerService(),
		"kube-scheduler":         cm.generateSchedulerService(),
	}
	for name, content := range services {
		servicePath := "/etc/systemd/system/" + name + ".service"
		if err := cm.sshClient.CopyContent(ctx, controller.IPAddress, content, servicePath); err != nil {
			return fmt.Errorf("failed to upload %s service: %w", name, err)
		}
	}

	// Start services in proper order with health checks
	// Start etcd first
	if _, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress, 
		"sudo systemctl daemon-reload && sudo systemctl enable etcd && sudo systemctl start etcd"); err != nil {
		return fmt.Errorf("failed to start etcd: %w", err)
	}

	// Wait for etcd to be healthy
	if err := cm.waitForService(ctx, controller.IPAddress, "etcd", 30*time.Second); err != nil {
		return fmt.Errorf("etcd failed to become healthy: %w", err)
	}

	// Start API server
	if _, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress,
		"sudo systemctl enable kube-apiserver && sudo systemctl start kube-apiserver"); err != nil {
		return fmt.Errorf("failed to start kube-apiserver: %w", err)
	}

	// Wait for API server
	if err := cm.waitForService(ctx, controller.IPAddress, "kube-apiserver", 60*time.Second); err != nil {
		return fmt.Errorf("kube-apiserver failed to become healthy: %w", err)
	}

	// Start controller manager and scheduler
	last_services := []string{"kube-controller-manager", "kube-scheduler"}
	for _, service := range last_services {
		if _, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress,
			fmt.Sprintf("sudo systemctl enable %s && sudo systemctl start %s", service, service)); err != nil {
			return fmt.Errorf("failed to start %s: %w", service, err)
		}
		if err := cm.waitForService(ctx, controller.IPAddress, service, 30*time.Second); err != nil {
			return fmt.Errorf("%s failed to become healthy: %w", service, err)
		}
	}

	cm.logger.Info("Control plane setup completed")
	return nil
}

// setupSingleWorkerNode sets up a single worker node.
func (cm *ClusterManager) setupSingleWorkerNode(ctx context.Context, workDir string, worker Node) error {
	cm.logger.Info(fmt.Sprintf("Setting up worker node %s...", worker.Name))

	// Install dependencies
	depCommands := []string{
		"sudo apt-get update",
		"sudo apt-get -y install socat conntrack ipset",
		"sudo mkdir -p /etc/cni/net.d /opt/cni/bin /var/lib/kubelet /var/lib/kube-proxy /var/lib/kubernetes /var/run/kubernetes",
	}
	for _, cmd := range depCommands {
		if _, err := cm.sshClient.ExecuteCommand(ctx, worker.IPAddress, cmd); err != nil {
			return fmt.Errorf("failed to execute dependency command '%s' on %s: %w", cmd, worker.Name, err)
		}
	}

	// Install CNI plugins
	cniCommands := []string{
		fmt.Sprintf("wget -q --show-progress --https-only --timestamping 'https://github.com/containernetworking/plugins/releases/download/%s/cni-plugins-linux-amd64-%s.tgz'", cm.config.CNIVersion, cm.config.CNIVersion),
		fmt.Sprintf("sudo tar -xzf cni-plugins-linux-amd64-%s.tgz -C /opt/cni/bin/", cm.config.CNIVersion),
		fmt.Sprintf("rm -f cni-plugins-linux-amd64-%s.tgz", cm.config.CNIVersion),
	}
	for _, cmd := range cniCommands {
		if _, err := cm.sshClient.ExecuteCommand(ctx, worker.IPAddress, cmd); err != nil {
			return fmt.Errorf("failed to execute CNI command '%s' on %s: %w", cmd, worker.Name, err)
		}
	}

	// Install containerd
	containerdCommands := []string{
		fmt.Sprintf("wget -q --show-progress --https-only --timestamping 'https://github.com/containerd/containerd/releases/download/%s/containerd-%s-linux-amd64.tar.gz'", cm.config.ContainerdVersion, cm.config.ContainerdVersion),
		"wget -q --show-progress --https-only --timestamping 'https://github.com/opencontainers/runc/releases/download/v1.1.7/runc.amd64'",
		fmt.Sprintf("sudo tar -xzf containerd-%s-linux-amd64.tar.gz -C /", cm.config.ContainerdVersion),
		"sudo mv runc.amd64 runc",
		"chmod +x runc",
		"sudo mv runc /usr/local/bin/",
		fmt.Sprintf("rm -f containerd-%s-linux-amd64.tar.gz", cm.config.ContainerdVersion),
	}
	for _, cmd := range containerdCommands {
		if _, err := cm.sshClient.ExecuteCommand(ctx, worker.IPAddress, cmd); err != nil {
			return fmt.Errorf("failed to execute containerd command '%s' on %s: %w", cmd, worker.Name, err)
		}
	}

	// Install Kubernetes binaries
	k8sWorkerCommands := []string{
		fmt.Sprintf("wget -q --show-progress --https-only --timestamping "+
			"'https://storage.googleapis.com/kubernetes-release/release/%s/bin/linux/amd64/kubectl' "+
			"'https://storage.googleapis.com/kubernetes-release/release/%s/bin/linux/amd64/kube-proxy' "+
			"'https://storage.googleapis.com/kubernetes-release/release/%s/bin/linux/amd64/kubelet'",
			cm.config.KubernetesVersion, cm.config.KubernetesVersion, cm.config.KubernetesVersion),
		"chmod +x kubectl kube-proxy kubelet",
		"sudo mv kubectl kube-proxy kubelet /usr/local/bin/",
	}
	for _, cmd := range k8sWorkerCommands {
		if _, err := cm.sshClient.ExecuteCommand(ctx, worker.IPAddress, cmd); err != nil {
			return fmt.Errorf("failed to execute kubernetes command '%s' on %s: %w", cmd, worker.Name, err)
		}
	}

	// Copy certificates and kubeconfigs
	workerFiles := []string{
		"ca.pem",
		worker.Name + "-key.pem",
		worker.Name + ".pem",
		worker.Name + ".kubeconfig",
		"kube-proxy.kubeconfig",
	}
	for _, file := range workerFiles {
		localPath := filepath.Join(workDir, file)
		remotePath := "/var/lib/kubelet/" + file
		if file == "kube-proxy.kubeconfig" {
			remotePath = "/var/lib/kube-proxy/" + file
		}
		if err := cm.sshClient.CopyFile(ctx, worker.IPAddress, localPath, remotePath); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", file, worker.Name, err)
		}
	}

	// Copy configuration files
	configs := map[string]string{
		"/etc/containerd/config.toml":            cm.generateContainerdConfig(),
		"/etc/cni/net.d/10-bridge.conf":          cm.generateBridgeNetworkConfig(worker.PodCIDR),
		"/etc/cni/net.d/99-loopback.conf":        cm.generateLoopbackNetworkConfig(),
		"/var/lib/kubelet/kubelet-config.yaml":   cm.generateKubeletConfig(worker),
		"/var/lib/kube-proxy/kube-proxy-config.yaml": cm.generateKubeProxyConfig(),
	}
	for path, content := range configs {
		if err := cm.sshClient.CopyContent(ctx, worker.IPAddress, content, path); err != nil {
			return fmt.Errorf("failed to upload config %s to %s: %w", path, worker.Name, err)
		}
	}

	// Setup services
	services := map[string]string{
		"containerd": cm.generateContainerdService(),
		"kubelet":    cm.generateKubeletService(worker),
		"kube-proxy": cm.generateKubeProxyService(),
	}
	for name, content := range services {
		servicePath := "/etc/systemd/system/" + name + ".service"
		if err := cm.sshClient.CopyContent(ctx, worker.IPAddress, content, servicePath); err != nil {
			return fmt.Errorf("failed to upload %s service to %s: %w", name, worker.Name, err)
		}
	}

	// Start services
	workerStartCommands := []string{
		"sudo systemctl daemon-reload",
		"sudo systemctl enable containerd kubelet kube-proxy",
		"sudo systemctl start containerd kubelet kube-proxy",
	}
	for _, cmd := range workerStartCommands {
		if _, err := cm.sshClient.ExecuteCommand(ctx, worker.IPAddress, cmd); err != nil {
			return fmt.Errorf("failed to execute start command '%s' on %s: %w", cmd, worker.Name, err)
		}
	}

	cm.logger.Info(fmt.Sprintf("Worker node %s setup completed", worker.Name))
	return nil
}

// setupWorkerNodes sets up all worker nodes.
func (cm *ClusterManager) setupWorkerNodes(ctx context.Context, workDir string) error {
	cm.logger.Info("Setting up worker nodes...")
	for _, worker := range cm.config.Workers {
		if err := cm.setupSingleWorkerNode(ctx, workDir, worker); err != nil {
			return fmt.Errorf("failed to setup worker %s: %w", worker.Name, err)
		}
	}
	cm.logger.Info("All worker nodes setup completed")
	return nil
}

// setupNetworking configures pod networking and deploys CoreDNS.
func (cm *ClusterManager) setupNetworking(ctx context.Context) error {
	cm.logger.Info("Setting up networking...")
	controller := cm.config.Controller

	// Setup pod routing
	for _, worker := range cm.config.Workers {
		for _, otherWorker := range cm.config.Workers {
			if worker.Name != otherWorker.Name {
				routeCmd := fmt.Sprintf("sudo ip route add %s via %s || true", otherWorker.PodCIDR, otherWorker.IPAddress)
				if _, err := cm.sshClient.ExecuteCommand(ctx, worker.IPAddress, routeCmd); err != nil {
					return fmt.Errorf("failed to add route on %s: %w", worker.Name, err)
				}
			}
		}
	}

	// Deploy CoreDNS
	coreDNSManifest := cm.generateCoreDNSManifest()
	manifestPath := "/tmp/coredns.yaml"
	if err := cm.sshClient.CopyContent(ctx, controller.IPAddress, coreDNSManifest, manifestPath); err != nil {
		return fmt.Errorf("failed to upload CoreDNS manifest: %w", err)
	}
	applyCmd := fmt.Sprintf("kubectl apply -f %s --kubeconfig /var/lib/kubernetes/admin.kubeconfig", manifestPath)
	if _, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress, applyCmd); err != nil {
		return fmt.Errorf("failed to apply CoreDNS manifest: %w", err)
	}

	cm.logger.Info("Networking setup completed")
	return nil
}

// validateCluster validates the cluster setup.
func (cm *ClusterManager) validateCluster(ctx context.Context) error {
	cm.logger.Info("Validating cluster...")
	controller := cm.config.Controller

	nodeStatus, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress,
		"kubectl get nodes --kubeconfig /var/lib/kubernetes/admin.kubeconfig")
	if err != nil {
		return fmt.Errorf("failed to get node status: %w", err)
	}
	cm.logger.Info(fmt.Sprintf("Node status: %s", nodeStatus))

	podStatus, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress,
		"kubectl get pods -n kube-system --kubeconfig /var/lib/kubernetes/admin.kubeconfig")
	if err != nil {
		return fmt.Errorf("failed to get system pods: %w", err)
	}
	cm.logger.Info(fmt.Sprintf("System pods: %s", podStatus))

	// Deploy test application
	testApp := cm.generateTestApplicationManifest()
	testPath := "/tmp/test-app.yaml"
	if err := cm.sshClient.CopyContent(ctx, controller.IPAddress, testApp, testPath); err != nil {
		return fmt.Errorf("failed to upload test app: %w", err)
	}
	applyTestCmd := fmt.Sprintf("kubectl apply -f %s --kubeconfig /var/lib/kubernetes/admin.kubeconfig", testPath)
	if _, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress, applyTestCmd); err != nil {
		return fmt.Errorf("failed to apply test app: %w", err)
	}

	time.Sleep(30 * time.Second)
	testStatus, err := cm.sshClient.ExecuteCommand(ctx, controller.IPAddress,
		"kubectl get deployment test-deployment --kubeconfig /var/lib/kubernetes/admin.kubeconfig")
	if err != nil {
		return fmt.Errorf("failed to get test app status: %w", err)
	}
	cm.logger.Info(fmt.Sprintf("Test application status: %s", testStatus))

	cm.logger.Info("Cluster validation completed")
	return nil
}


// 4. Add service health check helper
func (cm *ClusterManager) waitForService(ctx context.Context, host, serviceName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		output, err := cm.sshClient.ExecuteCommand(ctx, host, 
			fmt.Sprintf("sudo systemctl is-active %s", serviceName))
		if err == nil && strings.TrimSpace(output) == "active" {
			cm.logger.Info(fmt.Sprintf("Service %s is healthy", serviceName))
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("service %s did not become healthy within %v", serviceName, timeout)
}

// 5. Add checksum verification for downloads
// func (cm *ClusterManager) downloadWithVerification(ctx context.Context, host, url, checksum, destination string) error {
// 	downloadCmd := fmt.Sprintf("wget -q --show-progress --https-only '%s' -O %s", url, destination)
// 	if _, err := cm.sshClient.ExecuteCommand(ctx, host, downloadCmd); err != nil {
// 		return fmt.Errorf("failed to download %s: %w", url, err)
// 	}

// 	// Verify checksum if provided
// 	if checksum != "" {
// 		verifyCmd := fmt.Sprintf("echo '%s %s' | sha256sum -c", checksum, destination)
// 		if _, err := cm.sshClient.ExecuteCommand(ctx, host, verifyCmd); err != nil {
// 			return fmt.Errorf("checksum verification failed for %s: %w", destination, err)
// 		}
// 	}
// 	return nil
// }

// // 6. Add improved error handling with retries
// func (cm *ClusterManager) executeWithRetry(ctx context.Context, host, command string) (string, error) {
// 	var lastErr error
// 	maxRetries := 5
// 	for i := 0; i < maxRetries; i++ {
// 		result, err := cm.sshClient.ExecuteCommand(ctx, host, command)
// 		if err == nil {
// 			return result, nil
// 		}
// 		lastErr = err
// 		if i < maxRetries-1 {
// 			cm.logger.Warn(fmt.Sprintf("Command failed, retrying (%d/%d): %v", i+1, maxRetries, err))
// 			time.Sleep(time.Duration(i+1) * 5 * time.Second)
// 		}
// 	}
// 	return "", fmt.Errorf("command failed after %d retries: %w", maxRetries, lastErr)
// }