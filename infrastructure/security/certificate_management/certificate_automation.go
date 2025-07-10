package certificate_management

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CertificateManager handles all certificate operations for IAROS
type CertificateManager struct {
	vaultClient    *api.Client
	k8sClient      kubernetes.Interface
	caPrivateKey   *rsa.PrivateKey
	caCertificate  *x509.Certificate
	certStore      string
	renewalWindow  time.Duration
	keySize        int
}

// CertificateRequest defines a certificate request
type CertificateRequest struct {
	CommonName         string
	Organization       []string
	OrganizationalUnit []string
	Country            []string
	Province           []string
	Locality           []string
	DNSNames           []string
	IPAddresses        []net.IP
	ValidityPeriod     time.Duration
	KeyUsage           x509.KeyUsage
	ExtKeyUsage        []x509.ExtKeyUsage
	IsCA               bool
	ServiceName        string
	Namespace          string
}

// Certificate represents a managed certificate
type Certificate struct {
	CommonName    string
	SerialNumber  *big.Int
	NotBefore     time.Time
	NotAfter      time.Time
	Certificate   *x509.Certificate
	PrivateKey    *rsa.PrivateKey
	PEMCert       []byte
	PEMKey        []byte
	Fingerprint   string
	Status        string
	RenewalDate   time.Time
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(vaultClient *api.Client, k8sClient kubernetes.Interface, certStore string) (*CertificateManager, error) {
	cm := &CertificateManager{
		vaultClient:   vaultClient,
		k8sClient:     k8sClient,
		certStore:     certStore,
		renewalWindow: 30 * 24 * time.Hour, // 30 days before expiry
		keySize:       4096,
	}

	// Initialize root CA if it doesn't exist
	if err := cm.initializeRootCA(); err != nil {
		return nil, fmt.Errorf("failed to initialize root CA: %v", err)
	}

	return cm, nil
}

// initializeRootCA creates or loads the root CA
func (cm *CertificateManager) initializeRootCA() error {
	caKeyPath := filepath.Join(cm.certStore, "ca-key.pem")
	caCertPath := filepath.Join(cm.certStore, "ca-cert.pem")

	// Check if CA exists
	if _, err := os.Stat(caKeyPath); os.IsNotExist(err) {
		log.Println("Creating new Root CA for IAROS")
		return cm.createRootCA()
	}

	// Load existing CA
	return cm.loadRootCA()
}

// createRootCA creates a new root certificate authority
func (cm *CertificateManager) createRootCA() error {
	// Generate CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, cm.keySize)
	if err != nil {
		return fmt.Errorf("failed to generate CA private key: %v", err)
	}

	// Create CA certificate template
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:       []string{"IAROS Platform"},
			OrganizationalUnit: []string{"Security"},
			Country:            []string{"US"},
			Province:           []string{"NY"},
			Locality:           []string{"New York"},
			CommonName:         "IAROS Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
	}

	// Self-sign the CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %v", err)
	}

	// Parse the certificate
	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %v", err)
	}

	cm.caPrivateKey = caKey
	cm.caCertificate = caCert

	// Save CA certificate and key
	if err := cm.saveCACertificate(); err != nil {
		return fmt.Errorf("failed to save CA certificate: %v", err)
	}

	// Store in Vault
	if err := cm.storeCertificateInVault("root-ca", caCert, caKey); err != nil {
		log.Printf("Warning: Failed to store CA in Vault: %v", err)
	}

	log.Println("Successfully created IAROS Root CA")
	return nil
}

// loadRootCA loads existing root CA from storage
func (cm *CertificateManager) loadRootCA() error {
	caKeyPath := filepath.Join(cm.certStore, "ca-key.pem")
	caCertPath := filepath.Join(cm.certStore, "ca-cert.pem")

	// Load CA private key
	keyPEM, err := os.ReadFile(caKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read CA private key: %v", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil || keyBlock.Type != "RSA PRIVATE KEY" {
		return fmt.Errorf("invalid CA private key format")
	}

	caKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA private key: %v", err)
	}

	// Load CA certificate
	certPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %v", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return fmt.Errorf("invalid CA certificate format")
	}

	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %v", err)
	}

	cm.caPrivateKey = caKey
	cm.caCertificate = caCert

	log.Println("Successfully loaded IAROS Root CA")
	return nil
}

