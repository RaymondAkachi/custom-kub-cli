package clustersetup

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Mock implementations for testing
type MockSSHClient struct {
	commands     []string
	responses    map[string]string
	errors       map[string]error
	filesUploaded map[string]string
}

func NewMockSSHClient() *MockSSHClient {
	return &MockSSHClient{
		commands:     []string{},
		responses:    make(map[string]string),
		errors:       make(map[string]error),
		filesUploaded: make(map[string]string),
	}
}

func (m *MockSSHClient) ExecuteCommand(ctx context.Context, host, command string) (string, error) {
	m.commands = append(m.commands, fmt.Sprintf("%s: %s", host, command))
	if err, exists := m.errors[command]; exists {
		return "", err
	}
	if response, exists := m.responses[command]; exists {
		return response, nil
	}
	return "success", nil
}

func (m *MockSSHClient) CopyFile(ctx context.Context, host, localPath, remotePath string) error {
	content, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	m.filesUploaded[remotePath] = string(content)
	return nil
}

func (m *MockSSHClient) CopyContent(ctx context.Context, host, content, remotePath string) error {
	m.filesUploaded[remotePath] = content
	return nil
}

func (m *MockSSHClient) GetExecutedCommands() []string {
	return m.commands
}

func (m *MockSSHClient) SetCommandResponse(command, response string) {
	m.responses[command] = response
}

func (m *MockSSHClient) SetCommandError(command string, err error) {
	m.errors[command] = err
}

type MockLogger struct {
	logs []string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{logs: []string{}}
}

func (l *MockLogger) Info(msg string, args ...interface{})  { l.logs = append(l.logs, fmt.Sprintf("INFO: "+msg, args...)) }
func (l *MockLogger) Error(msg string, args ...interface{}) { l.logs = append(l.logs, fmt.Sprintf("ERROR: "+msg, args...)) }
func (l *MockLogger) Debug(msg string, args ...interface{}) { l.logs = append(l.logs, fmt.Sprintf("DEBUG: "+msg, args...)) }
func (l *MockLogger) Warn(msg string, args ...interface{})  { l.logs = append(l.logs, fmt.Sprintf("WARN: "+msg, args...)) }

func (l *MockLogger) GetLogs() []string {
	return l.logs
}

type MockProgressReporter struct {
	steps []string
}

func NewMockProgressReporter() *MockProgressReporter {
	return &MockProgressReporter{steps: []string{}}
}

func (p *MockProgressReporter) Start(total int, description string) {
	p.steps = append(p.steps, fmt.Sprintf("START: %s (%d)", description, total))
}

func (p *MockProgressReporter) Update(current int, status string) {
	p.steps = append(p.steps, fmt.Sprintf("UPDATE: %s (%d)", status, current))
}

func (p *MockProgressReporter) Finish(success bool, message string) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}
	p.steps = append(p.steps, fmt.Sprintf("%s: %s", status, message))
}

func (p *MockProgressReporter) ReportProgress(step, totalSteps int, phase string) {
	p.steps = append(p.steps, fmt.Sprintf("PROGRESS: Step %d/%d: %s", step, totalSteps, phase))
}

// Test helper functions
func createTestConfig() ClusterConfig {
	return ClusterConfig{
		ClusterName:       "test-cluster",
		KubernetesVersion: "v1.26.0",
		EtcdVersion:       "v3.5.9",
		ContainerdVersion: "1.7.2",
		CNIVersion:        "v1.3.0",
		CoreDNSVersion:    "1.10.1",
		PodCIDR:           "10.200.0.0/16",
		ServiceCIDR:       "10.32.0.0/24",
		ClusterDNS:        "10.32.0.10",
		WorkDir:           "/tmp/test-k8s",
		SSHKey:            "~/.ssh/test.pem",
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
		},
		Certificates: CertificateConfig{
			Country:            "US",
			State:              "CA",
			City:               "SF",
			Organization:       "Test",
			OrganizationalUnit: "IT",
			ValidityDays:       365,
		},
	}
}

// Configuration Tests
func TestLoadClusterConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: `
cluster_name: "test-cluster"
kubernetes_version: "v1.26.0"
etcd_version: "v3.5.9"
containerd_version: "1.7.2"
cni_version: "v1.3.0"
coredns_version: "1.10.1"
pod_cidr: "10.200.0.0/16"
service_cidr: "10.32.0.0/24"
cluster_dns: "10.32.0.10"
work_dir: "/tmp/test-k8s"
ssh_key: "~/.ssh/test.pem"
ssh_user: "ubuntu"
controller:
  name: "controller-0"
  ip_address: "10.240.0.10"
  hostname: "controller-0"
workers:
  - name: "worker-0"
    ip_address: "10.240.0.20"
    hostname: "worker-0"
    pod_cidr: "10.200.0.0/24"
certificates:
  country: "US"
  state: "CA"
  city: "SF"
  organization: "Test"
  organizational_unit: "IT"
  validity_days: 365
`,
			expectError: false,
		},
		{
			name: "missing cluster name",
			config: `
kubernetes_version: "v1.26.0"
etcd_version: "v3.5.9"
`,
			expectError: true,
			errorMsg:    "cluster_name is required",
		},
		{
			name: "missing workers",
			config: `
cluster_name: "test"
kubernetes_version: "v1.26.0"
etcd_version: "v3.5.9"
containerd_version: "1.7.2"
cni_version: "v1.3.0"
coredns_version: "1.10.1"
pod_cidr: "10.200.0.0/16"
service_cidr: "10.32.0.0/24"
cluster_dns: "10.32.0.10"
work_dir: "/tmp/test-k8s"
ssh_key: "~/.ssh/test.pem"
ssh_user: "ubuntu"
controller:
  name: "controller-0"
  ip_address: "10.240.0.10"
  hostname: "controller-0"
workers: []
certificates:
  country: "US"
  state: "CA"
  city: "SF"
  organization: "Test"
  organizational_unit: "IT"
  validity_days: 365
`,
			expectError: true,
			errorMsg:    "at least one worker node is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "config-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.config); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}
			tmpFile.Close()

			config, err := LoadClusterConfig(tmpFile.Name())

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if config.ClusterName == "" {
					t.Error("Config not loaded properly")
				}
			}
		})
	}
}

