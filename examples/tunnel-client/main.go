package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"net"

	"github.com/oatcode/portal"
)

func tunnelDialAndServe(address string, tlsConfig *tls.Config) {
	log.Printf("Tunnel client...")
	c, err := tls.Dial("tcp", address, tlsConfig)
	if err != nil {
		log.Fatal("Dial: ", err)
	}
	defer c.Close()
	log.Print("Tunnel client connected")

	// connection channel unused in this sample
	cch := make(chan net.Conn)
	portal.TunnelServe(c, cch)
}

func loadCert(certFile string, keyFile string, trustFile string) *tls.Config {
	// Load cert and key
	cer, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}
	// Load trust cert
	pemCerts, err := ioutil.ReadFile(trustFile)
	if err != nil {
		log.Fatal(err)
	}
	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(pemCerts)
	return &tls.Config{
		RootCAs:      rootCAs,
		Certificates: []tls.Certificate{cer},
	}
}

func main() {
	var address string
	var certFile string
	var keyFile string
	var trustFile string
	flag.StringVar(&address, "address", "", "Address [<server-ip>]:<server-port>")
	flag.StringVar(&certFile, "cert", "", "TLS certificate filename")
	flag.StringVar(&keyFile, "key", "", "TLS certificate key filename")
	flag.StringVar(&trustFile, "trust", "", "TLS trusted server certificates filename")
	flag.Parse()

	tlsConfig := loadCert(certFile, keyFile, trustFile)

	portal.Logf = log.Printf
	tunnelDialAndServe(address, tlsConfig)
}