// IssueCertificate issues a new certificate based on the request
func (cm *CertificateManager) IssueCertificate(req *CertificateRequest) (*Certificate, error) {
	// Generate private key for the certificate
	privateKey, err := rsa.GenerateKey(rand.Reader, cm.keySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			CommonName:         req.CommonName,
			Organization:       req.Organization,
			OrganizationalUnit: req.OrganizationalUnit,
			Country:            req.Country,
			Province:           req.Province,
			Locality:           req.Locality,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(req.ValidityPeriod),
		KeyUsage:     req.KeyUsage,
		ExtKeyUsage:  req.ExtKeyUsage,
		IPAddresses:  req.IPAddresses,
		DNSNames:     req.DNSNames,
		IsCA:         req.IsCA,
	}

	// Set default key usage if not specified
	if template.KeyUsage == 0 {
		template.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	}

	if len(template.ExtKeyUsage) == 0 {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, cm.caCertificate, &privateKey.PublicKey, cm.caPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Convert to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	certificate := &Certificate{
		CommonName:   req.CommonName,
		SerialNumber: cert.SerialNumber,
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		Certificate:  cert,
		PrivateKey:   privateKey,
		PEMCert:      certPEM,
		PEMKey:       keyPEM,
		Fingerprint:  fmt.Sprintf("%x", cert.Raw),
		Status:       "active",
		RenewalDate:  cert.NotAfter.Add(-cm.renewalWindow),
	}

	// Store certificate
	if err := cm.storeCertificate(certificate); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %v", err)
	}

	// Create Kubernetes secret if service details provided
	if req.ServiceName != "" && req.Namespace != "" {
		if err := cm.createKubernetesSecret(req.ServiceName, req.Namespace, certificate); err != nil {
			log.Printf("Warning: Failed to create Kubernetes secret: %v", err)
		}
	}

	log.Printf("Successfully issued certificate for %s", req.CommonName)
	return certificate, nil
}

// IssueServiceCertificate issues a certificate for a specific IAROS service
func (cm *CertificateManager) IssueServiceCertificate(serviceName, namespace string) (*Certificate, error) {
	req := &CertificateRequest{
		CommonName: fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace),
		Organization: []string{"IAROS Platform"},
		OrganizationalUnit: []string{"Microservices"},
		Country:    []string{"US"},
		DNSNames: []string{
			serviceName,
			fmt.Sprintf("%s.%s", serviceName, namespace),
			fmt.Sprintf("%s.%s.svc", serviceName, namespace),
			fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace),
		},
		ValidityPeriod: 365 * 24 * time.Hour, // 1 year
		ServiceName:    serviceName,
		Namespace:      namespace,
	}

	return cm.IssueCertificate(req)
}

// RenewCertificate renews an existing certificate
func (cm *CertificateManager) RenewCertificate(commonName string) (*Certificate, error) {
	// Load existing certificate
	existingCert, err := cm.loadCertificate(commonName)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing certificate: %v", err)
	}

	// Create renewal request based on existing certificate
	req := &CertificateRequest{
		CommonName:         existingCert.Certificate.Subject.CommonName,
		Organization:       existingCert.Certificate.Subject.Organization,
		OrganizationalUnit: existingCert.Certificate.Subject.OrganizationalUnit,
		Country:            existingCert.Certificate.Subject.Country,
		Province:           existingCert.Certificate.Subject.Province,
		Locality:           existingCert.Certificate.Subject.Locality,
		DNSNames:           existingCert.Certificate.DNSNames,
		IPAddresses:        existingCert.Certificate.IPAddresses,
		ValidityPeriod:     365 * 24 * time.Hour, // 1 year
		KeyUsage:           existingCert.Certificate.KeyUsage,
		ExtKeyUsage:        existingCert.Certificate.ExtKeyUsage,
	}

	// Issue new certificate
	newCert, err := cm.IssueCertificate(req)
	if err != nil {
		return nil, fmt.Errorf("failed to issue renewed certificate: %v", err)
	}

	// Mark old certificate as revoked
	existingCert.Status = "revoked"
	if err := cm.storeCertificate(existingCert); err != nil {
		log.Printf("Warning: Failed to update old certificate status: %v", err)
	}

	log.Printf("Successfully renewed certificate for %s", commonName)
	return newCert, nil
}

