/*
sample-http-client -url https://localhost:10003/tt/ -trust server.crt -proxy http://localhost:10002
*/

package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

func loadCert(trustFile string) *tls.Config {
	// Load trust cert
	trust, err := ioutil.ReadFile(trustFile)
	if err != nil {
		log.Fatal(err)
	}
	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(trust)
	return &tls.Config{
		RootCAs: rootCAs,
	}
}

func main() {
	var address string
	var proxy string
	var trustFile string
	flag.StringVar(&address, "url", "", "HTTP GET URL")
	flag.StringVar(&proxy, "proxy", "", "Proxy URL")
	flag.StringVar(&trustFile, "trust", "", "TLS trust certificate filename")
	flag.Parse()

	tlsConfig := loadCert(trustFile)

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		log.Fatal(err)
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyURL(proxyURL),
	}
	c := &http.Client{
		Transport: transport,
	}

	resp, err := c.Get(address)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("status=%d\n", resp.StatusCode)
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(data))
}
