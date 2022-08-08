package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/oatcode/portal"
)

func connString(c net.Conn) string {
	return fmt.Sprintf("%v->%v", c.LocalAddr(), c.RemoteAddr())
}

type NetConnFramer struct {
	conn net.Conn
}

func NewNetConnFramer(conn net.Conn) *NetConnFramer {
	return &NetConnFramer{conn: conn}
}

func (c *NetConnFramer) Read() (b []byte, err error) {
	// Read len first then content
	var dl int32
	if err = binary.Read(c.conn, binary.LittleEndian, &dl); err != nil {
		return nil, err
	}
	buf := make([]byte, dl)
	if _, err = io.ReadFull(c.conn, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (c *NetConnFramer) Write(b []byte) error {
	// Write len first then content
	dl := int32(len(b))
	if err := binary.Write(c.conn, binary.LittleEndian, dl); err != nil {
		return err
	}
	_, err := c.conn.Write(b)
	return err
}

func (c *NetConnFramer) Close(err error) error {
	return c.conn.Close()
}

var client bool
var server bool
var tunnelAddress string
var proxyAddress string

func main() {
	flag.BoolVar(&client, "client", false, "Run client")
	flag.BoolVar(&server, "server", false, "Run server")
	flag.StringVar(&tunnelAddress, "tunnelAddress", "", "Tunnel address [<ip>]:<port>")
	flag.StringVar(&proxyAddress, "proxyAddress", "", "Proxy [<ip>]:<port>")
	flag.Parse()

	portal.Logf = log.Printf

	if server {
		tunnelServer()
	} else if client {
		tunnelClient()
	}
}
