package redisdump

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
)

type TlsHandler struct {
	caCertPath string
	certPath   string
	keyPath    string
}

func NewTlsHandler(caCertPath, certPath, keyPath string) *TlsHandler {
	if caCertPath == "" && certPath == "" && keyPath == "" {
		return nil
	}

	return &TlsHandler{
		caCertPath: caCertPath,
		certPath:   certPath,
		keyPath:    keyPath,
	}
}

func tlsConfig(tlsHandler *TlsHandler) (*tls.Config, error) {
	if tlsHandler == nil {
		return nil, nil
	}

	certPool := x509.NewCertPool()
	// ca cert is optional
	if tlsHandler.caCertPath != "" {
		pem, err := ioutil.ReadFile(tlsHandler.caCertPath)
		if err != nil {
			return nil, fmt.Errorf("connectionpool: unable to open CA certs: %v", err)
		}

		if !certPool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("connectionpool: failed parsing or CA certs")
		}
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{},
		RootCAs:      certPool,
	}

	if tlsHandler.certPath != "" && tlsHandler.keyPath != "" {
		cert, err := tls.LoadX509KeyPair(tlsHandler.certPath, tlsHandler.keyPath)
		if err != nil {
			return nil, err
		}
		tlsCfg.Certificates = append(tlsCfg.Certificates, cert)
	}

	return tlsCfg, nil
}
