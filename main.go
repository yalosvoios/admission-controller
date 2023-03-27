package main

import (
	"context"
	"net/http"

	"k8s.io/klog/v2"
)

func main() {
	klog.InitFlags(nil)
	logger := klog.FromContext(context.Background())
	logger.Info("Simple admission controller")

	handler := newHandler()
	if handler == nil {
		klog.Fatalf("Cannot instanciate the handler")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.Serve(w, r)
	})

	certs := readCertificates()
	if certs == nil {
		klog.Fatalf("Cannot read certificates")
	}

	server := &http.Server{
		Addr:      ":8443", //TODO: parameter here
		TLSConfig: configTLS(certs.cert, certs.key),
	}

	if err := server.ListenAndServeTLS("", ""); err != nil {
		klog.Fatalf("HTTPS Error: %s", err)
	}
}
