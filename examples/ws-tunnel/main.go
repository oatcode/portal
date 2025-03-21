package main

import (
	"context"
	"flag"
	"log"

	"github.com/coder/websocket"
	"github.com/oatcode/portal"
)

var client bool
var server bool
var address string
var proxyBasicAuth string
var proxyBearerAuth string
var tunnelBasicAuth string
var tunnelBearerAuth string
var certFile string
var keyFile string
var trustFile string

func main() {
	flag.BoolVar(&client, "client", false, "Run client")
	flag.BoolVar(&server, "server", false, "Run server")
	flag.StringVar(&address, "address", "", "Address [<hostname>]:<port>")
	flag.StringVar(&proxyBasicAuth, "proxyBasicAuth", "", "Proxy basic auth <username>:<password>")
	flag.StringVar(&proxyBearerAuth, "proxyBearerAuth", "", "Proxy bearer token")
	flag.StringVar(&tunnelBasicAuth, "tunnelBasicAuth", "", "Tunnel basic auth <username>:<password>")
	flag.StringVar(&tunnelBearerAuth, "tunnelBearerAuth", "", "Tunnel bearer auth token")
	flag.StringVar(&certFile, "cert", "", "TLS certificate filename")
	flag.StringVar(&keyFile, "key", "", "TLS certificate key filename")
	flag.StringVar(&trustFile, "trust", "", "TLS client certificate filename to trust")
	flag.Parse()

	portal.Logf = log.Printf

	if server {
		tunnelServer()
	} else if client {
		tunnelClient()
	}
}

type WebsocketFramer struct {
	conn *websocket.Conn
}

func NewWebsocketFramer(conn *websocket.Conn, connString string) *WebsocketFramer {
	return &WebsocketFramer{conn: conn}
}
func (c *WebsocketFramer) Read() (b []byte, err error) {
	_, b, err = c.conn.Read(context.Background())
	return b, err
}

func (c *WebsocketFramer) Write(b []byte) error {
	return c.conn.Write(context.Background(), websocket.MessageBinary, b)
}

func (c *WebsocketFramer) Close(err error) error {
	if err == nil {
		return c.conn.Close(websocket.StatusNormalClosure, "")
	} else {
		return c.conn.Close(websocket.StatusInternalError, err.Error())
	}
}
