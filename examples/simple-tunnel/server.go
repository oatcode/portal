package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/oatcode/portal"
)

var tunnel *portal.Tunnel

type proxyConnectHandler struct{}

func (h proxyConnectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if tunnel != nil {
		tunnel.Hijack(w, r)
	} else {
		http.Error(w, "tunnel not available", http.StatusServiceUnavailable)
		return
	}
}

func tunnelListenAndServe() {
	l, err := net.Listen("tcp", tunnelAddress)
	if err != nil {
		log.Fatal(err)
	}
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Tunnel server connected: %s", connString(c))
		tunnel = &portal.Tunnel{}
		tunnel.Serve(context.Background(), NewNetConnFramer(c))
	}
}

func tunnelServer() {
	log.Printf("Tunnel server...")
	go http.ListenAndServe(proxyAddress, proxyConnectHandler{})
	tunnelListenAndServe()
}
