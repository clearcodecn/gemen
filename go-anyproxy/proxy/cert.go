package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"io/ioutil"
	"math/big"
	"net"
	"sync"
	"time"
)

var (
	certManager *lru.Cache

	rootCA  *x509.Certificate
	rootKey interface{}

	rootOnce sync.Once
)

func init() {
	certManager, _ = lru.New(1024)
}

func loadCa(rootCaPath string, rootKeyPath string) (*x509.Certificate, interface{}, error) {
	var err error
	rootOnce.Do(func() {
		var ca, key []byte
		ca, err = ioutil.ReadFile(rootCaPath)
		if err != nil {
			return
		}
		key, err = ioutil.ReadFile(rootKeyPath)
		if err != nil {
			return
		}
		var block *pem.Block
		block, _ = pem.Decode(ca)
		rootCA, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return
		}
		block, _ = pem.Decode(key)
		rootKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return
		}
	})

	if err != nil {
		return nil, nil, err
	}
	return rootCA, rootKey, nil
}

func generateTLSByHost(host string) (*tls.Config, error) {
	if rootCA == nil || rootKey == nil {
		panic("rootCA or rootKey is nil")
	}
	if h, _, _ := net.SplitHostPort(host); h != "" {
		host = h
	}
	if v, ok := certManager.Get(host); ok {
		return v.(*tls.Config), nil
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{AppName},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	if ip := net.ParseIP(host); err != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		template.DNSNames = append(template.DNSNames, host)
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, rootCA, &priv.PublicKey, rootKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create certificate")
	}
	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	}
	serverCert := pem.EncodeToMemory(certBlock)

	keyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	}
	serverKey := pem.EncodeToMemory(keyBlock)

	conf, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load x509 key pair")
	}

	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{conf},
	}

	certManager.Add(host, tlsConf)

	return tlsConf, nil
}
