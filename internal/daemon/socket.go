package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"

	"github.com/amirkhaki/makima/internal/log"
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
	lockPath string
	listener net.Listener
	handler  Handler
	daemon   *Daemon
	mu       sync.Mutex
}

func NewSocketServer(sockPath string) (*SocketServer, error) {
	lockPath := sockPath + ".lock"

	// Try to acquire lock
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			// Check if the lock is stale (daemon crashed)
			if isStaleLock(lockPath) {
				os.Remove(lockPath)
				lockFile, err = os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if err != nil {
					return nil, fmt.Errorf("failed to acquire lock: %w", err)
				}
			} else {
				return nil, fmt.Errorf("another daemon is already running (lock file: %s)", lockPath)
			}
		} else {
			return nil, fmt.Errorf("failed to create lock file: %w", err)
		}
	}

	// Write PID to lock file
	fmt.Fprintf(lockFile, "%d", os.Getpid())
	lockFile.Close()

	// Remove existing socket
	os.Remove(sockPath)

	l, err := net.Listen("unix", sockPath)
	if err != nil {
		os.Remove(lockPath)
		return nil, err
	}

	return &SocketServer{
		sockPath: sockPath,
		lockPath: lockPath,
		listener: l,
	}, nil
}

func isStaleLock(lockPath string) bool {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return true
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return true
	}

	// Check if process is still running
	process, err := os.FindProcess(pid)
	if err != nil {
		return true
	}

	err = process.Signal(syscall.Signal(0))
	return err != nil
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
	log.Info("socket: listening on %s", s.sockPath)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Error("socket: accept error: %v", err)
			return
		}
		log.Info("socket: client connected from %s", conn.RemoteAddr())
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
	err := s.listener.Close()
	os.Remove(s.lockPath)
	os.Remove(s.sockPath)
	return err
}
