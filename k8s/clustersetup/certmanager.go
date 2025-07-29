// Package clustersetup provides automation for setting up Kubernetes clusters.
// certmanager.go implements certificate generation using CFSSL.
package clustersetup

import (
	"fmt"
	"os"
	"path/filepath"

	"encoding/pem"

	cfssl_config "github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/cloudflare/cfssl/initca"
	"github.com/cloudflare/cfssl/signer"
	"github.com/cloudflare/cfssl/signer/local"
)

// RealCertificateManager implements the CertificateManager interface using CFSSL.
type RealCertificateManager struct{}

// NewCertificateManager creates a new RealCertificateManager.
func NewCertificateManager() *RealCertificateManager {
	return &RealCertificateManager{}
}

// GenerateCA generates a CA certificate and key.
func (cm *RealCertificateManager) GenerateCA(workDir string, config CertificateConfig) error {
	req := &csr.CertificateRequest{
		CN:         config.Organization,
		Names: []csr.Name{{
			C:  config.Country,
			ST: config.State,
			L:  config.City,
			O:  config.Organization,
			OU: config.OrganizationalUnit,
		}},
		KeyRequest: &csr.KeyRequest{A: "ecdsa", S: 256},
		CA: &csr.CAConfig{
			PathLength: 1,
			Expiry:     fmt.Sprintf("%dh", config.ValidityDays*24),
		},
	}
	cert, _, key, err := initca.New(req)
	if err != nil {
		return fmt.Errorf("failed to generate CA: %w", err)
	}

	if err := os.WriteFile(filepath.Join(workDir, "ca.pem"), cert, 0644); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "ca-key.pem"), key, 0600); err != nil {
		return fmt.Errorf("failed to write CA key: %w", err)
	}
	
	// Create CA config file
	caConfigContent := `{
		"signing": {
			"default": {
				"expiry": "8760h"
			},
			"profiles": {
				"kubernetes": {
					"usages": ["signing", "key encipherment", "server auth", "client auth"],
					"expiry": "8760h"
				}
			}
		}
	}`
	
	if err := os.WriteFile(filepath.Join(workDir, "ca-config.json"), []byte(caConfigContent), 0644); err != nil {
		return fmt.Errorf("failed to write CA config: %w", err)
	}
	return nil
}

// GenerateClientCert generates a client certificate.
func (cm *RealCertificateManager) GenerateClientCert(workDir, name string, config CertificateConfig) error {
	req := &csr.CertificateRequest{
		CN: name,
		Names: []csr.Name{{
			C:  config.Country,
			ST: config.State,
			L:  config.City,
			O:  config.Organization,
			OU: config.OrganizationalUnit,
		}},
		KeyRequest: &csr.KeyRequest{A: "ecdsa", S: 256},
	}
	
	// Generate CSR and private key
	generator := &csr.Generator{Validator: nil}
	csrBytes, key, err := generator.ProcessRequest(req)
	if err != nil {
		return fmt.Errorf("failed to generate CSR for %s: %w", name, err)
	}

	// Encode CSR to PEM
	pemCSR := &pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	}
	pemCSRBytes := pem.EncodeToMemory(pemCSR)
	if pemCSRBytes == nil {
		return fmt.Errorf("failed to encode CSR to PEM for %s", name)
	}

	// Load CA config
	caConfigBytes, err := os.ReadFile(filepath.Join(workDir, "ca-config.json"))
	if err != nil {
		return fmt.Errorf("failed to read CA config: %w", err)
	}
	
	caConfig, err := cfssl_config.LoadConfig(caConfigBytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA config: %w", err)
	}

	// Load and parse CA certificate and key
	caCertBytes, err := os.ReadFile(filepath.Join(workDir, "ca.pem"))
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}
	caKeyBytes, err := os.ReadFile(filepath.Join(workDir, "ca-key.pem"))
	if err != nil {
		return fmt.Errorf("failed to read CA key: %w", err)
	}

	cert, err := helpers.ParseCertificatePEM(caCertBytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}
	privateKey, err := helpers.ParsePrivateKeyPEM(caKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA key: %w", err)
	}

	// Create signer
	s, err := local.NewSigner(privateKey, cert, signer.DefaultSigAlgo(privateKey), caConfig.Signing)
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	// Sign the certificate
	signReq := signer.SignRequest{
		Request: string(pemCSRBytes),
		Profile: "kubernetes",
	}
	
	certBytes, err := s.Sign(signReq)
	if err != nil {
		return fmt.Errorf("failed to sign certificate for %s: %w", name, err)
	}

	// Encode certificate to PEM format
	pemCert := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}
	pemCertBytes := pem.EncodeToMemory(pemCert)
	if pemCSRBytes == nil {
		return fmt.Errorf("failed to encode certificate to PEM for %s", name)
	}

	// Write certificate and key files
	if err := os.WriteFile(filepath.Join(workDir, name+".pem"), pemCertBytes, 0644); err != nil {
		return fmt.Errorf("failed to write certificate for %s: %w", name, err)
	}
	if err := os.WriteFile(filepath.Join(workDir, name+"-key.pem"), key, 0600); err != nil {
		return fmt.Errorf("failed to write key for %s: %w", name, err)
	}
	return nil
}