func TestGenerateDefaultConfig(t *testing.T) {
	config := GenerateDefaultConfig()
	
	if config.ClusterName == "" {
		t.Error("Default config should have cluster name")
	}
	if len(config.Workers) == 0 {
		t.Error("Default config should have workers")
	}
	if config.Certificates.ValidityDays <= 0 {
		t.Error("Default config should have valid certificate validity days")
	}
}

func TestSaveConfig(t *testing.T) {
	config := createTestConfig()
	tmpFile, err := os.CreateTemp("", "save-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if err := SaveConfig(config, tmpFile.Name()); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was written and can be loaded back
	loadedConfig, err := LoadClusterConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.ClusterName != config.ClusterName {
		t.Errorf("Saved config mismatch: expected %s, got %s", 
			config.ClusterName, loadedConfig.ClusterName)
	}
}

// Certificate Tests
func TestCertificateGeneration(t *testing.T) {
	workDir, err := os.MkdirTemp("", "cert-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	certManager := NewCertificateManager()
	config := CertificateConfig{
		Country:            "US",
		State:              "CA",
		City:               "SF",
		Organization:       "Test",
		OrganizationalUnit: "IT",
		ValidityDays:       365,
	}

	t.Run("CA Generation", func(t *testing.T) {
		if err := certManager.GenerateCA(workDir, config); err != nil {
			t.Fatalf("CA generation failed: %v", err)
		}

		// Check if CA files were created
		caFiles := []string{"ca.pem", "ca-key.pem", "ca-config.json"}
		for _, file := range caFiles {
			filePath := filepath.Join(workDir, file)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("Expected file %s was not created", file)
			}
		}

		// Validate CA certificate
		caCertData, err := os.ReadFile(filepath.Join(workDir, "ca.pem"))
		if err != nil {
			t.Fatalf("Failed to read CA cert: %v", err)
		}

		block, _ := pem.Decode(caCertData)
		if block == nil {
			t.Fatal("Failed to decode CA certificate PEM")
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatalf("Failed to parse CA certificate: %v", err)
		}

		if !cert.IsCA {
			t.Error("Generated certificate is not a CA")
		}

		if cert.Subject.Organization[0] != config.Organization {
			t.Errorf("Expected organization %s, got %s", 
				config.Organization, cert.Subject.Organization[0])
		}
	})

	t.Run("Client Certificate Generation", func(t *testing.T) {
		clientName := "test-client"
		if err := certManager.GenerateClientCert(workDir, clientName, config); err != nil {
			t.Fatalf("Client cert generation failed: %v", err)
		}

		// Check if client cert files were created
		clientFiles := []string{clientName + ".pem", clientName + "-key.pem"}
		for _, file := range clientFiles {
			filePath := filepath.Join(workDir, file)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("Expected file %s was not created", file)
			}
		}

		// Validate client certificate
		certData, err := os.ReadFile(filepath.Join(workDir, clientName+".pem"))
		if err != nil {
			t.Fatalf("Failed to read client cert: %v", err)
		}

		block, _ := pem.Decode(certData)
		if block == nil {
			t.Fatal("Failed to decode client certificate PEM")
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatalf("Failed to parse client certificate: %v", err)
		}

		if cert.Subject.CommonName != clientName {
			t.Errorf("Expected CN %s, got %s", clientName, cert.Subject.CommonName)
		}
	})

	t.Run("Server Certificate Generation", func(t *testing.T) {
		serverName := "kubernetes"
		hosts := []string{"127.0.0.1", "10.32.0.1", "kubernetes", "kubernetes.default"}
		
		if err := certManager.GenerateServerCert(workDir, serverName, hosts, config); err != nil {
			t.Fatalf("Server cert generation failed: %v", err)
		}

		// Check if server cert files were created
		serverFiles := []string{serverName + ".pem", serverName + "-key.pem"}
		for _, file := range serverFiles {
			filePath := filepath.Join(workDir, file)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("Expected file %s was not created", file)
			}
		}

		// Validate server certificate has correct SANs
		certData, err := os.ReadFile(filepath.Join(workDir, serverName+".pem"))
		if err != nil {
			t.Fatalf("Failed to read server cert: %v", err)
		}

		block, _ := pem.Decode(certData)
		if block == nil {
			t.Fatal("Failed to decode server certificate PEM")
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatalf("Failed to parse server certificate: %v", err)
		}

		// Check if all hosts are in SANs
		for _, host := range hosts {
			found := false
			for _, san := range cert.DNSNames {
				if san == host {
					found = true
					break
				}
			}
			if !found && host != "127.0.0.1" && host != "10.32.0.1" {
				t.Errorf("Host %s not found in certificate SANs", host)
			}
		}
	})
}

