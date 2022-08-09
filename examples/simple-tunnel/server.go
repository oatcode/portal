package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/oatcode/portal"
)

var coch = make(chan portal.ConnectOperation)

type proxyConnectHandler struct{}

func (h proxyConnectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Need to clean deadlines in case it was set
	conn.SetDeadline(time.Time{})
	coch <- portal.ConnectOperation{Conn: conn, Address: r.URL.Host}

	log.Printf("Proxy connect: %s", connString(conn))
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
		go portal.TunnelServe(context.Background(), NewNetConnFramer(c), coch)
	}
}

func tunnelServer() {
	log.Printf("Tunnel server...")
	go http.ListenAndServe(proxyAddress, proxyConnectHandler{})
	tunnelListenAndServe()
}
