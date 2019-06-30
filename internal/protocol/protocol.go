package protocol

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
)

var (
	crt = "certs/cert.pem"
	key = "certs/key.pem"
)

// SetKeyAndCertPaths initializes path to private key and certificate
func SetKeyAndCertPaths(keyPath, certPath string) {
	if strings.Trim(keyPath, " ") != "" {
		crt = certPath
	}
	if strings.Trim(certPath, " ") != "" {
		key = keyPath
	}
}

// GetCert returns a certificate pair, pool and an error
func GetCert() (*tls.Certificate, *x509.CertPool, error) {
	serverCrt, err := ioutil.ReadFile(crt)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't read file: %s", err)
	}
	serverKey, err := ioutil.ReadFile(key)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't read file: %s", err)
	}

	cert, err := tls.X509KeyPair(serverCrt, serverKey)
	if err != nil {
		return nil, nil, fmt.Errorf("could not load server key pair: %s", err)
	}

	cp := x509.NewCertPool()
	ok := cp.AppendCertsFromPEM(serverCrt)
	if !ok {
		return nil, nil, errors.New("bad certs")
	}

	return &cert, cp, nil
}

// ClientTLS creates a tls config object for client
func ClientTLS() (*tls.Config, error) {
	cert, certPool, err := GetCert()
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		RootCAs:            certPool,
		Certificates:       []tls.Certificate{*cert},
		InsecureSkipVerify: true,
	}

	return tlsConfig, nil
}

// GRPCServerTLS creates a tls config object for grpc server
func GRPCServerTLS() (*tls.Config, error) {
	cert, certPool, err := GetCert()
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		ClientAuth:         tls.VerifyClientCertIfGiven,
		Certificates:       []tls.Certificate{*cert},
		ClientCAs:          certPool,
		InsecureSkipVerify: true,
	}

	return tlsConfig, nil
}

// HTTPServerTLS creates a tls config object for http server
func HTTPServerTLS() (*tls.Config, error) {
	cert, certPool, err := GetCert()
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		ClientAuth:   tls.VerifyClientCertIfGiven,
		Certificates: []tls.Certificate{*cert},
		ClientCAs:    certPool,
		NextProtos:   []string{"h2"},
	}

	return tlsConfig, nil
}
