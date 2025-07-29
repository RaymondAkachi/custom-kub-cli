// Package k8shard provides automation for setting up Kubernetes clusters.
// sshclient.go implements SSH operations for remote command execution and file transfer.
package clustersetup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// RealSSHClient implements the SSHClient interface using golang.org/x/crypto/ssh.
type RealSSHClient struct {
	user   string
	keyPath string
}

// NewSSHClient creates a new RealSSHClient.
func NewSSHClient(user, keyPath string) (*RealSSHClient, error) {
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("SSH key file %s does not exist", keyPath)
	}
	return &RealSSHClient{user: user, keyPath: keyPath}, nil
}

// ExecuteCommand executes a command on the remote host via SSH.
func (c *RealSSHClient) ExecuteCommand(ctx context.Context, host, command string) (string, error) {
	client, err := c.createSSHClient(host)
	if err != nil {
		return "", fmt.Errorf("failed to create SSH client for %s: %w", host, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session for %s: %w", host, err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(command); err != nil {
		return "", fmt.Errorf("failed to execute command '%s' on %s: %w, stderr: %s", command, host, err, stderr.String())
	}

	return stdout.String(), nil
}

// CopyFile copies a local file to the remote host via SSH.
func (c *RealSSHClient) CopyFile(ctx context.Context, host, localPath, remotePath string) error {
	client, err := c.createSSHClient(host)
	if err != nil {
		return fmt.Errorf("failed to create SSH client for %s: %w", host, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session for %s: %w", host, err)
	}
	defer session.Close()

	fileContent, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file %s: %w", localPath, err)
	}

	pipe, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe for %s: %w", host, err)
	}

	go func() {
		defer pipe.Close()
		if _, err := pipe.Write(fileContent); err != nil {
			return
		}
	}()

	if err := session.Run(fmt.Sprintf("sudo tee %s > /dev/null", remotePath)); err != nil {
		return fmt.Errorf("failed to copy file to %s on %s: %w", remotePath, host, err)
	}

	// Create new session for permission setting
	permSession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create permission session for %s: %w", host, err)
	}
	defer permSession.Close()

	if err := permSession.Run(fmt.Sprintf("sudo chmod 644 %s", remotePath)); err != nil {
		return fmt.Errorf("failed to set permissions for %s on %s: %w", remotePath, host, err)
	}

	return nil
}

// CopyContent copies content directly to a remote file via SSH.
func (c *RealSSHClient) CopyContent(ctx context.Context, host, content, remotePath string) error {
	client, err := c.createSSHClient(host)
	if err != nil {
		return fmt.Errorf("failed to create SSH client for %s: %w", host, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session for %s: %w", host, err)
	}
	defer session.Close()

	pipe, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe for %s: %w", host, err)
	}

	go func() {
		defer pipe.Close()
		if _, err := io.WriteString(pipe, content); err != nil {
			return
		}
	}()

	if err := session.Run(fmt.Sprintf("sudo tee %s > /dev/null", remotePath)); err != nil {
		return fmt.Errorf("failed to copy content to %s on %s: %w", remotePath, host, err)
	}

	// Create new session for permission setting
	permSession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create permission session for %s: %w", host, err)
	}
	defer permSession.Close()

	if err := permSession.Run(fmt.Sprintf("sudo chmod 644 %s", remotePath)); err != nil {
		return fmt.Errorf("failed to set permissions for %s on %s: %w", remotePath, host, err)
	}

	return nil
}

// createSSHClient creates an SSH client for the specified host.
func (c *RealSSHClient) createSSHClient(host string) (*ssh.Client, error) {
	// Ensure host has port
	if !strings.Contains(host, ":") {
		host = host + ":22"
	}
	
	key, err := os.ReadFile(c.keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key %s: %w", c.keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key %s: %w", c.keyPath, err)
	}

	config := &ssh.ClientConfig{
		User: c.user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:        30 * time.Second, // Add timeout
	}

	return ssh.Dial("tcp", host, config)
}