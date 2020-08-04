package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
)

func loadCert(certFile string, keyFile string) *tls.Config {
	// Load cert and key
	cer, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cer},
	}
}

func main() {
	var address string
	var certFile string
	var keyFile string
	flag.StringVar(&address, "address", "", "Address [<server-ip>]:<server-port>")
	flag.StringVar(&certFile, "cert", "", "TLS certificate filename")
	flag.StringVar(&keyFile, "key", "", "TLS certificate key filename")
	flag.Parse()

	tlsConfig := loadCert(certFile, keyFile)

	l, err := tls.Listen("tcp", address, tlsConfig)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello url=%s", r.URL)
	})
	log.Fatal(http.Serve(l, nil))
}
