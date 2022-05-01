// Package portal provides ability to build a 2-node HTTP tunnel
package portal

import (
	"bufio"
	"encoding/binary"
	fmt "fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/oatcode/portal/pkg/message"
	"google.golang.org/protobuf/proto"
)

/*
This is a 2-node HTTP tunnel proxy
The main goal is to provide access from cloud to on-prem without opening ports on-prem.
This proxy has two sides: tunnel client and tunnel server
On-prem running tunnel client will connect to tunnel server running in cloud.
Proxy port can be opened on cloud side to allow access to on-prem via HTTP tunnelling:
  https://en.wikipedia.org/wiki/HTTP_tunnel
Proxy listens to incoming tunneling request and interact with the remote side
A mapper keeps keep track of remote/local initiated connections separately to prevent ID conflicts

Command line:
  proxy -server [<server-ip>]:<server-port> [-proxy [<proxy-ip>]:<proxy-port>]
  proxy -client <server-ip>:<server-port> [-proxy [<proxy-ip>]:<proxy-port>]

Build with:
  protoc -I . message.proto --go_out=plugins=grpc:.
   install ./...

Appreviations used in code:
ich = tunnel input channel
och = tunnel output channel
cch = connect channel for passing new proxy connection
pch = proxy writer channel
co = command

The close sequence for sides s1 and s2
s1 proxy-reader: read error. send disconnect to tunnel
s2 mapper: recv disconnect. remove mapping. send to proxy-writer
s2 proxy-writer: recv disconnect. close socket.
s2 proxy-reader: read error (as writer closed it). send disconnect to tunnel
s1 mapper: recv disconnect. remove mapping. send to proxy-writer
s1 proxy-writer: recv disconnect. close socket

Flow
C  = Client
PL = Proxy Listener
TS = Tunnel Server
TC = Tunnel Client
PC = Proxy Connector
S  = Server
PR = Proxy Reader
PW = Proxy Writer

+------+          +------+          +------+            +------+          +------+          +------+
|      |          |      |          |      |            |      |          |      |          |      |
|  C   |----------|  PL  |----------|  TS  |------------|  TC  |----------|  PC  |----------|  S   |
|      |          |      |          |      |            |      |          |      |          |      |
+------+          +------+          +------+            +------+          +------+          +------+


+------+          +------+          +------+            +------+          +------+          +------+
|      |----------|  PR  |----------|      |            |      |----------|  PR  |----------|      |
|  C   |          +------+          |  TS  |------------|  TC  |          +------+          |  S   |
|      |----------|  PW  |----------|      |            |      |----------|  PW  |----------|      |
+------+          +------+          +------+            +------+          +------+          +------+

Note
- Proxy can also run on tunnel client side or both
- HTTP Connector on remote side will return 503 for any connection error
- TODO FIX: id is unique for 2 billion, but still finite
- TODO TRY: without origin, but data_origin/data_remote?
*/

type connectOperation struct {
	c       net.Conn
	address string
}

var (
	// Logf is for setting logging function
	Logf func(string, ...interface{})

	// Filter is for filtering HTTP CONNECT requests
	Filter func(http.Request) bool

	// ConnectTimeout is for timing out HTTP CONNECT
	ConnectTimeout = 5 * time.Second
)

func proxyWriter(c net.Conn, pch <-chan *message.Message) {
	logf("proxyWriter starts. conn=%s", connString(c))
	defer c.Close()
	for co := range pch {
		if co.Type == message.Message_HTTP_CONNECT_OK {
			c.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
			logf("proxyWriter connected. conn=%s", connString(c))
		} else if co.Type == message.Message_HTTP_SERVICE_UNAVAILABLE {
			c.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n"))
			logf("proxyWriter service unavailable. conn=%s", connString(c))
			return
		} else if co.Type == message.Message_DISCONNECTED {
			logf("proxyWriter disconnected. conn=%s", connString(c))
			return
		} else if co.Type == message.Message_DATA {
			c.Write(co.Buf)
		}
	}
	logf("proxyWriter channel closed. conn=%s", connString(c))
}