// Configuration File Generation Tests
func TestConfigurationFileGeneration(t *testing.T) {
	config := createTestConfig()
	logger := NewMockLogger()
	progress := NewMockProgressReporter()
	sshClient := NewMockSSHClient()
	certManager := NewCertificateManager()
	
	cm := NewClusterManager(config, logger, sshClient, certManager, progress)

	t.Run("Encryption Config Generation", func(t *testing.T) {
		workDir, err := os.MkdirTemp("", "encryption-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(workDir)

		if err := cm.generateEncryptionConfig(workDir); err != nil {
			t.Fatalf("Failed to generate encryption config: %v", err)
		}

		encryptionFile := filepath.Join(workDir, "encryption-config.yaml")
		if _, err := os.Stat(encryptionFile); os.IsNotExist(err) {
			t.Error("Encryption config file was not created")
		}

		content, err := os.ReadFile(encryptionFile)
		if err != nil {
			t.Fatalf("Failed to read encryption config: %v", err)
		}

		if !strings.Contains(string(content), "EncryptionConfig") {
			t.Error("Encryption config doesn't contain required EncryptionConfig kind")
		}
		if !strings.Contains(string(content), "aescbc") {
			t.Error("Encryption config doesn't contain aescbc provider")
		}
	})

	t.Run("Kubeconfig Generation", func(t *testing.T) {
		workDir, err := os.MkdirTemp("", "kubeconfig-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(workDir)

		// Generate CA first (required for kubeconfig)
		if err := certManager.GenerateCA(workDir, config.Certificates); err != nil {
			t.Fatalf("Failed to generate CA: %v", err)
		}
		if err := certManager.GenerateClientCert(workDir, "admin", config.Certificates); err != nil {
			t.Fatalf("Failed to generate admin cert: %v", err)
		}

		if err := cm.generateKubeconfig(workDir, "admin", ""); err != nil {
			t.Fatalf("Failed to generate kubeconfig: %v", err)
		}

		kubeconfigFile := filepath.Join(workDir, "admin.kubeconfig")
		if _, err := os.Stat(kubeconfigFile); os.IsNotExist(err) {
			t.Error("Kubeconfig file was not created")
		}

		content, err := os.ReadFile(kubeconfigFile)
		if err != nil {
			t.Fatalf("Failed to read kubeconfig: %v", err)
		}

		kubeconfigStr := string(content)
		if !strings.Contains(kubeconfigStr, "apiVersion: v1") {
			t.Error("Kubeconfig doesn't contain correct apiVersion")
		}
		if !strings.Contains(kubeconfigStr, config.ClusterName) {
			t.Error("Kubeconfig doesn't contain cluster name")
		}
		if !strings.Contains(kubeconfigStr, config.Controller.IPAddress) {
			t.Error("Kubeconfig doesn't contain controller IP")
		}
	})

	t.Run("Service File Generation", func(t *testing.T) {
		// Test etcd service generation
		etcdService := cm.generateEtcdService(config.Controller)
		if etcdService == "" {
			t.Error("etcd service generation returned empty string")
		}
		if !strings.Contains(etcdService, "etcd") {
			t.Error("etcd service doesn't contain etcd binary")
		}
		if !strings.Contains(etcdService, config.Controller.Name) {
			t.Error("etcd service doesn't contain controller name")
		}

		// Test API server service generation
		apiService := cm.generateAPIServerService()
		if apiService == "" {
			t.Error("API server service generation returned empty string")
		}
		if !strings.Contains(apiService, "kube-apiserver") {
			t.Error("API server service doesn't contain kube-apiserver binary")
		}
		if !strings.Contains(apiService, config.Controller.IPAddress) {
			t.Error("API server service doesn't contain controller IP")
		}
		if !strings.Contains(apiService, config.ServiceCIDR) {
			t.Error("API server service doesn't contain service CIDR")
		}

		// Test controller manager service generation
		controllerService := cm.generateControllerManagerService()
		if !strings.Contains(controllerService, "kube-controller-manager") {
			t.Error("Controller manager service doesn't contain binary name")
		}
		if !strings.Contains(controllerService, config.PodCIDR) {
			t.Error("Controller manager service doesn't contain pod CIDR")
		}

		// Test scheduler service generation
		schedulerService := cm.generateSchedulerService()
		if !strings.Contains(schedulerService, "kube-scheduler") {
			t.Error("Scheduler service doesn't contain binary name")
		}

		// Test containerd service generation
		containerdService := cm.generateContainerdService()
		if !strings.Contains(containerdService, "containerd") {
			t.Error("Containerd service doesn't contain binary name")
		}

		// Test kubelet service generation
		kubeletService := cm.generateKubeletService(config.Workers[0])
		if !strings.Contains(kubeletService, "kubelet") {
			t.Error("Kubelet service doesn't contain binary name")
		}
		if !strings.Contains(kubeletService, config.Workers[0].Name) {
			t.Error("Kubelet service doesn't contain worker name")
		}

		// Test kube-proxy service generation
		proxyService := cm.generateKubeProxyService()
		if !strings.Contains(proxyService, "kube-proxy") {
			t.Error("Kube-proxy service doesn't contain binary name")
		}
	})

	t.Run("CNI Configuration Generation", func(t *testing.T) {
		worker := config.Workers[0]
		
		// Test bridge network config
		bridgeConfig := cm.generateBridgeNetworkConfig(worker.PodCIDR)
		if !strings.Contains(bridgeConfig, worker.PodCIDR) {
			t.Error("Bridge config doesn't contain pod CIDR")
		}
		if !strings.Contains(bridgeConfig, "bridge") {
			t.Error("Bridge config doesn't contain bridge type")
		}

		// Test loopback network config
		loopbackConfig := cm.generateLoopbackNetworkConfig()
		if !strings.Contains(loopbackConfig, "loopback") {
			t.Error("Loopback config doesn't contain loopback type")
		}
	})

	t.Run("Kubelet Configuration Generation", func(t *testing.T) {
		worker := config.Workers[0]
		kubeletConfig := cm.generateKubeletConfig(worker)
		
		if !strings.Contains(kubeletConfig, worker.IPAddress) {
			t.Error("Kubelet config doesn't contain worker IP")
		}
		if !strings.Contains(kubeletConfig, worker.PodCIDR) {
			t.Error("Kubelet config doesn't contain pod CIDR")
		}
		if !strings.Contains(kubeletConfig, config.ClusterDNS) {
			t.Error("Kubelet config doesn't contain cluster DNS")
		}
	})

	t.Run("Kube-proxy Configuration Generation", func(t *testing.T) {
		proxyConfig := cm.generateKubeProxyConfig()
		
		if !strings.Contains(proxyConfig, config.PodCIDR) {
			t.Error("Kube-proxy config doesn't contain pod CIDR")
		}
		if !strings.Contains(proxyConfig, "iptables") {
			t.Error("Kube-proxy config doesn't contain iptables mode")
		}
	})

	t.Run("CoreDNS Manifest Generation", func(t *testing.T) {
		coreDNSManifest := cm.generateCoreDNSManifest()
		
		if !strings.Contains(coreDNSManifest, config.CoreDNSVersion) {
			t.Error("CoreDNS manifest doesn't contain CoreDNS version")
		}
		if !strings.Contains(coreDNSManifest, config.ClusterDNS) {
			t.Error("CoreDNS manifest doesn't contain cluster DNS IP")
		}
		if !strings.Contains(coreDNSManifest, "coredns") {
			t.Error("CoreDNS manifest doesn't contain coredns")
		}
	})
}

// Cluster Setup Integration Tests
func TestClusterSetupFlow(t *testing.T) {
	config := createTestConfig()
	logger := NewMockLogger()
	progress := NewMockProgressReporter()
	sshClient := NewMockSSHClient()
	certManager := NewCertificateManager()
	
	// Setup mock responses for SSH commands
	sshClient.SetCommandResponse("echo 'SSH test'", "SSH test")
	sshClient.SetCommandResponse("sudo systemctl is-active etcd", "active")
	sshClient.SetCommandResponse("sudo systemctl is-active kube-apiserver", "active")
	sshClient.SetCommandResponse("sudo systemctl is-active kube-controller-manager", "active")
	sshClient.SetCommandResponse("sudo systemctl is-active kube-scheduler", "active")
	sshClient.SetCommandResponse("kubectl get nodes --kubeconfig /var/lib/kubernetes/admin.kubeconfig", 
		"NAME         STATUS   ROLES    AGE   VERSION\nworker-0     Ready    <none>   1m    v1.26.0\nworker-1     Ready    <none>   1m    v1.26.0")
	sshClient.SetCommandResponse("kubectl get pods -n kube-system --kubeconfig /var/lib/kubernetes/admin.kubeconfig", 
		"NAME                      READY   STATUS    RESTARTS   AGE\ncoredns-xxx               2/2     Running   0          1m")
	sshClient.SetCommandResponse("kubectl get deployment test-deployment --kubeconfig /var/lib/kubernetes/admin.kubeconfig", 
		"NAME              READY   UP-TO-DATE   AVAILABLE   AGE\ntest-deployment   2/2     2            2           1m")

	cm := NewClusterManager(config, logger, sshClient, certManager, progress)

	// Create temporary work directory
	workDir, err := os.MkdirTemp("", "cluster-setup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp work dir: %v", err)
	}
	defer os.RemoveAll(workDir)
	config.WorkDir = workDir
	cm.config = config

	t.Run("Prerequisites Validation", func(t *testing.T) {
		if err := cm.ValidateK8sPrerequisites(); err != nil {
			t.Fatalf("Prerequisites validation failed: %v", err)
		}

		commands := sshClient.GetExecutedCommands()
		expectedNodes := len(config.Workers) + 1 // +1 for controller
		if len(commands) < expectedNodes {
			t.Errorf("Expected at least %d SSH test commands, got %d", expectedNodes, len(commands))
		}
	})

	t.Run("Certificate Generation Phase", func(t *testing.T) {
		ctx := context.Background()
		if err := cm.generateCertificates(ctx, workDir); err != nil {
			t.Fatalf("Certificate generation failed: %v", err)
		}

		// Verify all required certificates were generated
		requiredCerts := []string{
			"ca.pem", "ca-key.pem",
			"admin.pem", "admin-key.pem",
			"kube-controller-manager.pem", "kube-controller-manager-key.pem",
			"kube-proxy.pem", "kube-proxy-key.pem",
			"kube-scheduler.pem", "kube-scheduler-key.pem",
			"service-account.pem", "service-account-key.pem",
			"kubernetes.pem", "kubernetes-key.pem",
		}

		for _, worker := range config.Workers {
			requiredCerts = append(requiredCerts, worker.Name+".pem", worker.Name+"-key.pem")
		}

		for _, cert := range requiredCerts {
			certPath := filepath.Join(workDir, cert)
			if _, err := os.Stat(certPath); os.IsNotExist(err) {
				t.Errorf("Required certificate %s was not generated", cert)
			}
		}
	})

	t.Run("Configuration Generation Phase", func(t *testing.T) {
		ctx := context.Background()
		if err := cm.createConfigurations(ctx, workDir); err != nil {
			t.Fatalf("Configuration generation failed: %v", err)
		}

		// Verify all required configurations were generated
		requiredConfigs := []string{
			"encryption-config.yaml",
			"kube-proxy.kubeconfig",
			"kube-controller-manager.kubeconfig",
			"kube-scheduler.kubeconfig",
			"admin.kubeconfig",
		}

		for _, worker := range config.Workers {
			requiredConfigs = append(requiredConfigs, worker.Name+".kubeconfig")
		}

		for _, configFile := range requiredConfigs {
			configPath := filepath.Join(workDir, configFile)
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				t.Errorf("Required configuration %s was not generated", configFile)
			}
		}
	})

	t.Run("Full Cluster Setup", func(t *testing.T) {
		ctx := context.Background()
		if err := cm.SetupCluster(ctx); err != nil {
			t.Fatalf("Cluster setup failed: %v", err)
		}

		// Verify progress was reported
		steps := progress.steps
		if len(steps) == 0 {
			t.Error("No progress steps were reported")
		}

		// Verify expected phases were executed
		expectedPhases := []string{
			"Checking Prerequisites",
			"Generating Certificates", 
			"Creating Configurations",
			"Setting Up Control Plane",
			"Setting Up Worker Nodes",
			"Setting Up Networking",
			"Validating Cluster",
		}

		progressStr := strings.Join(steps, " ")
		for _, phase := range expectedPhases {
			if !strings.Contains(progressStr, phase) {
				t.Errorf("Expected phase '%s' not found in progress", phase)
			}
		}

		// Verify SSH commands were executed
		commands := sshClient.GetExecutedCommands()
		if len(commands) == 0 {
			t.Error("No SSH commands were executed")
		}

		// Verify essential commands were run
		commandStr := strings.Join(commands, " ")
		essentialCommands := []string{
			"wget", // Binary downloads
			"systemctl", // Service management
			"kubectl", // Kubernetes commands
		}

		for _, cmd := range essentialCommands {
			if !strings.Contains(commandStr, cmd) {
				t.Errorf("Essential command '%s' was not executed", cmd)
			}
		}
	})
}

