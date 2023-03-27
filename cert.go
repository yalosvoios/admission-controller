package main

import (
	"crypto/tls"
	"io/ioutil"
	"os"

	"k8s.io/klog/v2"
)

type certificates struct {
	ca   []byte
	key  []byte
	cert []byte
}

func configTLS(serverCert, serverKey []byte) *tls.Config {
	sCert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		klog.Fatal(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}
}

func readCertificates() *certificates {
	cert := certificates{}
	var err error

	filepath := os.Getenv("ca_path")
	if len(filepath) == 0 {
		filepath = "/etc/tls/caCert.pem"
	}

	cert.ca, err = ioutil.ReadFile(filepath)
	if err != nil {
		klog.Errorf("Cannot read certificate file %s %v", filepath, err)
		return nil
	}

	filepath = os.Getenv("cert_path")
	if len(filepath) == 0 {
		filepath = "/etc/tls/serverCert.pem"
	}

	cert.cert, err = ioutil.ReadFile(filepath)
	if err != nil {
		klog.Errorf("Cannot read key file %s %v", filepath, err)
		return nil
	}

	filepath = os.Getenv("key_path")
	if len(filepath) == 0 {
		filepath = "/etc/tls/serverKey.pem"
	}

	cert.key, err = ioutil.ReadFile(filepath)
	if err != nil {
		klog.Errorf("Cannot read certificate file %s %v", filepath, err)
		return nil
	}

	return &cert
}
