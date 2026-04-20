package telemetry

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"sync"
)

// Handler consumes frames received from worker connections. A frame's worker
// identity is carried inside the frame itself.
type Handler func(f Frame)

// Server is the harness-side unix-socket listener that multiplexes frames
// from many worker connections into a single Handler callback.
type Server struct {
	path    string
	lis     net.Listener
	handler Handler
	wg      sync.WaitGroup
}

// Listen opens the unix socket at path. The file is created fresh (any
// pre-existing entry is removed first) and os.Remove is called on Close.
func Listen(path string, h Handler) (*Server, error) {
	_ = os.Remove(path)
	lis, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	s := &Server{path: path, lis: lis, handler: h}
	s.wg.Add(1)
	go s.acceptLoop()
	return s, nil
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.lis.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			// Transient accept error — bail; callers can restart if needed.
			return
		}
		s.wg.Add(1)
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer s.wg.Done()
	defer func() { _ = conn.Close() }()
	for {
		f, err := ReadFrame(conn)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			return
		}
		s.handler(f)
	}
}

// Path returns the socket path the server is listening on.
func (s *Server) Path() string { return s.path }

// Close stops accepting connections and waits for in-flight handlers.
func (s *Server) Close(_ context.Context) error {
	err := s.lis.Close()
	s.wg.Wait()
	_ = os.Remove(s.path)
	return err
}