// Component-specific tests
func TestControlPlaneSetup(t *testing.T) {
	config := createTestConfig()
	logger := NewMockLogger()
	progress := NewMockProgressReporter()
	sshClient := NewMockSSHClient()
	certManager := NewCertificateManager()

	// Setup service health check responses
	sshClient.SetCommandResponse("sudo systemctl is-active etcd", "active")
	sshClient.SetCommandResponse("sudo systemctl is-active kube-apiserver", "active")
	sshClient.SetCommandResponse("sudo systemctl is-active kube-controller-manager", "active")
	sshClient.SetCommandResponse("sudo systemctl is-active kube-scheduler", "active")

	cm := NewClusterManager(config, logger, sshClient, certManager, progress)

	workDir, err := os.MkdirTemp("", "control-plane-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp work dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	// Generate required certificates first
	ctx := context.Background()
	if err := cm.generateCertificates(ctx, workDir); err != nil {
		t.Fatalf("Failed to generate certificates: %v", err)
	}
	if err := cm.createConfigurations(ctx, workDir); err != nil {
		t.Fatalf("Failed to create configurations: %v", err)
	}

	t.Run("Control Plane Setup", func(t *testing.T) {
		if err := cm.setupControlPlane(ctx, workDir); err != nil {
			t.Fatalf("Control plane setup failed: %v", err)
		}

		commands := sshClient.GetExecutedCommands()
		commandStr := strings.Join(commands, " ")

		// Verify etcd setup commands
		if !strings.Contains(commandStr, "etcd") {
			t.Error("etcd setup commands not found")
		}

		// Verify kubernetes binary downloads
		if !strings.Contains(commandStr, "kube-apiserver") {
			t.Error("kube-apiserver download not found")
		}
		if !strings.Contains(commandStr, "kube-controller-manager") {
			t.Error("kube-controller-manager download not found")
		}
		if !strings.Contains(commandStr, "kube-scheduler") {
			t.Error("kube-scheduler download not found")
		}

		// Verify service startup commands
		if !strings.Contains(commandStr, "systemctl start etcd") {
			t.Error("etcd start command not found")
		}
		if !strings.Contains(commandStr, "systemctl start kube-apiserver") {
			t.Error("kube-apiserver start command not found")
		}

		// Verify files were uploaded
		filesUploaded := sshClient.filesUploaded
		expectedServices := []string{
			"/etc/systemd/system/etcd.service",
			"/etc/systemd/system/kube-apiserver.service",
			"/etc/systemd/system/kube-controller-manager.service",
			"/etc/systemd/system/kube-scheduler.service",
		}

		for _, service := range expectedServices {
			if _, exists := filesUploaded[service]; !exists {
				t.Errorf("Service file %s was not uploaded", service)
			}
		}
	})
}

func TestWorkerNodeSetup(t *testing.T) {
	config := createTestConfig()
	logger := NewMockLogger()
	progress := NewMockProgressReporter()
	sshClient := NewMockSSHClient()
	certManager := NewCertificateManager()

	cm := NewClusterManager(config, logger, sshClient, certManager, progress)

	workDir, err := os.MkdirTemp("", "worker-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp work dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	// Generate required certificates first
	ctx := context.Background()
	if err := cm.generateCertificates(ctx, workDir); err != nil {
		t.Fatalf("Failed to generate certificates: %v", err)
	}
	if err := cm.createConfigurations(ctx, workDir); err != nil {
		t.Fatalf("Failed to create configurations: %v", err)
	}

	t.Run("Single Worker Setup", func(t *testing.T) {
		worker := config.Workers[0]
		if err := cm.setupSingleWorkerNode(ctx, workDir, worker); err != nil {
			t.Fatalf("Worker setup failed: %v", err)
		}

		commands := sshClient.GetExecutedCommands()
		commandStr := strings.Join(commands, " ")

		// Verify dependency installation
		if !strings.Contains(commandStr, "apt-get update") {
			t.Error("Package update command not found")
		}
		if !strings.Contains(commandStr, "socat conntrack ipset") {
			t.Error("Required packages installation not found")
		}

		// Verify CNI plugin installation
		if !strings.Contains(commandStr, "cni-plugins") {
			t.Error("CNI plugins installation not found")
		}

		// Verify containerd installation
		if !strings.Contains(commandStr, "containerd") {
			t.Error("containerd installation not found")
		}
		if !strings.Contains(commandStr, "runc") {
			t.Error("runc installation not found")
		}

		// Verify kubernetes binary installation
		if !strings.Contains(commandStr, "kubelet") {
			t.Error("kubelet installation not found")
		}
		if !strings.Contains(commandStr, "kube-proxy") {
			t.Error("kube-proxy installation not found")
		}

		// Verify service startup
		if !strings.Contains(commandStr, "systemctl start containerd") {
			t.Error("containerd start command not found")
		}
		if !strings.Contains(commandStr, "systemctl start kubelet") {
			t.Error("kubelet start command not found")
		}

		// Verify configuration files were uploaded
		filesUploaded := sshClient.filesUploaded
		expectedConfigs := []string{
			"/etc/containerd/config.toml",
			"/etc/cni/net.d/10-bridge.conf",
			"/etc/cni/net.d/99-loopback.conf",
			"/var/lib/kubelet/kubelet-config.yaml",
			"/var/lib/kube-proxy/kube-proxy-config.yaml",
		}

		for _, configFile := range expectedConfigs {
			if _, exists := filesUploaded[configFile]; !exists {
				t.Errorf("Configuration file %s was not uploaded", configFile)
			}
		}

		// Verify bridge network config contains correct pod CIDR
		bridgeConfig := filesUploaded["/etc/cni/net.d/10-bridge.conf"]
		if !strings.Contains(bridgeConfig, worker.PodCIDR) {
			t.Error("Bridge config doesn't contain worker pod CIDR")
		}
	})

	t.Run("All Workers Setup", func(t *testing.T) {
		// Reset SSH client for fresh command tracking
		sshClient = NewMockSSHClient()
		cm.sshClient = sshClient

		if err := cm.setupWorkerNodes(ctx, workDir); err != nil {
			t.Fatalf("All workers setup failed: %v", err)
		}

		commands := sshClient.GetExecutedCommands()
		
		// Verify commands were run for each worker
		for _, worker := range config.Workers {
			workerCommands := 0
			for _, cmd := range commands {
				if strings.Contains(cmd, worker.IPAddress) {
					workerCommands++
				}
			}
			if workerCommands == 0 {
				t.Errorf("No commands executed for worker %s", worker.Name)
			}
		}
	})
}

func TestNetworkingSetup(t *testing.T) {
	config := createTestConfig()
	logger := NewMockLogger()
	progress := NewMockProgressReporter()
	sshClient := NewMockSSHClient()
	certManager := NewCertificateManager()

	cm := NewClusterManager(config, logger, sshClient, certManager, progress)

	t.Run("Pod Routing Setup", func(t *testing.T) {
		ctx := context.Background()
		if err := cm.setupNetworking(ctx); err != nil {
			t.Fatalf("Networking setup failed: %v", err)
		}

		commands := sshClient.GetExecutedCommands()
		commandStr := strings.Join(commands, " ")

		// Verify routing commands were executed
		if !strings.Contains(commandStr, "ip route add") {
			t.Error("Pod routing commands not found")
		}

		// Verify CoreDNS deployment
		if !strings.Contains(commandStr, "kubectl apply") {
			t.Error("CoreDNS deployment command not found")
		}

		// Verify CoreDNS manifest was uploaded
		filesUploaded := sshClient.filesUploaded
		coreDNSUploaded := false
		for path := range filesUploaded {
			if strings.Contains(path, "coredns") {
				coreDNSUploaded = true
				break
			}
		}
		if !coreDNSUploaded {
			t.Error("CoreDNS manifest was not uploaded")
		}
	})
}

func TestClusterValidation(t *testing.T) {
	config := createTestConfig()
	logger := NewMockLogger()
	progress := NewMockProgressReporter()
	sshClient := NewMockSSHClient()
	certManager := NewCertificateManager()

	// Setup expected validation responses
	sshClient.SetCommandResponse("kubectl get nodes --kubeconfig /var/lib/kubernetes/admin.kubeconfig",
		"NAME         STATUS   ROLES    AGE   VERSION\nworker-0     Ready    <none>   1m    v1.26.0\nworker-1     Ready    <none>   1m    v1.26.0")
	sshClient.SetCommandResponse("kubectl get pods -n kube-system --kubeconfig /var/lib/kubernetes/admin.kubeconfig",
		"NAME                      READY   STATUS    RESTARTS   AGE\ncoredns-xxx               2/2     Running   0          1m")
	sshClient.SetCommandResponse("kubectl get deployment test-deployment --kubeconfig /var/lib/kubernetes/admin.kubeconfig",
		"NAME              READY   UP-TO-DATE   AVAILABLE   AGE\ntest-deployment   2/2     2            2           1m")

	cm := NewClusterManager(config, logger, sshClient, certManager, progress)

	t.Run("Cluster Validation", func(t *testing.T) {
		ctx := context.Background()
		if err := cm.validateCluster(ctx); err != nil {
			t.Fatalf("Cluster validation failed: %v", err)
		}

		commands := sshClient.GetExecutedCommands()
		commandStr := strings.Join(commands, " ")

		// Verify validation commands were executed
		if !strings.Contains(commandStr, "kubectl get nodes") {
			t.Error("Node status check not found")
		}
		if !strings.Contains(commandStr, "kubectl get pods -n kube-system") {
			t.Error("System pods check not found")
		}
		if !strings.Contains(commandStr, "kubectl get deployment test-deployment") {
			t.Error("Test deployment check not found")
		}

		// Verify test application was deployed
		if !strings.Contains(commandStr, "kubectl apply -f") {
			t.Error("Test application deployment not found")
		}

		// Verify test application manifest was uploaded
		filesUploaded := sshClient.filesUploaded
		testAppUploaded := false
		for path := range filesUploaded {
			if strings.Contains(path, "test-app") {
				testAppUploaded = true
				break
			}
		}
		if !testAppUploaded {
			t.Error("Test application manifest was not uploaded")
		}

		logs := logger.GetLogs()
		logStr := strings.Join(logs, " ")
		if !strings.Contains(logStr, "Cluster validation completed") {
			t.Error("Validation completion not logged")
		}
	})

	t.Run("Cluster Status Retrieval", func(t *testing.T) {
		ctx := context.Background()
		status, err := cm.GetClusterStatus(ctx)
		if err != nil {
			t.Fatalf("Failed to get cluster status: %v", err)
		}

		if status.Nodes == "" {
			t.Error("Node status is empty")
		}
		if status.PodStatus == "" {
			t.Error("Pod status is empty")
		}
		if status.TestStatus == "" {
			t.Error("Test status is empty")
		}

		// Verify status contains expected information
		if !strings.Contains(status.Nodes, "Ready") {
			t.Error("Node status doesn't show Ready nodes")
		}
		if !strings.Contains(status.PodStatus, "Running") {
			t.Error("Pod status doesn't show Running pods")
		}
	})
}

func TestClusterDestruction(t *testing.T) {
	config := createTestConfig()
	logger := NewMockLogger()
	progress := NewMockProgressReporter()
	sshClient := NewMockSSHClient()
	certManager := NewCertificateManager()

	cm := NewClusterManager(config, logger, sshClient, certManager, progress)

	// Create temporary work directory to test cleanup
	workDir, err := os.MkdirTemp("", "destroy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp work dir: %v", err)
	}
	config.WorkDir = workDir
	cm.config = config

	// Create some test files
	testFile := filepath.Join(workDir, "test-file.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("Cluster Destruction", func(t *testing.T) {
		ctx := context.Background()
		if err := cm.DestroyCluster(ctx); err != nil {
			t.Fatalf("Cluster destruction failed: %v", err)
		}

		// Verify cleanup commands were executed
		commands := sshClient.GetExecutedCommands()
		commandStr := strings.Join(commands, " ")

		// Verify service stop commands
		if !strings.Contains(commandStr, "systemctl stop") {
			t.Error("Service stop commands not found")
		}
		if !strings.Contains(commandStr, "systemctl disable") {
			t.Error("Service disable commands not found")
		}

		// Verify cleanup commands
		if !strings.Contains(commandStr, "rm -rf") {
			t.Error("Cleanup commands not found")
		}

		// Verify daemon-reload was called
		if !strings.Contains(commandStr, "systemctl daemon-reload") {
			t.Error("systemctl daemon-reload not found")
		}

		// Verify work directory was removed
		if _, err := os.Stat(workDir); !os.IsNotExist(err) {
			t.Error("Work directory was not removed")
		}

		logs := logger.GetLogs()
		logStr := strings.Join(logs, " ")
		if !strings.Contains(logStr, "Cluster destroyed successfully") {
			t.Error("Destruction completion not logged")
		}
	})
}

