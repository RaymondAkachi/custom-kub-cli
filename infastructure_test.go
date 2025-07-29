// infrastructure_test.go - Test AWS infrastructure connectivity
package main

import (
	"context"
	"fmt"

	"github.com/RaymondAkachi/custom-kub-cli/k8s/clustersetup"
)

// TestInfrastructure validates AWS infrastructure setup
func TestInfrastructure(configPath string) error {
	config, err := clustersetup.LoadClusterConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	sshClient, err := clustersetup.NewSSHClient(config.SSHUser, config.SSHKey)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}

	ctx := context.Background()
	
	// Test SSH connectivity to all nodes
	fmt.Println("Testing SSH connectivity...")
	nodes := append([]clustersetup.Node{config.Controller}, config.Workers...)
	
	for _, node := range nodes {
		fmt.Printf("Testing connection to %s (%s)...", node.Name, node.IPAddress)
		
		output, err := sshClient.ExecuteCommand(ctx, node.IPAddress, "echo 'SSH test successful'")
		if err != nil {
			return fmt.Errorf("SSH test failed for %s: %w", node.Name, err)
		}
		
		if output != "SSH test successful\n" {
			return fmt.Errorf("unexpected output from %s: %s", node.Name, output)
		}
		
		fmt.Printf(" ✅\n")
	}

	// Test sudo privileges
	fmt.Println("Testing sudo privileges...")
	for _, node := range nodes {
		fmt.Printf("Testing sudo on %s...", node.Name)
		
		_, err := sshClient.ExecuteCommand(ctx, node.IPAddress, "sudo whoami")
		if err != nil {
			return fmt.Errorf("sudo test failed for %s: %w", node.Name, err)
		}
		
		fmt.Printf(" ✅\n")
	}

	// Test network connectivity between nodes
	fmt.Println("Testing inter-node connectivity...")
	for _, node := range nodes {
		for _, target := range nodes {
			if node.Name != target.Name {
				fmt.Printf("Testing %s -> %s...", node.Name, target.Name)
				
				pingCmd := fmt.Sprintf("ping -c 1 -W 5 %s", target.IPAddress)
				_, err := sshClient.ExecuteCommand(ctx, node.IPAddress, pingCmd)
				if err != nil {
					return fmt.Errorf("network connectivity test failed %s -> %s: %w", node.Name, target.Name, err)
				}
				
				fmt.Printf(" ✅\n")
			}
		}
	}

	// Test internet connectivity (for downloading binaries)
	fmt.Println("Testing internet connectivity...")
	testCmd := "curl -s -I https://storage.googleapis.com/kubernetes-release/release/stable.txt"
	_, err = sshClient.ExecuteCommand(ctx, config.Controller.IPAddress, testCmd)
	if err != nil {
		return fmt.Errorf("internet connectivity test failed: %w", err)
	}
	fmt.Println("Internet connectivity ✅")

	fmt.Println("\n🎉 All infrastructure tests passed!")
	return nil
}

// func main() {
// 	if len(os.Args) < 2 {
// 		log.Fatal("Usage: go run infrastructure_test.go <config-file>")
// 	}
	
// 	configPath := os.Args[1]
// 	if err := TestInfrastructure(configPath); err != nil {
// 		log.Fatalf("Infrastructure test failed: %v", err)
// 	}
// }