package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/oatcode/portal"
	"nhooyr.io/websocket"
)

var JwtCert = os.Getenv("JWT_CERT")
var SecretCert = os.Getenv("SECRET_CERT")
var SecretKey = os.Getenv("SECRET_KEY")
var ServerPort = os.Getenv("SERVER_PORT")
var ServerInternalPort = os.Getenv("SERVER_INTERNAL_PORT")
var RedisAddress = os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT")
var client bool
var address string
var jwtToken string
var trustFile string

func main() {
	flag.BoolVar(&client, "client", false, "Run client")
	flag.StringVar(&address, "address", "", "Address [<ip>]:<port>")
	flag.StringVar(&jwtToken, "jwt", "", "JWT token")
	flag.StringVar(&trustFile, "trust", "", "TLS trust certificate filename")
	flag.Parse()

	portal.Logf = log.Printf

	if client {
		tunnelClient()
	} else {
		// default is server
		// TODO separate client and server
		// TODO only thing common is WebsocketFramer
		// TODO use JWT instead of tunnelPassword
		tunnelServer()
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

func connString(c net.Conn) string {
	return fmt.Sprintf("%v->%v", c.LocalAddr(), c.RemoteAddr())
}
