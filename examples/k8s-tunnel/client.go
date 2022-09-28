package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/oatcode/portal"
	"nhooyr.io/websocket"
)

func loadTrust(trustFile string) *tls.Config {
	trust, err := ioutil.ReadFile(trustFile)
	if err != nil {
		log.Fatal(err)
	}
	rootCAs := x509.NewCertPool()
	if !rootCAs.AppendCertsFromPEM(trust) {
		log.Fatalf("failed to append pem: %v", trust)
	}
	return &tls.Config{
		RootCAs: rootCAs,
	}
}

func tunnelClient() {
	log.Printf("Tunnel client...")

	u := url.URL{
		Scheme: "https",
		Host:   address,
		Path:   "tunnel",
	}

	options := &websocket.DialOptions{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: loadTrust(trustFile),
			},
		},
		HTTPHeader: http.Header{
			"Authorization": {"Bearer " + jwtToken},
		},
	}
	c, _, err := websocket.Dial(context.Background(), u.String(), options)
	if err != nil {
		log.Fatal("Dial: ", err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")
	log.Print("Tunnel client connected")

	portal.TunnelServe(context.Background(), NewWebsocketFramer(c, address), nil)
}
