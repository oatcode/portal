// Package portal provides the ability to build a 2-node HTTP tunnel
package portal

import (
	"context"
	fmt "fmt"
	"io"
	"math"
	"net"
	"net/http"
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

Generate protobuf file with:
  protoc --proto_path=pkg/message --go_out=. pkg/message/message.proto

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
*/

// Framer is for reading and writing messages with boundaries (i.e. frame)
type Framer interface {
	// Read reads a message from the connection
	// The returned byte array is of the exact length of the message
	Read() (b []byte, err error)

	// Write writes the entire byte array as a message to the connection
	Write(b []byte) error

	// Close closes the connection
	Close(err error) error
}

var (
	// Logf is for setting logging function
	Logf func(string, ...interface{})
)

const (
	bufferSize = 2048
)

func logf(fmt string, v ...interface{}) {
	if Logf != nil {
		Logf(fmt, v...)
	}
}

// Tunnel is for building a tunnel connection between two nodes
type Tunnel struct {
	// Create a connection on receiving proxy HTTP CONNECT from remote
	ProxyConnect func(context.Context, string) (net.Conn, error)
	// Feed new session to initiate new connection
	initiateSessionCh chan *session
}

type session struct {
	id            uint32
	conn          net.Conn
	isLocal       bool
	remoteAddress string // only used for proxy connect
	proxyWriterCh chan *message.Message
	tunnel        *Tunnel
}

// Hijack hijacks the proxied HTTP connection.
// The function name is borrowed from http.Hijacker.
// If this function returns no error, the caller should not use the http.ResponseWriter anymore.
// The only error returned is when hijacking is not supported.
func (tn *Tunnel) Hijack(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodConnect {
		return fmt.Errorf("method not supported")
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return fmt.Errorf("webserver doesn't support hijacking")
	}
	// HTTP CONNECT does not have a body. So there is no need to handle buf
	conn, _, err := hj.Hijack()
	if err != nil {
		return fmt.Errorf("failed to hijack connection: %v", err)
	}
	// Reset deadline to prevent timeout
	conn.SetDeadline(time.Time{})

	address := r.URL.Host
	s := &session{
		conn:          conn,
		isLocal:       true,
		tunnel:        tn,
		remoteAddress: address,
	}

	tn.initiateSessionCh <- s
	return nil
}

func (s *session) proxyWriter() {
	logf("proxyWriter starts. %s", s.String())
	defer func() {
		logf("proxyWriter ends. %s", s.String())
		s.conn.Close()
	}()
	for co := range s.proxyWriterCh {
		switch co.Type {
		case message.Message_PROXY_CONNECTED:
			s.conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
			logf("proxyWriter proxy connected. %s", s.String())
		case message.Message_SERVICE_UNAVAILABLE:
			s.conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\nConnection: close\r\n\r\n"))
			logf("proxyWriter service unavailable. %s", s.String())
		case message.Message_DISCONNECTED:
			logf("proxyWriter disconnected. %s", s.String())
			// channel will be closed by mapper
		case message.Message_DATA:
			s.conn.Write(co.Data)
		}
	}
}

func (s *session) String() string {
	return fmt.Sprintf("id=%d conn=%v->%v", s.id, s.conn.LocalAddr(), s.conn.RemoteAddr())
}

// proxyReader uses the origin to denote if it is handling a local initiated connection or a remote one
func (s *session) proxyReader(outputCh chan<- *message.Message) {
	logf("proxyReader starts. %s", s.String())
	defer logf("proxyReader ends. %s", s.String())
	var origin message.Message_Origin
	if s.isLocal {
		origin = message.Message_ORIGIN_LOCAL
	} else {
		origin = message.Message_ORIGIN_REMOTE
	}
	for {
		buf := make([]byte, bufferSize)
		len, err := s.conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				logf("proxyReader local disconnected. %s", s.String())
			} else if strings.Contains(err.Error(), "use of closed network connection") {
				logf("proxyReader remote disconnected. %s", s.String())
			} else {
				logf("proxyReader read error. %s err=%v", s.String(), err)
			}

			co := &message.Message{
				Type:   message.Message_DISCONNECTED,
				Origin: origin,
				Id:     s.id,
			}
			outputCh <- co
			return
		}

		co := &message.Message{
			Type:   message.Message_DATA,
			Origin: origin,
			Id:     s.id,
			Data:   buf[0:len],
		}
		outputCh <- co
	}
}

func (s *session) proxyConnector(ctx context.Context, outputCh chan<- *message.Message) {
	logf("proxyConnector connecting. id=%d sa=%s", s.id, s.remoteAddress)
	var conn net.Conn
	var err error
	if s.tunnel.ProxyConnect == nil {
		var d net.Dialer
		conn, err = d.DialContext(ctx, "tcp", s.remoteAddress)
	} else {
		conn, err = s.tunnel.ProxyConnect(ctx, s.remoteAddress)
	}
	if err != nil {
		outputCh <- &message.Message{Id: s.id, Type: message.Message_SERVICE_UNAVAILABLE}
		logf("proxyConnector ProxyConnect error. id=%d sa=%s err=%v", s.id, s.remoteAddress, err)
		return
	}
	s.conn = conn
	logf("proxyConnector connected. %s", s.String())

	go s.proxyWriter()
	go s.proxyReader(outputCh)
	outputCh <- &message.Message{Id: s.id, Type: message.Message_PROXY_CONNECTED}
}

