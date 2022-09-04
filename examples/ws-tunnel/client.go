package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/oatcode/portal"
	"nhooyr.io/websocket"
)

func dialAndServe(tlsConfig *tls.Config) {
	u := url.URL{
		Scheme: "https",
		Host:   address,
		Path:   "tunnel",
	}
	options := &websocket.DialOptions{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
		HTTPHeader: http.Header{
			"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte(tunnelUsername+":"+tunnelPassword))},
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

func createClientTlsConfig(trustFile string) *tls.Config {
	pemCerts, err := ioutil.ReadFile(trustFile)
	if err != nil {
		log.Fatal(err)
	}
	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(pemCerts)
	return &tls.Config{
		RootCAs: rootCAs,
	}
}

func tunnelClient() {
	log.Printf("Tunnel client...")
	dialAndServe(createClientTlsConfig(trustFile))
}
