package redisdump

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/mediocregopher/radix/v3"
	"io/ioutil"
	"strconv"
	"time"
)

type TlsHandler struct {
	tls        bool
	caCertPath string
	certPath   string
	keyPath    string
}

func NewTlsHandler(tls bool, caCertPath, certPath, keyPath string) *TlsHandler {
	return &TlsHandler{
		tls:        tls,
		caCertPath: caCertPath,
		certPath:   certPath,
		keyPath:    keyPath,
	}
}
func NewRedisClient(redisURL string, tlsHandler *TlsHandler, redisPassword string, nWorkers int, db string) (*radix.Pool, error) {
	var tlsConfig *tls.Config
	if tlsHandler != nil {
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
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{},
			RootCAs:      certPool,
		}
		if tlsHandler.certPath != "" && tlsHandler.keyPath != "" {
			cert, err := tls.LoadX509KeyPair(tlsHandler.certPath, tlsHandler.keyPath)
			if err != nil {
				return nil, err
			}
			tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
		}
	}

	customConnFunc := func(network, addr string) (radix.Conn, error) {
		dialOpts := []radix.DialOpt{
			radix.DialTimeout(5 * time.Minute),
		}
		if redisPassword != "" {
			dialOpts = append(dialOpts, radix.DialAuthPass(redisPassword))
		}
		if tlsHandler != nil {
			dialOpts = append(dialOpts, radix.DialUseTLS(tlsConfig))
		}
		if db != "" {
			dbVal, err := strconv.Atoi(db)
			if err != nil {
				return nil, err
			}
			dialOpts = append(dialOpts, radix.DialSelectDB(dbVal))
		}
		return radix.Dial(network, addr, dialOpts...)
	}
	client, err := radix.NewPool("tcp", redisURL, nWorkers, radix.PoolConnFunc(customConnFunc))
	if err != nil {
		return nil, err
	}
	return client, nil
}
