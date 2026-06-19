package daemon

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"sync"
)

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     int             `json:"id"`
}

type Response struct {
	ID     int    `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type Handler func(req Request) Response

type SocketServer struct {
	sockPath string
	listener net.Listener
	handler  Handler
	daemon   *Daemon
	mu       sync.Mutex
}

func NewSocketServer(sockPath string) (*SocketServer, error) {
	os.Remove(sockPath)

	l, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, err
	}

	return &SocketServer{
		sockPath: sockPath,
		listener: l,
	}, nil
}

func (s *SocketServer) SetHandler(h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = h
}

func (s *SocketServer) SetDaemon(d *Daemon) {
	s.daemon = d
}

func (s *SocketServer) Serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *SocketServer) handleConn(conn net.Conn) {
	// Create a channel for this client to receive broadcasts
	clientCh := make(chan []byte, 100)

	// Register with daemon for broadcasts
	if s.daemon != nil {
		s.daemon.RegisterClient(clientCh)
		defer s.daemon.UnregisterClient(clientCh)
	}

	// Start goroutine to send broadcasts to this client
	go func() {
		for msg := range clientCh {
			msg = append(msg, '\n')
			conn.Write(msg)
		}
	}()

	// Read requests from client
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		s.mu.Lock()
		h := s.handler
		s.mu.Unlock()

		var resp Response
		if h != nil {
			resp = h(req)
		} else {
			resp = Response{
				ID:    req.ID,
				Error: "no handler",
			}
		}

		data, _ := json.Marshal(resp)
		data = append(data, '\n')
		conn.Write(data)
	}
}

func (s *SocketServer) Close() error {
	return s.listener.Close()
}