// GenerateServerCert generates a server certificate.
func (cm *RealCertificateManager) GenerateServerCert(workDir, name string, hosts []string, config CertificateConfig) error {
	req := &csr.CertificateRequest{
		CN: name,
		Names: []csr.Name{{
			C:  config.Country,
			ST: config.State,
			L:  config.City,
			O:  config.Organization,
			OU: config.OrganizationalUnit,
		}},
		KeyRequest: &csr.KeyRequest{A: "ecdsa", S: 256},
		Hosts:      hosts,
	}
	
	// Generate CSR and private key
	generator := &csr.Generator{Validator: nil}
	csrBytes, key, err := generator.ProcessRequest(req)
	if err != nil {
		return fmt.Errorf("failed to generate CSR for %s: %w", name, err)
	}

	// Encode CSR to PEM
	pemCSR := &pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	}
	pemCSRBytes := pem.EncodeToMemory(pemCSR)
	if pemCSRBytes == nil {
		return fmt.Errorf("failed to encode CSR to PEM for %s", name)
	}

	// Load CA config
	caConfigBytes, err := os.ReadFile(filepath.Join(workDir, "ca-config.json"))
	if err != nil {
		return fmt.Errorf("failed to read CA config: %w", err)
	}
	
	caConfig, err := cfssl_config.LoadConfig(caConfigBytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA config: %w", err)
	}

	// Load and parse CA certificate and key
	caCertBytes, err := os.ReadFile(filepath.Join(workDir, "ca.pem"))
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}
	caKeyBytes, err := os.ReadFile(filepath.Join(workDir, "ca-key.pem"))
	if err != nil {
		return fmt.Errorf("failed to read CA key: %w", err)
	}

	cert, err := helpers.ParseCertificatePEM(caCertBytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}
	privateKey, err := helpers.ParsePrivateKeyPEM(caKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA key: %w", err)
	}

	// Create signer
	s, err := local.NewSigner(privateKey, cert, signer.DefaultSigAlgo(privateKey), caConfig.Signing)
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	// Sign the certificate
	signReq := signer.SignRequest{
		Request: string(pemCSRBytes),
		Profile: "kubernetes",
	}
	
	certBytes, err := s.Sign(signReq)
	if err != nil {
		return fmt.Errorf("failed to sign certificate for %s: %w", name, err)
	}

	// Encode certificate to PEM format
	pemCert := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}
	pemCertBytes := pem.EncodeToMemory(pemCert)
	if pemCertBytes == nil {
		return fmt.Errorf("failed to encode certificate to PEM for %s", name)
	}

	// Write certificate and key files
	if err := os.WriteFile(filepath.Join(workDir, name+".pem"), pemCertBytes, 0644); err != nil {
		return fmt.Errorf("failed to write certificate for %s: %w", name, err)
	}
	if err := os.WriteFile(filepath.Join(workDir, name+"-key.pem"), key, 0600); err != nil {
		return fmt.Errorf("failed to write key for %s: %w", name, err)
	}
	return nil
}