// Requires 2 maps to differenciate local and remote originated connections
//
//	lm is local channel map
//	rm is remote channel map
//
// Connection map is only used until connection is connected
//
//	lcm is local connection map
func (tn *Tunnel) mapper(ctx context.Context, inputCh <-chan *message.Message, outputCh chan<- *message.Message) {
	logf("mapper starts")
	defer logf("mapper ends")

	var availableId uint32
	lconn := make(map[uint32]*session)
	rconn := make(map[uint32]*session)
	defer func() {
		// Tunnel closed. Close sessions
		for _, s := range lconn {
			close(s.proxyWriterCh)
			s.conn.Close()
		}
		for _, s := range rconn {
			close(s.proxyWriterCh)
			s.conn.Close()
		}
	}()

	for {
		select {
		case i, ok := <-inputCh:
			if !ok {
				return
			}
			// From remote
			switch i.Type {
			case message.Message_PROXY_CONNECT:
				// Remote initiated. Local conn not created yet
				s := &session{
					tunnel:        tn,
					remoteAddress: i.Address,
					isLocal:       false,
					id:            i.Id,
					proxyWriterCh: make(chan *message.Message),
				}
				rconn[i.Id] = s
				go s.proxyConnector(ctx, outputCh)
			case message.Message_PROXY_CONNECTED:
				// Local initiated
				s := lconn[i.Id]
				go s.proxyReader(outputCh)
				s.proxyWriterCh <- i
			case message.Message_SERVICE_UNAVAILABLE:
				// Local initiated
				pch := lconn[i.Id].proxyWriterCh
				delete(lconn, i.Id)
				pch <- i
				close(pch)
			default:
				var m map[uint32]*session
				if i.Origin == message.Message_ORIGIN_LOCAL {
					// Received from other side with local origin. Use remote map
					m = rconn
				} else {
					m = lconn
				}
				pwch := m[i.Id].proxyWriterCh
				pwch <- i
				if i.Type == message.Message_DISCONNECTED {
					delete(m, i.Id)
					close(pwch)
				}
			}
		case s := <-tn.initiateSessionCh:
			// Find next available id.
			used := true
			for range int32(math.MaxInt32) {
				availableId++
				if _, used = lconn[availableId]; !used {
					break
				}
			}
			if used {
				s.conn.Write([]byte("HTTP/1.1 429 Too Many Requests\r\n\r\n"))
				s.conn.Close()
				logf("Too many connections")
				continue
			}
			// Setup session
			s.id = availableId
			s.proxyWriterCh = make(chan *message.Message)
			lconn[s.id] = s
			go s.proxyWriter()

			outputCh <- &message.Message{Id: s.id, Type: message.Message_PROXY_CONNECT, Address: s.remoteAddress}
		}
	}
}

// Send data to the other side of the tunnel
func tunnelWriter(ctx context.Context, c Framer, och <-chan *message.Message) {
	logf("tunnelWriter starts")
	defer logf("tunnelWriter ends")
	for {
		select {
		case co, ok := <-och:
			if !ok {
				logf("tunnelWriter channel closed")
				c.Close(nil)
				return
			}
			data, err := proto.Marshal(co)
			if err != nil {
				logf("tunnelWriter marshal error: %v", err)
				c.Close(err)
				return
			}
			if err = c.Write(data); err != nil {
				logf("tunnelWriter write error: %v", err)
				c.Close(err)
				return
			}
		case <-ctx.Done():
			c.Close(ctx.Err())
			return
		}
	}
}

// Read commands comming from the other side of the tunnel
func tunnelReader(c Framer, inputCh chan<- *message.Message) {
	logf("tunnelReader starts")
	defer logf("tunnelReader ends")
	var err error
	var buf []byte
	for {
		buf, err = c.Read()
		if err != nil {
			break
		}
		co := &message.Message{}
		if err = proto.Unmarshal(buf, co); err != nil {
			break
		}
		inputCh <- co
	}
	if err == io.EOF {
		logf("tunnelReader disconnected remotely")
		c.Close(err)
	} else if strings.Contains(err.Error(), "use of closed network connection") {
		// Connection closed locally. No need to close it again
		logf("tunnelReader disconnected locally")
	} else {
		logf("tunnelReader error: %v", err)
		c.Close(err)
	}
}

// Serve starts the tunnel communication with the remote side. This blocks until connection is closed or context is cancelled.
func (tn *Tunnel) Serve(ctx context.Context, c Framer) {
	logf("TunnelServe starts")
	defer logf("TunnelServe ends")

	tn.initiateSessionCh = make(chan *session)
	inputCh := make(chan *message.Message)
	outputCh := make(chan *message.Message)

	go tn.mapper(ctx, inputCh, outputCh)
	go tunnelWriter(ctx, c, outputCh)
	// This blocks until connection closed
	tunnelReader(c, inputCh)

	close(inputCh)
	// Don't close outputCh, as mapper may still use it. Let GC takes care of it.
	// Don't close initiateSessionCh, as proxyConnect may still use it. Let GC takes care of it.
}
