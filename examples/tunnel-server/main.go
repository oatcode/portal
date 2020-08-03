/*
openssl req -x509 -nodes -newkey rsa:2048 -sha256 -keyout server.key -out server.crt -subj "/C=US/ST=California/L=San Jose/O=Example/OU=Developer/CN=localhost"
openssl req -x509 -nodes -newkey rsa:2048 -sha256 -keyout client.key -out client.crt -subj "/C=US/ST=California/L=San Jose/O=Example/OU=Developer/CN=client"

tunnel-server -address :10001 -proxy :10002 -cert server.crt -key server.key -trust client.crt
*/
package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/oatcode/portal"
)

func proxyListenAndServe(address string, cch chan<- net.Conn) {
	log.Printf("Proxy server...")
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("Listen: ", err)
	}
	defer l.Close()
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal("Accept: ", err)
		}
		log.Printf("Proxy server connected. conn=%s", connString(c))
		cch <- c
	}
}

func tunnelListenAndServe(address string, tlsConfig *tls.Config, cch <-chan net.Conn) {
	log.Printf("Tunnel server...")
	l, err := tls.Listen("tcp", address, tlsConfig)
	if err != nil {
		log.Fatal("Listen: ", err)
	}
	defer l.Close()
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal("Accept: ", err)
		}
		defer c.Close()
		log.Printf("Tunnel server connected. conn=%s", connString(c))
		// Serve one client connection at a time. This blocks until finish.
		portal.TunnelServe(c, cch)
	}
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
	clientCAs := x509.NewCertPool()
	clientCAs.AppendCertsFromPEM(pemCerts)
	return &tls.Config{
		Certificates: []tls.Certificate{cer},
		ClientCAs:    clientCAs,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
}

func connString(c net.Conn) string {
	return fmt.Sprintf("%v->%v", c.LocalAddr(), c.RemoteAddr())
}

func main() {
	var address string
	var proxy string
	var certFile string
	var keyFile string
	var trustFile string
	flag.StringVar(&address, "address", "", "Address [<server-ip>]:<server-port>")
	flag.StringVar(&proxy, "proxy", "", "Proxy [<ip>]:<port>")
	flag.StringVar(&certFile, "cert", "", "TLS certificate filename")
	flag.StringVar(&keyFile, "key", "", "TLS certificate key filename")
	flag.StringVar(&trustFile, "trust", "", "TLS trusted client certificates filename")
	flag.Parse()

	tlsConfig := loadCert(certFile, keyFile, trustFile)

	portal.SetPrintf(log.Printf)

	cch := make(chan net.Conn)
	go proxyListenAndServe(proxy, cch)
	tunnelListenAndServe(address, tlsConfig, cch)
}