// proxyReader uses the origin to denote if it is handling a local initiated connection or a remote one
func proxyReader(c net.Conn, och chan<- *message.Message, id int32, origin message.Message_Origin) {
	logf("proxyReader starts. conn=%s", connString(c))
	for {
		buf := make([]byte, 2048)
		len, err := c.Read(buf)
		if err != nil {
			if err == io.EOF {
				logf("proxyReader local disconnected. conn=%s", connString(c))
			} else if strings.Contains(err.Error(), "use of closed network connection") {
				logf("proxyReader remote disconnected. conn=%s", connString(c))
			} else {
				logf("proxyReader read error. conn=%s err=%v", connString(c), err)
			}

			co := &message.Message{
				Type:   message.Message_DISCONNECTED,
				Origin: origin,
				Id:     id,
			}
			och <- co
			return
		}

		co := &message.Message{
			Type:   message.Message_DATA,
			Origin: origin,
			Id:     id,
			Buf:    buf[0:len],
		}
		och <- co
	}
}

// Process HTTP CONNECT
func proxyConnect(cch <-chan net.Conn, coch chan<- connectOperation) {
	for c := range cch {
		// Set timeout to read the HTTP CONNECT message
		c.SetReadDeadline(time.Now().Add(ConnectTimeout))

		// Read request line
		bufReader := bufio.NewReader(c)
		first, err := bufReader.ReadString('\n')
		if err != nil {
			logf("Proxy request read error: %v", err)
			c.Close()
			continue
		}

		// Process request line
		tokens := strings.Split(first, " ")
		if len(tokens) != 3 {
			c.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
			c.Close()
			continue
		}
		method, address := tokens[0], tokens[1]

		if method != http.MethodConnect {
			c.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\n\r\n"))
			c.Close()
			continue
		}

		req := http.Request{
			Method: http.MethodConnect,
			URL:    &url.URL{Host: address},
			Header: make(http.Header),
		}
		// Read headers
		for {
			line, err := bufReader.ReadString('\n')
			if err != nil {
				logf("Proxy headers error: %v", err)
				c.Close()
				continue
			}
			if len(line) == 2 {
				// Only \r\n
				break
			}
			tokens := strings.SplitN(line, ":", 2)
			if len(tokens) == 2 {
				name := tokens[0]
				value := strings.Trim(tokens[1], " \r\n")
				req.Header.Add(name, value)
			}
		}

		// Authorize
		if Filter != nil {
			if !Filter(req) {
				c.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\n\r\n"))
				c.Close()
				continue
			}
		}

		// Reset timeout
		c.SetReadDeadline(time.Time{})

		coch <- connectOperation{c: c, address: address}
	}
}

func proxyConnector(sa string, och chan<- *message.Message, pch <-chan *message.Message, id int32) {
	logf("proxyConnector connecting. sa=%s", sa)
	c, err := net.Dial("tcp", sa)
	if err != nil {
		co := &message.Message{
			Type: message.Message_HTTP_SERVICE_UNAVAILABLE,
			Id:   id,
		}
		och <- co
		logf("proxyConnector connect error. sa=%s err=%v", sa, err)
		return
	}
	logf("proxyConnector connected. conn=%s", connString(c))

	go proxyWriter(c, pch)
	go proxyReader(c, och, id, message.Message_ORIGIN_REMOTE)

	co := &message.Message{
		Type: message.Message_HTTP_CONNECT_OK,
		Id:   id,
	}
	och <- co
}