// Error handling tests
func TestErrorHandling(t *testing.T) {
	config := createTestConfig()
	logger := NewMockLogger()
	progress := NewMockProgressReporter()
	sshClient := NewMockSSHClient()
	certManager := NewCertificateManager()

	cm := NewClusterManager(config, logger, sshClient, certManager, progress)

	t.Run("SSH Connection Failure", func(t *testing.T) {
		sshClient.SetCommandError("echo 'SSH test'", fmt.Errorf("connection refused"))

		if err := cm.ValidateK8sPrerequisites(); err == nil {
			t.Error("Expected error for SSH connection failure")
		}
	})

	t.Run("Service Startup Failure", func(t *testing.T) {
		sshClient.SetCommandResponse("sudo systemctl is-active etcd", "failed")

		workDir, err := os.MkdirTemp("", "error-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp work dir: %v", err)
		}
		defer os.RemoveAll(workDir)

		// Generate certificates first
		ctx := context.Background()
		if err := cm.generateCertificates(ctx, workDir); err != nil {
			t.Fatalf("Failed to generate certificates: %v", err)
		}
		if err := cm.createConfigurations(ctx, workDir); err != nil {
			t.Fatalf("Failed to create configurations: %v", err)
		}

		// This should fail due to service health check failure
		if err := cm.setupControlPlane(ctx, workDir); err == nil {
			t.Error("Expected error for service startup failure")
		}
	})

	t.Run("Invalid Configuration", func(t *testing.T) {
		invalidConfig := config
		invalidConfig.ClusterName = ""

		if _, err := LoadClusterConfig("nonexistent.yaml"); err == nil {
			t.Error("Expected error for nonexistent config file")
		}
	})
}

