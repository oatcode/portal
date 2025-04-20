package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"

	"github.com/oatcode/portal"
)

func tunnelClient() {
	log.Printf("Tunnel client...")
	c, err := net.Dial("tcp", tunnelAddress)
	if err != nil {
		log.Fatalf("Tunnel client dial error: %v", err)
	}
	defer c.Close()
	log.Print("Tunnel client connected")

	tunnel := &portal.Tunnel{
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
	tunnel.Serve(context.Background(), NewNetConnFramer(c))
}