// Requires 2 maps to differenciate local and remote originated connections
//   lm is local channel map
//   rm is remote channel map
// Connection map is only used until connection is connected
//   lcm is local connection map
func mapper(ich <-chan *message.Message, coch <-chan connectOperation, och chan<- *message.Message) {
	var id int32
	lm := make(map[int32]chan<- *message.Message)
	rm := make(map[int32]chan<- *message.Message)
	lcm := make(map[int32]net.Conn)

	for {
		select {
		case i, ok := <-ich:
			if !ok {
				// Channel closed. Clear connections
				for _, ch := range lm {
					close(ch)
				}
				for _, ch := range rm {
					close(ch)
				}
				return
			}
			// From remote
			if i.Type == message.Message_HTTP_CONNECT {
				// Remote initiated
				pch := make(chan *message.Message)
				rm[i.Id] = pch
				go proxyConnector(i.SocketAddress, och, pch, i.Id)
			} else if i.Type == message.Message_HTTP_CONNECT_OK {
				// Local initiated
				c := lcm[i.Id]
				delete(lcm, i.Id)
				go proxyReader(c, och, i.Id, message.Message_ORIGIN_LOCAL)
				pch := lm[i.Id]
				pch <- i
			} else if i.Type == message.Message_HTTP_SERVICE_UNAVAILABLE {
				// Local initiated
				delete(lcm, i.Id)
				pch := lm[i.Id]
				delete(lm, i.Id)
				pch <- i
			} else {
				var m map[int32]chan<- *message.Message
				if i.Origin == message.Message_ORIGIN_LOCAL {
					// Received from other side with local origin. Use remote map
					m = rm
				} else {
					m = lm
				}
				pch := m[i.Id]
				if i.Type == message.Message_DISCONNECTED {
					delete(m, i.Id)
				}
				pch <- i
			}
		case co := <-coch:
			// New connection from local
			lcm[id] = co.c
			pch := make(chan *message.Message)
			lm[id] = pch
			go proxyWriter(co.c, pch)

			och <- &message.Message{
				Type:          message.Message_HTTP_CONNECT,
				Id:            id,
				SocketAddress: co.address,
			}
			id++
		}
	}
}

// Send data to the other side of the tunnel
func tunnelWriter(c net.Conn, och <-chan *message.Message, stop <-chan struct{}) {
	logf("tunnelWriter starts. conn=%s", connString(c))
	var err error
OutterLoop:
	for {
		select {
		case co := <-och:
			var data []byte
			data, err = proto.Marshal(co)
			if err != nil {
				break OutterLoop
			}
			dl := int32(len(data))
			if err = binary.Write(c, binary.LittleEndian, dl); err != nil {
				break OutterLoop
			}
			c.Write(data)
		case <-stop:
			break OutterLoop
		}
	}
	if err != nil {
		logf("tunnelWriter error. conn=%s err=%v", connString(c), err)
	}
	logf("tunnelWriter ends. conn=%s", connString(c))
}

// Read commands comming from the other side of the tunnel
func tunnelReader(c net.Conn, ich chan<- *message.Message) {
	logf("tunnelReader starts. conn=%s", connString(c))
	defer c.Close()
	var err error
	for {
		// Read len first
		var dl int32
		if err = binary.Read(c, binary.LittleEndian, &dl); err != nil {
			break
		}
		// Then read content
		buf := make([]byte, dl)
		if _, err = io.ReadFull(c, buf); err != nil {
			break
		}

		co := &message.Message{}
		if err = proto.Unmarshal(buf, co); err != nil {
			break
		}
		ich <- co
	}
	if err == io.EOF {
		logf("tunnelReader disconnected. conn=%s", connString(c))
	} else {
		logf("tunnelReader error. conn=%s err=%v", connString(c), err)
	}
}

// TunnelServe starts the communication with the remote side with tunnel messages connection c.
// It handles new proxy connections coming into connection channel cch.
func TunnelServe(c net.Conn, cch <-chan net.Conn) {
	ich := make(chan *message.Message)
	och := make(chan *message.Message)
	stop := make(chan struct{})
	coch := make(chan connectOperation)

	go proxyConnect(cch, coch)
	go mapper(ich, coch, och)
	go tunnelWriter(c, och, stop)
	// This blocks until connection closed
	tunnelReader(c, ich)

	// Reader closed means socket closed.
	// Stop writer so it won't write to closed socket anymore
	close(stop)

	// Close ich to trigger mapper to close all proxyWriters,
	// while proxyWriters close socket will stop proxyReaders
	close(ich)

	// Don't close och, as proxyReaders may still use it. Let GC takes care of it.
}

func connString(c net.Conn) string {
	return fmt.Sprintf("%v->%v", c.LocalAddr(), c.RemoteAddr())
}

func logf(fmt string, v ...interface{}) {
	if Logf != nil {
		Logf(fmt, v)
	}
}