// Performance and resource tests
func TestResourceUsage(t *testing.T) {
	t.Run("Certificate Generation Performance", func(t *testing.T) {
		workDir, err := os.MkdirTemp("", "perf-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(workDir)

		certManager := NewCertificateManager()
		config := CertificateConfig{
			Country:            "US",
			State:              "CA",
			City:               "SF",
			Organization:       "Test",
			OrganizationalUnit: "IT",
			ValidityDays:       365,
		}

		start := time.Now()
		
		// Generate CA
		if err := certManager.GenerateCA(workDir, config); err != nil {
			t.Fatalf("CA generation failed: %v", err)
		}

		// Generate multiple client certificates
		for i := 0; i < 10; i++ {
			name := fmt.Sprintf("client-%d", i)
			if err := certManager.GenerateClientCert(workDir, name, config); err != nil {
				t.Fatalf("Client cert generation failed: %v", err)
			}
		}

		// Generate server certificate
		hosts := []string{"127.0.0.1", "kubernetes", "kubernetes.default"}
		if err := certManager.GenerateServerCert(workDir, "server", hosts, config); err != nil {
			t.Fatalf("Server cert generation failed: %v", err)
		}

		duration := time.Since(start)
		if duration > 30*time.Second {
			t.Errorf("Certificate generation took too long: %v", duration)
		}

		t.Logf("Certificate generation completed in %v", duration)
	})

	t.Run("Memory Usage", func(t *testing.T) {
		// This is a basic test - in real scenarios you'd use runtime.MemStats
		config := createTestConfig()
		
		// Create multiple cluster managers to test memory usage
		for i := 0; i < 100; i++ {
			logger := NewMockLogger()
			progress := NewMockProgressReporter()
			sshClient := NewMockSSHClient()
			certManager := NewCertificateManager()
			
			cm := NewClusterManager(config, logger, sshClient, certManager, progress)
			
			// Use the cluster manager to prevent optimization
			_ = cm.config.ClusterName
		}
		
		// Basic validation that we didn't panic or run out of memory
		t.Log("Memory usage test completed")
	})
}

// Integration test helpers
func TestServiceHealthCheck(t *testing.T) {
	config := createTestConfig()
	logger := NewMockLogger()
	progress := NewMockProgressReporter()
	sshClient := NewMockSSHClient()
	certManager := NewCertificateManager()

	cm := NewClusterManager(config, logger, sshClient, certManager, progress)

	t.Run("Service Health Check Success", func(t *testing.T) {
		sshClient.SetCommandResponse("sudo systemctl is-active test-service", "active")

		ctx := context.Background()
		if err := cm.waitForService(ctx, "10.0.0.1", "test-service", 10*time.Second); err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})

	t.Run("Service Health Check Timeout", func(t *testing.T) {
		sshClient.SetCommandResponse("sudo systemctl is-active failing-service", "failed")

		ctx := context.Background()
		err := cm.waitForService(ctx, "10.0.0.1", "failing-service", 1*time.Second)
		if err == nil {
			t.Error("Expected timeout error")
		}
		if !strings.Contains(err.Error(), "did not become healthy") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	})
}

// Comprehensive end-to-end test
func TestEndToEndClusterSetup(t *testing.T) {
	// This test simulates a complete cluster setup from configuration to validation
	config := createTestConfig()
	
	// Create temporary work directory
	workDir, err := os.MkdirTemp("", "e2e-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp work dir: %v", err)
	}
	defer os.RemoveAll(workDir)
	config.WorkDir = workDir

	// Save configuration
	configPath := filepath.Join(workDir, "cluster-config.yaml")
	if err := SaveConfig(config, configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load configuration back
	loadedConfig, err := LoadClusterConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Setup mocks with comprehensive responses
	logger := NewMockLogger()
	progress := NewMockProgressReporter()
	sshClient := NewMockSSHClient()
	certManager := NewCertificateManager()

	// Setup SSH responses for all expected commands
	sshClient.SetCommandResponse("echo 'SSH test'", "SSH test")
	sshClient.SetCommandResponse("sudo systemctl is-active etcd", "active")
	sshClient.SetCommandResponse("sudo systemctl is-active kube-apiserver", "active")
	sshClient.SetCommandResponse("sudo systemctl is-active kube-controller-manager", "active")
	sshClient.SetCommandResponse("sudo systemctl is-active kube-scheduler", "active")
	sshClient.SetCommandResponse("kubectl get nodes --kubeconfig /var/lib/kubernetes/admin.kubeconfig",
		"NAME         STATUS   ROLES    AGE   VERSION\nworker-0     Ready    <none>   1m    v1.26.0\nworker-1     Ready    <none>   1m    v1.26.0")
	sshClient.SetCommandResponse("kubectl get pods -n kube-system --kubeconfig /var/lib/kubernetes/admin.kubeconfig",
		"NAME                      READY   STATUS    RESTARTS   AGE\ncoredns-xxx               2/2     Running   0          1m")
	sshClient.SetCommandResponse("kubectl get deployment test-deployment --kubeconfig /var/lib/kubernetes/admin.kubeconfig",
		"NAME              READY   UP-TO-DATE   AVAILABLE   AGE\ntest-deployment   2/2     2            2           1m")

	cm := NewClusterManager(loadedConfig, logger, sshClient, certManager, progress)

	t.Run("Complete Cluster Lifecycle", func(t *testing.T) {
		ctx := context.Background()

		// Setup cluster
		if err := cm.SetupCluster(ctx); err != nil {
			t.Fatalf("Cluster setup failed: %v", err)
		}

		// Verify cluster status
		status, err := cm.GetClusterStatus(ctx)
		if err != nil {
			t.Fatalf("Failed to get cluster status: %v", err)
		}

		if !strings.Contains(status.Nodes, "Ready") {
			t.Error("Cluster nodes are not ready")
		}

		// Destroy cluster
		if err := cm.DestroyCluster(ctx); err != nil {
			t.Fatalf("Cluster destruction failed: %v", err)
		}

		// Verify logs contain all phases
		logs := logger.GetLogs()
		logStr := strings.Join(logs, " ")
		
		expectedLogMessages := []string{
			"Generating certificates",
			"Creating configurations", 
			"Setting up control plane",
			"Setting up worker nodes",
			"Setting up networking",
			"Validating cluster",
			"Cluster destroyed successfully",
		}

		for _, msg := range expectedLogMessages {
			if !strings.Contains(logStr, msg) {
				t.Errorf("Expected log message '%s' not found", msg)
			}
		}

		// Verify progress reporting
		steps := progress.steps
		if len(steps) < 6 { // At least 6 main phases
			t.Errorf("Expected at least 6 progress steps, got %d", len(steps))
		}

		// Verify comprehensive command execution
		commands := sshClient.GetExecutedCommands()
		if len(commands) < 50 { // Expect many commands for full setup
			t.Errorf("Expected many commands to be executed, got %d", len(commands))
		}

		t.Logf("End-to-end test completed successfully with %d commands executed", len(commands))
	})
}