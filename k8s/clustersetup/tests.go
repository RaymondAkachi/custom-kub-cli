package clustersetup

import "fmt"

func hello() {
	fmt.Println("Hello, World!")
}

// import (
// 	"os"
// 	"path/filepath"
// 	"testing"
// )

// func TestLoadClusterConfig(t *testing.T) {
// 	// Create a temporary config file
// 	configContent := `
// cluster_name: "test-cluster"
// kubernetes_version: "v1.26.0"
// etcd_version: "v3.5.9"
// containerd_version: "1.7.2"
// cni_version: "v1.3.0"
// coredns_version: "1.10.1"
// pod_cidr: "10.200.0.0/16"
// service_cidr: "10.32.0.0/24"
// cluster_dns: "10.32.0.10"
// work_dir: "/tmp/test-k8s"
// ssh_key: "~/.ssh/test.pem"
// ssh_user: "ubuntu"
// controller:
//   name: "controller-0"
//   ip_address: "10.240.0.10"
//   hostname: "controller-0"
// workers:
//   - name: "worker-0"
//     ip_address: "10.240.0.20"
//     hostname: "worker-0"
//     pod_cidr: "10.200.0.0/24"
// certificates:
//   country: "US"
//   state: "CA"
//   city: "SF"
//   organization: "Test"
//   organizational_unit: "IT"
//   validity_days: 365
// `

// 	tmpFile, err := os.CreateTemp("", "config-*.yaml")
// 	if err != nil {
// 		t.Fatalf("Failed to create temp file: %v", err)
// 	}
// 	defer os.Remove(tmpFile.Name())

// 	if _, err := tmpFile.WriteString(configContent); err != nil {
// 		t.Fatalf("Failed to write config: %v", err)
// 	}
// 	tmpFile.Close()

// 	// Test loading config
// 	config, err := LoadClusterConfig(tmpFile.Name())
// 	if err != nil {
// 		t.Fatalf("Failed to load config: %v", err)
// 	}

// 	// Validate loaded config
// 	if config.ClusterName != "test-cluster" {
// 		t.Errorf("Expected cluster name 'test-cluster', got '%s'", config.ClusterName)
// 	}
// 	if len(config.Workers) != 1 {
// 		t.Errorf("Expected 1 worker, got %d", len(config.Workers))
// 	}
// }

// func TestCertificateGeneration(t *testing.T) {
// 	workDir, err := os.MkdirTemp("", "cert-test-*")
// 	if err != nil {
// 		t.Fatalf("Failed to create temp dir: %v", err)
// 	}
// 	defer os.RemoveAll(workDir)

// 	certManager := NewCertificateManager()
// 	config := CertificateConfig{
// 		Country:            "US",
// 		State:              "CA",
// 		City:               "SF",
// 		Organization:       "Test",
// 		OrganizationalUnit: "IT",
// 		ValidityDays:       365,
// 	}

// 	// Test CA generation
// 	if err := certManager.GenerateCA(workDir, config); err != nil {
// 		t.Fatalf("CA generation failed: %v", err)
// 	}

// 	// Check if CA files were created
// 	caFiles := []string{"ca.pem", "ca-key.pem", "ca-config.json"}
// 	for _, file := range caFiles {
// 		filePath := filepath.Join(workDir, file)
// 		if _, err := os.Stat(filePath); os.IsNotExist(err) {
// 			t.Errorf("Expected file %s was not created", file)
// 		}
// 	}

// 	// Test client certificate generation
// 	if err := certManager.GenerateClientCert(workDir, "test-client", config); err != nil {
// 		t.Fatalf("Client cert generation failed: %v", err)
// 	}

// 	// Check if client cert files were created
// 	clientFiles := []string{"test-client.pem", "test-client-key.pem"}
// 	for _, file := range clientFiles {
// 		filePath := filepath.Join(workDir, file)
// 		if _, err := os.Stat(filePath); os.IsNotExist(err) {
// 			t.Errorf("Expected file %s was not created", file)
// 		}
// 	}
// }

// func TestServiceFileGeneration(t *testing.T) {
// 	config := ClusterConfig{
// 		ClusterName:       "test",
// 		PodCIDR:          "10.200.0.0/16",
// 		ServiceCIDR:      "10.32.0.0/24",
// 		ClusterDNS:       "10.32.0.10",
// 		CoreDNSVersion:   "1.10.1",
// 		Controller: Node{
// 			Name:      "controller-0",
// 			IPAddress: "10.240.0.10",
// 		},
// 	}

// 	logger := NewLogger()
// 	progress := NewProgressReporter()
// 	cm := &ClusterManager{
// 		config:   config,
// 		logger:   logger,
// 		progress: progress,
// 	}

// 	// Test etcd service generation
// 	etcdService := cm.generateEtcdService(config.Controller)
// 	if etcdService == "" {
// 		t.Error("etcd service generation returned empty string")
// 	}

// 	// Test API server service generation
// 	apiService := cm.generateAPIServerService()
// 	if apiService == "" {
// 		t.Error("API server service generation returned empty string")
// 	}

// 	// Test CoreDNS manifest generation
// 	coreDNS := cm.generateCoreDNSManifest()
// 	if coreDNS == "" {
// 		t.Error("CoreDNS manifest generation returned empty string")
// 	}
// }