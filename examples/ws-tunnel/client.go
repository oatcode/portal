package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/coder/websocket"
	"github.com/oatcode/portal"
)

func dialAndServe(tlsConfig *tls.Config) {
	u := url.URL{
		Scheme: "https",
		Host:   address,
		Path:   "tunnel",
	}

	h := http.Header{}
	if tunnelBasicAuth != "" {
		h.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(tunnelBasicAuth)))
	}
	if tunnelBearerAuth != "" {
		h.Add("Authorization", "Bearer "+tunnelBearerAuth)
	}
	options := &websocket.DialOptions{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
				Proxy:           http.ProxyFromEnvironment,
			},
		},
		HTTPHeader: h,
	}
	c, _, err := websocket.Dial(context.Background(), u.String(), options)
	if err != nil {
		log.Fatal("Dial: ", err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")
	log.Print("Tunnel client connected")

	tn := portal.Tunnel{
		Client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					log.Printf("ProxyConnect: %s", addr)
					return net.Dial("tcp", directAddress)
				},
				DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					log.Printf("ProxyConnectTLS: %s", addr)
					return tls.Dial("tcp", directAddress, &tls.Config{
						InsecureSkipVerify: true,
					})
				},
			},
		},
	}
	tn.Serve(context.Background(), NewWebsocketFramer(c, address))
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
