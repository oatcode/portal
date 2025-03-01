package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"log"
	"net/http"
	"strings"

	"github.com/coder/websocket"
	"github.com/oatcode/portal"
)

var tunnel *portal.Tunnel

type proxyConnectHandler struct {
	other *http.ServeMux
}

func (h *proxyConnectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		if !proxyAuth(r) {
			http.Error(w, "proxy authentication failed", http.StatusUnauthorized)
			return
		}
		if tunnel != nil {
			if err := tunnel.Hijack(w, r); err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
			}
		} else {
			http.Error(w, "tunnel not available", http.StatusServiceUnavailable)
			return
		}
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

	// TODO Only one tunnel now!!!
	// TODO Need to load balance

	tunnel = &portal.Tunnel{}
	tunnel.Serve(context.Background(), NewWebsocketFramer(conn, r.RemoteAddr))
}

// directHandler for query that target directly to the tunnel server
// We will send it down and let the tunnel client handle where to connect
// We need to re-send the request. The net.Conn to pass in has to be a special one.
// It will replay the request method and header first then go back to the hijacked connection
func directHandler(w http.ResponseWriter, r *http.Request) {
	if !proxyAuth(r) {
		http.Error(w, "proxy authentication failed", http.StatusUnauthorized)
		return
	}

	if tunnel != nil {
		tunnel.Hijack(w, r)
	} else {
		http.Error(w, "tunnel not available", http.StatusServiceUnavailable)
		return
	}
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

func tunnelServer() {
	log.Printf("Tunnel server...")

	otherHandler := http.NewServeMux()
	otherHandler.HandleFunc("/tunnel", tunnelHandler)
	otherHandler.HandleFunc("/", directHandler)

	listener, err := tls.Listen("tcp", address, createServerTlsConfig(certFile, keyFile))
	if err != nil {
		log.Fatal(err)
	}
	http.Serve(listener, &proxyConnectHandler{
		other: otherHandler,
	})
}
