/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhook

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/knative/pkg/logging"
)

// Cert Template option
type CertTemplateOption struct {
	Organizations      []string
	DNSNames           []string
	IPAddress          []net.IP
	SignatureAlgorithm x509.SignatureAlgorithm
}

// Generate a default cert template option
func NewDefaultCertTemplateOption() *CertTemplateOption {
	return &CertTemplateOption{
		Organizations:      []string{"kube"},
		SignatureAlgorithm: x509.SHA256WithRSA,
	}
}

// Create the common parts of the cert. These don't change between
// the root/CA cert and the server cert.
func createCertTemplate(option *CertTemplateOption) (*x509.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.New("failed to generate serial number: " + err.Error())
	}

	return &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: option.Organizations},
		SignatureAlgorithm:    option.SignatureAlgorithm,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // valid for 1 years
		BasicConstraintsValid: true,
		DNSNames:              option.DNSNames,
		IPAddresses:           option.IPAddress,
	}, nil
}

// Create cert template suitable for CA and hence signing
func createCACertTemplate(option *CertTemplateOption) (*x509.Certificate, error) {
	rootCert, err := createCertTemplate(option)
	if err != nil {
		return nil, err
	}
	// Make it into a CA cert and change it so we can use it to sign certs
	rootCert.IsCA = true
	rootCert.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	rootCert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	return rootCert, nil
}

// Create cert template that we can use on the server for TLS
func createServerCertTemplate(option *CertTemplateOption) (*x509.Certificate, error) {
	serverCert, err := createCertTemplate(option)
	if err != nil {
		return nil, err
	}
	serverCert.KeyUsage = x509.KeyUsageDigitalSignature
	serverCert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	return serverCert, err
}

// Actually sign the cert and return things in a form that we can use later on
func createCert(template, parent *x509.Certificate, pub interface{}, parentPriv interface{}) (
	cert *x509.Certificate, certPEM []byte, err error) {

	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, pub, parentPriv)
	if err != nil {
		return
	}
	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		return
	}
	b := pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	certPEM = pem.EncodeToMemory(&b)
	return
}

func createCA(ctx context.Context, option *CertTemplateOption) (*rsa.PrivateKey, *x509.Certificate, []byte, error) {
	logger := logging.FromContext(ctx)
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		logger.Error("error generating random key", zap.Error(err))
		return nil, nil, nil, err
	}

	rootCertTmpl, err := createCACertTemplate(option)
	if err != nil {
		logger.Error("error generating CA cert", zap.Error(err))
		return nil, nil, nil, err
	}

	rootCert, rootCertPEM, err := createCert(rootCertTmpl, rootCertTmpl, &rootKey.PublicKey, rootKey)
	if err != nil {
		logger.Error("error signing the CA cert", zap.Error(err))
		return nil, nil, nil, err
	}
	return rootKey, rootCert, rootCertPEM, nil
}

// CreateCerts creates and returns a CA certificate and certificate and
// key for the server. serverKey and serverCert are used by the server
// to establish trust for clients, CA certificate is used by the
// client to verify the server authentication chain.
func CreateCerts(ctx context.Context, option *CertTemplateOption) (serverKey, serverCert, caCert []byte, err error) {
	logger := logging.FromContext(ctx)
	// First create a CA certificate and private key
	caKey, caCertificate, caCertificatePEM, err := createCA(ctx, option)
	if err != nil {
		return nil, nil, nil, err
	}

	// Then create the private key for the serving cert
	servKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		logger.Error("error generating random key", zap.Error(err))
		return nil, nil, nil, err
	}
	servCertTemplate, err := createServerCertTemplate(option)
	if err != nil {
		logger.Error("failed to create the server certificate template", zap.Error(err))
		return nil, nil, nil, err
	}

	// create a certificate which wraps the server's public key, sign it with the CA private key
	_, servCertPEM, err := createCert(servCertTemplate, caCertificate, &servKey.PublicKey, caKey)
	if err != nil {
		logger.Error("error signing server certificate template", zap.Error(err))
		return nil, nil, nil, err
	}
	servKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(servKey),
	})
	return servKeyPEM, servCertPEM, caCertificatePEM, nil
}
