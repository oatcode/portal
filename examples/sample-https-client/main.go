package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type stringFlags []string

func (s *stringFlags) String() string {
	return fmt.Sprint(*s)
}

func (s *stringFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func loadCert(trustFiles []string) *tls.Config {
	rootCAs := x509.NewCertPool()
	for _, f := range trustFiles {
		trust, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		rootCAs.AppendCertsFromPEM(trust)
	}
	return &tls.Config{
		RootCAs: rootCAs,
	}
}

func main() {
	var address string
	var proxy string
	var trustFiles stringFlags
	flag.StringVar(&address, "url", "", "HTTP GET URL")
	flag.StringVar(&proxy, "proxy", "", "Proxy URL")
	flag.Var(&trustFiles, "trust", "TLS trust certificate filename")
	flag.Parse()

	tlsConfig := loadCert(trustFiles)

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