// CheckAndRenewCertificates checks all certificates and renews those nearing expiry
func (cm *CertificateManager) CheckAndRenewCertificates() error {
	certificates, err := cm.listCertificates()
	if err != nil {
		return fmt.Errorf("failed to list certificates: %v", err)
	}

	for _, cert := range certificates {
		if time.Now().After(cert.RenewalDate) && cert.Status == "active" {
			log.Printf("Certificate %s is due for renewal", cert.CommonName)
			
			_, err := cm.RenewCertificate(cert.CommonName)
			if err != nil {
				log.Printf("Failed to renew certificate %s: %v", cert.CommonName, err)
				continue
			}
		}
	}

	return nil
}

// RevokeCertificate revokes a certificate
func (cm *CertificateManager) RevokeCertificate(commonName string) error {
	cert, err := cm.loadCertificate(commonName)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %v", err)
	}

	cert.Status = "revoked"
	if err := cm.storeCertificate(cert); err != nil {
		return fmt.Errorf("failed to update certificate status: %v", err)
	}

	// Add to CRL
	if err := cm.addToCRL(cert.Certificate); err != nil {
		log.Printf("Warning: Failed to add certificate to CRL: %v", err)
	}

	log.Printf("Successfully revoked certificate for %s", commonName)
	return nil
}

// ValidateCertificate validates a certificate against the CA
func (cm *CertificateManager) ValidateCertificate(cert *x509.Certificate) error {
	roots := x509.NewCertPool()
	roots.AddCert(cm.caCertificate)

	opts := x509.VerifyOptions{
		Roots: roots,
	}

	_, err := cert.Verify(opts)
	return err
}

// Helper methods

// saveCACertificate saves the CA certificate and key to storage
func (cm *CertificateManager) saveCACertificate() error {
	// Ensure directory exists
	if err := os.MkdirAll(cm.certStore, 0700); err != nil {
		return err
	}

	// Save CA certificate
	caCertPath := filepath.Join(cm.certStore, "ca-cert.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cm.caCertificate.Raw,
	})
	if err := os.WriteFile(caCertPath, certPEM, 0644); err != nil {
		return err
	}

	// Save CA private key
	caKeyPath := filepath.Join(cm.certStore, "ca-key.pem")
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(cm.caPrivateKey),
	})
	if err := os.WriteFile(caKeyPath, keyPEM, 0600); err != nil {
		return err
	}

	return nil
}

// storeCertificate stores a certificate to the certificate store
func (cm *CertificateManager) storeCertificate(cert *Certificate) error {
	certDir := filepath.Join(cm.certStore, "certificates", cert.CommonName)
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return err
	}

	// Save certificate
	certPath := filepath.Join(certDir, "cert.pem")
	if err := os.WriteFile(certPath, cert.PEMCert, 0644); err != nil {
		return err
	}

	// Save private key
	keyPath := filepath.Join(certDir, "key.pem")
	if err := os.WriteFile(keyPath, cert.PEMKey, 0600); err != nil {
		return err
	}

	// Store in Vault if available
	if cm.vaultClient != nil {
		if err := cm.storeCertificateInVault(cert.CommonName, cert.Certificate, cert.PrivateKey); err != nil {
			log.Printf("Warning: Failed to store certificate in Vault: %v", err)
		}
	}

	return nil
}

