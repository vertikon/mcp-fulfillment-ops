package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var (
	ErrCertificateNotFound = errors.New("certificate not found")
	ErrCertificateInvalid  = errors.New("invalid certificate")
)

// CertificateManager handles TLS certificate management
type CertificateManager interface {
	// GetTLSCertificate returns TLS certificate for server
	GetTLSCertificate() (*tls.Certificate, error)

	// GenerateSelfSignedCert generates a self-signed certificate
	GenerateSelfSignedCert(commonName string, dnsNames []string) (*tls.Certificate, error)

	// LoadCertificateFromFile loads certificate from file
	LoadCertificateFromFile(certFile, keyFile string) error

	// RotateCertificate rotates the certificate
	RotateCertificate() error

	// GetCertificateExpiry returns certificate expiration time
	GetCertificateExpiry() (time.Time, error)
}

// certificateManagerImpl implements CertificateManager
type certificateManagerImpl struct {
	tlsCert      *tls.Certificate
	certExpiry   time.Time
	rotationTTL  time.Duration
	lastRotation time.Time
	logger       *zap.Logger
}

// CertificateManagerConfig holds configuration for CertificateManager
type CertificateManagerConfig struct {
	RotationTTL time.Duration
}

// NewCertificateManager creates a new CertificateManager
func NewCertificateManager(config CertificateManagerConfig) CertificateManager {
	return &certificateManagerImpl{
		rotationTTL:  config.RotationTTL,
		lastRotation: time.Now(),
		logger:       logger.WithContext(nil),
	}
}

// GetTLSCertificate returns TLS certificate for server
func (m *certificateManagerImpl) GetTLSCertificate() (*tls.Certificate, error) {
	if m.tlsCert == nil {
		return nil, ErrCertificateNotFound
	}

	// Check if rotation is needed
	if time.Since(m.lastRotation) > m.rotationTTL {
		go m.RotateCertificate()
	}

	return m.tlsCert, nil
}

// GenerateSelfSignedCert generates a self-signed certificate
func (m *certificateManagerImpl) GenerateSelfSignedCert(commonName string, dnsNames []string) (*tls.Certificate, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		m.logger.Error("Failed to generate private key", zap.Error(err))
		return nil, err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              dnsNames,
		BasicConstraintsValid: true,
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		m.logger.Error("Failed to create certificate", zap.Error(err))
		return nil, err
	}

	// Encode certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Create TLS certificate
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		m.logger.Error("Failed to create TLS certificate", zap.Error(err))
		return nil, err
	}

	m.tlsCert = &tlsCert
	m.certExpiry = template.NotAfter
	m.lastRotation = time.Now()

	m.logger.Info("Self-signed certificate generated",
		zap.String("common_name", commonName),
		zap.Time("expiry", m.certExpiry),
	)

	return &tlsCert, nil
}

// LoadCertificateFromFile loads certificate from file
func (m *certificateManagerImpl) LoadCertificateFromFile(certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		m.logger.Error("Failed to load certificate from file",
			zap.String("cert_file", certFile),
			zap.String("key_file", keyFile),
			zap.Error(err),
		)
		return err
	}

	// Parse certificate to get expiry
	if len(cert.Certificate) > 0 {
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err == nil {
			m.certExpiry = x509Cert.NotAfter
		}
	}

	m.tlsCert = &cert
	m.lastRotation = time.Now()

	m.logger.Info("Certificate loaded from file",
		zap.String("cert_file", certFile),
		zap.Time("expiry", m.certExpiry),
	)

	return nil
}

// RotateCertificate rotates the certificate
func (m *certificateManagerImpl) RotateCertificate() error {
	if m.tlsCert == nil {
		return ErrCertificateNotFound
	}

	// Parse current certificate to get common name and DNS names
	var commonName string
	var dnsNames []string

	if len(m.tlsCert.Certificate) > 0 {
		x509Cert, err := x509.ParseCertificate(m.tlsCert.Certificate[0])
		if err == nil {
			commonName = x509Cert.Subject.CommonName
			dnsNames = x509Cert.DNSNames
		}
	}

	// Generate new certificate
	_, err := m.GenerateSelfSignedCert(commonName, dnsNames)
	if err != nil {
		m.logger.Error("Failed to rotate certificate", zap.Error(err))
		return err
	}

	m.logger.Info("Certificate rotated",
		zap.Time("new_expiry", m.certExpiry),
	)

	return nil
}

// GetCertificateExpiry returns certificate expiration time
func (m *certificateManagerImpl) GetCertificateExpiry() (time.Time, error) {
	if m.tlsCert == nil {
		return time.Time{}, ErrCertificateNotFound
	}
	return m.certExpiry, nil
}
