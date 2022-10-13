package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/oatcode/portal"
	"nhooyr.io/websocket"
)

var coch = make(chan portal.ConnectOperation)

type proxyConnectHandler struct {
	other *http.ServeMux
}

func (h proxyConnectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		if !proxyAuth(r) {
			http.Error(w, "proxy authentication failed", http.StatusUnauthorized)
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
		log.Printf("Proxy connect: %s", connString(conn))
		coch <- portal.ConnectOperation{Conn: conn, Address: r.URL.Host}
	} else {
		h.other.ServeHTTP(w, r)
	}
}

func tunnelHandler(w http.ResponseWriter, r *http.Request) {
	if !tunnelAuth(r) {
		http.Error(w, "tunnel authentication failed", http.StatusUnauthorized)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		panic(err)
	}
	go portal.TunnelServe(context.Background(), NewWebsocketFramer(conn, r.RemoteAddr), coch)
}

// Copied from golang's http lib
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

func verifyBasic(auth, userpw string) bool {
	const prefix = "Basic "
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return false
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return false
	}
	return string(c) == userpw
}

func verifyBearer(auth, token string) bool {
	const prefix = "Bearer "
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return false
	}
	return auth[len(prefix):] == token
}

func proxyAuth(r *http.Request) bool {
	auth := r.Header.Get("Proxy-Authorization")
	if proxyBasicAuth != "" {
		return verifyBasic(auth, proxyBasicAuth)
	}
	if proxyBearerAuth != "" {
		return verifyBearer(auth, proxyBearerAuth)
	}
	return true
}

func tunnelAuth(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if tunnelBasicAuth != "" {
		return verifyBasic(auth, tunnelBasicAuth)
	}
	if tunnelBearerAuth != "" {
		return verifyBearer(auth, tunnelBearerAuth)
	}
	return true
}

func createServerTlsConfig(certFile string, keyFile string) *tls.Config {
	cer, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cer},
	}
}

func connString(c net.Conn) string {
	return fmt.Sprintf("%v->%v", c.LocalAddr(), c.RemoteAddr())
}

func tunnelServer() {
	log.Printf("Tunnel server...")

	otherHandler := http.NewServeMux()
	otherHandler.HandleFunc("/tunnel", tunnelHandler)

	listener, err := tls.Listen("tcp", address, createServerTlsConfig(certFile, keyFile))
	if err != nil {
		log.Fatal(err)
	}
	http.Serve(listener, proxyConnectHandler{
		other: otherHandler,
	})
}