// loadCertificate loads a certificate from storage
func (cm *CertificateManager) loadCertificate(commonName string) (*Certificate, error) {
	certDir := filepath.Join(cm.certStore, "certificates", commonName)
	
	certPath := filepath.Join(certDir, "cert.pem")
	keyPath := filepath.Join(certDir, "key.pem")

	// Load certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("invalid certificate format")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, err
	}

	// Load private key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("invalid private key format")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	certificate := &Certificate{
		CommonName:   commonName,
		SerialNumber: cert.SerialNumber,
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		Certificate:  cert,
		PrivateKey:   privateKey,
		PEMCert:      certPEM,
		PEMKey:       keyPEM,
		Fingerprint:  fmt.Sprintf("%x", cert.Raw),
		Status:       "active",
		RenewalDate:  cert.NotAfter.Add(-cm.renewalWindow),
	}

	return certificate, nil
}

// listCertificates returns all certificates in the store
func (cm *CertificateManager) listCertificates() ([]*Certificate, error) {
	certDir := filepath.Join(cm.certStore, "certificates")
	entries, err := os.ReadDir(certDir)
	if err != nil {
		return nil, err
	}

	var certificates []*Certificate
	for _, entry := range entries {
		if entry.IsDir() {
			cert, err := cm.loadCertificate(entry.Name())
			if err != nil {
				log.Printf("Warning: Failed to load certificate %s: %v", entry.Name(), err)
				continue
			}
			certificates = append(certificates, cert)
		}
	}

	return certificates, nil
}

// storeCertificateInVault stores certificate in HashiCorp Vault
func (cm *CertificateManager) storeCertificateInVault(name string, cert *x509.Certificate, key *rsa.PrivateKey) error {
	if cm.vaultClient == nil {
		return fmt.Errorf("vault client not configured")
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	data := map[string]interface{}{
		"certificate": string(certPEM),
		"private_key": string(keyPEM),
		"common_name": cert.Subject.CommonName,
		"not_before":  cert.NotBefore.Format(time.RFC3339),
		"not_after":   cert.NotAfter.Format(time.RFC3339),
		"serial":      cert.SerialNumber.String(),
	}

	path := fmt.Sprintf("secret/certificates/%s", name)
	_, err := cm.vaultClient.Logical().Write(path, data)
	return err
}

// createKubernetesSecret creates a Kubernetes TLS secret
func (cm *CertificateManager) createKubernetesSecret(serviceName, namespace string, cert *Certificate) error {
	secretName := fmt.Sprintf("%s-tls", serviceName)
	
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":                   serviceName,
				"managed-by":           "iaros-cert-manager",
				"certificate.iaros.io": "true",
			},
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": cert.PEMCert,
			"tls.key": cert.PEMKey,
		},
	}

	_, err := cm.k8sClient.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil && strings.Contains(err.Error(), "already exists") {
		// Update existing secret
		_, err = cm.k8sClient.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	}

	return err
}

// addToCRL adds a certificate to the Certificate Revocation List
func (cm *CertificateManager) addToCRL(cert *x509.Certificate) error {
	// Implementation for CRL management
	// This would typically involve updating a CRL file and publishing it
	log.Printf("Adding certificate %s to CRL", cert.Subject.CommonName)
	return nil
}

// GetTLSConfig returns a TLS configuration for the service
func (cm *CertificateManager) GetTLSConfig(serviceName string) (*tls.Config, error) {
	cert, err := cm.loadCertificate(fmt.Sprintf("%s.iaros-prod.svc.cluster.local", serviceName))
	if err != nil {
		return nil, fmt.Errorf("failed to load service certificate: %v", err)
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{cert.Certificate.Raw},
		PrivateKey:  cert.PrivateKey,
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
		},
	}

	return config, nil
} 