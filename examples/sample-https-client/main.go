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

func loadTrust(trustFiles []string) *tls.Config {
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
	var proxyBearer string
	var trustFiles stringFlags
	flag.StringVar(&address, "url", "", "HTTP GET URL")
	flag.StringVar(&proxy, "proxy", "", "Proxy URL")
	flag.StringVar(&proxyBearer, "proxy-bearer", "", "Proxy bearer token")
	flag.Var(&trustFiles, "trust", "TLS trust certificate filename")
	flag.Parse()

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		log.Fatal(err)
	}

	transport := &http.Transport{
		TLSClientConfig: loadTrust(trustFiles),
		Proxy:           http.ProxyURL(proxyURL),
	}
	if proxyBearer != "" {
		transport.ProxyConnectHeader = http.Header{"Proxy-Authorization": []string{"Bearer " + proxyBearer}}
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
