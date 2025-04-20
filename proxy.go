// Package portal provides the ability to build a 2-node HTTP tunnel
package portal

import (
	"context"
	fmt "fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/coder/websocket"
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
	Client            *http.Client
	initiateRequestCh chan *session
	initiateWsCh      chan *session
}

type session struct {
	id         uint32
	origin     message.Message_Origin // To determine if local or remote map
	request    *http.Request
	responseCh chan *http.Response
	bodyCh     chan *message.Message // To receive body stream from remote
	wsInputCh  chan *message.Message
	wsConn     *websocket.Conn
	tunnel     *Tunnel
}

func (tn *Tunnel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check if websocket
	if r.Method == http.MethodGet && strings.Contains(r.Header.Get("Upgrade"), "websocket") {
		tn.WsHandler(w, r)
		return
	}
	resp, err := tn.tunnelClientDo(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	for k, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func createRequestHeader(id uint32, r *http.Request) *message.Message {
	clString := r.Header.Get("Content-Length")
	var cl int64
	if clString != "" {
		var err error
		cl, err = strconv.ParseInt(clString, 10, 64)
		if err != nil {
			logf("error parsing Content-Length: %v", err)
			return nil
		}
	} else {
		cl = -1
	}
	return &message.Message{
		Id:            id,
		Type:          message.Message_REQUEST_HEADER,
		Method:        r.Method,
		Url:           r.URL.String(),
		Proto:         r.Proto,
		ContentLength: int64(cl),
		Headers:       extractHeaders(r.Header),
	}
}

// tunnelClientDo emulates http.Client.Do
func (tn *Tunnel) tunnelClientDo(r *http.Request) (*http.Response, error) {
	// Create a new request. Id will be set by the mapper
	s := &session{
		request:    r,
		responseCh: make(chan *http.Response),
		bodyCh:     make(chan *message.Message),
	}
	// Send request to mapper
	tn.initiateRequestCh <- s
	// Wait for response
	resp := <-s.responseCh
	return resp, nil
}

func (tn *Tunnel) WsHandler(w http.ResponseWriter, r *http.Request) {
	logf("*** WsHandler. %s", r.URL.String())
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		logf("ws accept error: %v", err)
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "")
	s := &session{
		request:   r,
		wsConn:    c,
		wsInputCh: make(chan *message.Message),
		tunnel:    tn,
		origin:    message.Message_ORIGIN_LOCAL,
	}
	s.tunnel.initiateWsCh <- s

	logf("*** WsHandler initiated. %s", s)

	s.wsWriter(c, s.wsInputCh)
}

func (s *session) wsWriter(c *websocket.Conn, inputCh <-chan *message.Message) {
	var wsType websocket.MessageType
	var writer io.WriteCloser
	var err error
	// TODO ctx??
	for i := range inputCh {
		logf("*** wsWriter receive. %s i=%v", s, i)
		if i.Type == message.Message_WS_MSG_HEADER {
			if i.WsMsgType == message.Message_TEXT {
				wsType = websocket.MessageText
			} else if i.WsMsgType == message.Message_BINARY {
				wsType = websocket.MessageBinary
			} else {
				logf("ws unknown message type: %v", i.WsMsgType)
				break
			}
			writer, err = c.Writer(context.Background(), wsType)
			if err != nil {
				logf("ws writer error: %v", err)
				break
			}
		} else if i.Type == message.Message_WS_MSG_DATA {
			if writer == nil {
				logf("ws writer not initialized")
				break
			}
			_, err := writer.Write(i.Data)
			if err != nil {
				logf("ws writer error: %v", err)
				break
			}
			if i.Eof {
				logf("*** wsWriter EOF. %s", s)
				if err := writer.Close(); err != nil {
					logf("ws writer close error: %v", err)
					break
				}
				writer = nil
			}
		} else if i.Type == message.Message_WS_CLOSE {
			logf("*** wsWriter CLOSE. %s", s)
			if writer != nil {
				if err := writer.Close(); err != nil {
					logf("ws writer close error: %v", err)
					break
				}
				writer = nil
			}
			c.Close(websocket.StatusCode(i.Code), i.Message)
			break
		}
	}
}

func (s *session) wsConnector(m *message.Message, outputCh chan<- *message.Message) {
	headers := make(http.Header)
	for _, h := range m.Headers {
		headers[h.Name] = h.Values
	}

	logf("wsConnector m=%v", m)

	opt := &websocket.DialOptions{
		HTTPClient: s.tunnel.Client,
		HTTPHeader: headers,
	}
	u := "wss://localhost" + m.Url
	// TODO ctx!!
	c, _, err := websocket.Dial(context.Background(), u, opt)
	if err != nil {
		logf("wsDial error: %v", err)
		outputCh <- &message.Message{
			Type:   message.Message_ERROR,
			Id:     s.id,
			Origin: s.origin,
		}
		return
	}
	outputCh <- &message.Message{
		Type:   message.Message_WS_CONNECTED,
		Id:     s.id,
		Origin: s.origin,
	}
	logf("wsConnect connected. %s", s.String())
	go s.wsWriter(c, s.wsInputCh)
	s.wsReader(c, outputCh)
}

// createResponse creates a response for the originating side
func (s *session) createResponse(m *message.Message) *http.Response {
	resp := &http.Response{
		StatusCode: int(m.Code),
		Proto:      m.Proto,
		Header:     make(http.Header),
	}
	for _, h := range m.Headers {
		resp.Header[h.Name] = h.Values
	}
	resp.Body = newBodyReader(s.request.Context(), s.bodyCh)
	return resp
}

func createErrorResponse(code int, err error) *http.Response {
	resp := &http.Response{
		StatusCode: code,
		Proto:      "HTTP/1.1",
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "text/plain")
	resp.Body = io.NopCloser(strings.NewReader(err.Error()))
	return resp
}

func (s *session) String() string {
	return fmt.Sprintf("id=%d", s.id)
}

func (s *session) Close() error {
	if s.bodyCh != nil {
		close(s.bodyCh)
	}
	if s.wsInputCh != nil {
		close(s.wsInputCh)
	}
	return nil
}

func extractHeaders(h http.Header) []*message.Header {

	// TODO should remove Content-Length??

	headers := make([]*message.Header, 0, len(h))
	for name, values := range h {
		headers = append(headers, &message.Header{
			Name:   name,
			Values: values,
		})
	}
	return headers
}

type bodyReader struct {
	in     <-chan *message.Message
	data   []byte
	offset int
	ctx    context.Context
}

func newBodyReader(ctx context.Context, in <-chan *message.Message) *bodyReader {
	return &bodyReader{
		in:  in,
		ctx: ctx,
	}
}

// Read reads from the tunnel input for this session
// It reads until p is filled or EOF or error
// Data from tunnel comes in chunks
func (b *bodyReader) Read(p []byte) (n int, err error) {
	for n < len(p) {
		if b.data == nil || b.offset == len(b.data) {
			// read new data
			select {
			case m, ok := <-b.in:
				if !ok {
					return n, io.EOF
				}
				if m.Type != message.Message_BODY {
					return n, fmt.Errorf("unexpected message type: %s", m.Type)
				}
				b.data = m.Data
				b.offset = 0
			case <-b.ctx.Done():
				return n, b.ctx.Err()
			}
			// TODO check disconnected?
		}
		x := copy(p[n:], b.data[b.offset:])
		n += x
		b.offset += x
	}
	return n, nil
}

func (b *bodyReader) Close() error {
	// TODO do nothing?
	return nil
}

func (s *session) requestBodyWriter(outputCh chan<- *message.Message) {
	buf := make([]byte, bufferSize)
	for {
		n, err := s.request.Body.Read(buf)
		if err != nil && err != io.EOF {
			logf("requestBodyWriter read error. %s err=%v", s.String(), err)
			outputCh <- &message.Message{
				Type:   message.Message_ERROR,
				Id:     s.id,
				Origin: s.origin,
			}
			break
		}
		outputCh <- &message.Message{
			Type:   message.Message_BODY,
			Id:     s.id,
			Data:   buf[0:n],
			Origin: s.origin,
			Eof:    err == io.EOF,
		}
		if err == io.EOF {
			logf("requestBodyWriter read EOF. %s", s.String())
			break
		}
	}
}

func (s *session) createWsCloseMessage(code websocket.StatusCode, msg string) *message.Message {
	m := &message.Message{
		Id:      s.id,
		Origin:  s.origin,
		Type:    message.Message_WS_CLOSE,
		Code:    uint32(code),
		Message: msg,
	}
	return m
}

func (s *session) wsReader(c *websocket.Conn, outputCh chan<- *message.Message) {
	for {
		// TODO ctx??
		wsType, reader, err := c.Reader(context.Background())
		if err != nil {
			logf("wsReader Reader error. %s err=%v", s, err)
			outputCh <- s.createWsCloseMessage(websocket.StatusInternalError, err.Error())
			break
		}
		logf("*** wsReader read. %s wsType=%d", s.String(), wsType)
		var wsMsgType message.Message_WsMsgType
		if wsType == websocket.MessageText {
			wsMsgType = message.Message_TEXT
		} else if wsType == websocket.MessageBinary {
			wsMsgType = message.Message_BINARY
		} else {
			logf("wsReader unknown message type. %s wsType=%d", s.String(), wsType)
			outputCh <- s.createWsCloseMessage(websocket.StatusUnsupportedData, "unknown message type")
			break
		}

		// send header first
		outputCh <- &message.Message{
			Type:      message.Message_WS_MSG_HEADER,
			Id:        s.id,
			Origin:    s.origin,
			WsMsgType: wsMsgType,
		}

		// send body of message
		for {
			buf := make([]byte, bufferSize)
			n, err := reader.Read(buf)
			if err != nil && err != io.EOF {
				logf("wsReader read error. %s err=%v", s, err)
				outputCh <- s.createWsCloseMessage(websocket.StatusInternalError, err.Error())
			}
			logf("*** wsReader n=%d err=%v buf=%s", n, err, string(buf[0:n]))
			outputCh <- &message.Message{
				Type:      message.Message_WS_MSG_DATA,
				Id:        s.id,
				Data:      buf[0:n],
				Origin:    s.origin,
				WsMsgType: wsMsgType,
				Eof:       err == io.EOF,
			}
			if err == io.EOF {
				break
			}
		}
	}
}

func (s *session) requestInitiator(ctx context.Context, m *message.Message, outputCh chan<- *message.Message) {
	// m is the request header

	// TODO add prefix for now!!!
	u := "https://localhost" + m.Url

	req, err := http.NewRequest(m.Method, u, nil)
	if err != nil {
		logf("mapper error creating request: %v", err)
		// TODO send error message
		return
	}
	// add headers
	for _, h := range m.Headers {
		req.Header[h.Name] = h.Values
	}
	// Set Content-Length explicitly as setting in header would be ignored
	req.ContentLength = m.ContentLength
	req.Body = newBodyReader(ctx, s.bodyCh)

	resp, err := s.tunnel.Client.Do(req)
	if err != nil {
		outputCh <- &message.Message{Id: s.id, Type: message.Message_SERVICE_UNAVAILABLE}
		logf("remoteCaller error. id=%d err=%v", s.id, err)
		return
	}
	defer resp.Body.Close()

	outputCh <- &message.Message{
		Id:      s.id,
		Type:    message.Message_RESPONSE_HEADER,
		Origin:  s.origin,
		Code:    uint32(resp.StatusCode),
		Proto:   resp.Proto,
		Headers: extractHeaders(resp.Header),
	}

	// read and send back body
	for {
		buf := make([]byte, bufferSize)
		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			logf("remoteCaller read error. %s err=%v", s.String(), err)
			outputCh <- &message.Message{Id: s.id, Origin: s.origin, Type: message.Message_ERROR}
			break
		}
		outputCh <- &message.Message{
			Id:     s.id,
			Origin: s.origin,
			Type:   message.Message_BODY,
			Data:   buf[0:n],
			Eof:    err == io.EOF,
		}
		if err == io.EOF {
			break
		}
	}

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
			s.Close()
		}
		for _, s := range rconn {
			s.Close()
		}
	}()

	for {
		select {
		case i, ok := <-inputCh:
			if !ok {
				return
			}
			if i.Type == message.Message_REQUEST_HEADER {
				s := &session{
					id:     i.Id,
					origin: message.Message_ORIGIN_REMOTE,
					tunnel: tn,
					bodyCh: make(chan *message.Message),
				}
				rconn[i.Id] = s
				go s.requestInitiator(ctx, i, outputCh)

			} else if i.Type == message.Message_RESPONSE_HEADER {
				s := lconn[i.Id]
				if s == nil {
					logf("mapper error: no session for id %d", i.Id)
					continue
				}
				resp := s.createResponse(i)
				s.responseCh <- resp
			} else if i.Type == message.Message_SERVICE_UNAVAILABLE {
				// Local initiated
				pch := lconn[i.Id].bodyCh
				delete(lconn, i.Id)
				pch <- i
				close(pch)
			} else if i.Type == message.Message_WS_CONNECT {
				logf("*** mapper connect. remote origin %v", i)
				s := &session{
					id:        i.Id,
					origin:    message.Message_ORIGIN_REMOTE,
					wsInputCh: make(chan *message.Message),
					tunnel:    tn,
				}
				rconn[i.Id] = s
				go s.wsConnector(i, outputCh)
			} else if i.Type == message.Message_WS_CONNECTED {
				logf("*** mapper connected. local origin %v", i)
				s := lconn[i.Id]
				go s.wsReader(s.wsConn, outputCh)
			} else if i.Type == message.Message_WS_MSG_DATA || i.Type == message.Message_WS_MSG_HEADER || i.Type == message.Message_WS_CLOSE {
				var m map[uint32]*session
				if i.Origin == message.Message_ORIGIN_LOCAL {
					// Incoming message from other side with local origin. Use remote map
					m = rconn
				} else {
					m = lconn
				}
				ch := m[i.Id].wsInputCh
				ch <- i
				if i.Type == message.Message_WS_CLOSE {
					delete(m, i.Id)
					// TODO need to close? As WS_CLOSE will make it stop
					close(ch)
				}
			} else if i.Type == message.Message_ERROR {
				// TODO
				logf("mapper error. %s", i.String())
			} else if i.Type == message.Message_BODY {
				logf("*** mapper received. i=%v", i)
				var m map[uint32]*session
				if i.Origin == message.Message_ORIGIN_LOCAL {
					// Incoming message from other side with local origin. Use remote map
					m = rconn
				} else {
					m = lconn
				}
				pwch := m[i.Id].bodyCh
				pwch <- i

				if i.Eof {
					delete(m, i.Id)
					close(pwch)
				}
			}
		case s := <-tn.initiateRequestCh:
			// Find next available id.
			used := true
			for i := int32(0); i < math.MaxInt32; i++ {
				availableId++
				if _, used = lconn[availableId]; !used {
					break
				}
			}
			if used {
				resp := createErrorResponse(http.StatusTooManyRequests, fmt.Errorf("too many connections"))
				s.responseCh <- resp
				// TODO close required? because the function only takes one response and ends
				close(s.responseCh)
				logf("Too many connections")
				continue
			}
			// Setup session
			s.id = availableId
			lconn[s.id] = s
			// Send request
			outputCh <- createRequestHeader(s.id, s.request)
			go s.requestBodyWriter(outputCh)
		case s := <-tn.initiateWsCh:
			// Find next available id.
			used := true
			for i := int32(0); i < math.MaxInt32; i++ {
				availableId++
				if _, used = lconn[availableId]; !used {
					break
				}
			}
			if used {
				// TODO create error message for ws. going to wsInputCh
				close(s.wsInputCh)
				logf("Too many connections")
				continue
			}
			// Setup session
			s.id = availableId
			lconn[s.id] = s

			outputCh <- &message.Message{Id: s.id,
				Type:    message.Message_WS_CONNECT,
				Url:     s.request.URL.String(),
				Headers: extractHeaders(s.request.Header),
			}

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

	tn.initiateRequestCh = make(chan *session)
	tn.initiateWsCh = make(chan *session)
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
