package stream

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultAwaitConnectionTimeout = time.Second * 5
const streamPathPrefix = "/streams/"

type Server struct {
	server    *http.Server
	baseURL   string
	endpoints map[string]*Endpoint
	lastID    int
}

func NewServer(host string, port int) *Server {
	s := &Server{
		server: &http.Server{
			Addr: fmt.Sprintf(":%d", port),
		},
		baseURL:   fmt.Sprintf("http://%s:%d", host, port),
		endpoints: make(map[string]*Endpoint),
	}
	s.server.Handler = http.HandlerFunc(s.serveHTTP)
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond * 10)
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			panic("Could not detect own listener at " + s.server.Addr)
		case <-ticker.C:
			resp, err := http.DefaultClient.Head(fmt.Sprintf("http://localhost:%d", port))
			if err == nil && resp.StatusCode == 200 {
				return s
			}
		}
	}
}

func (s *Server) NewEndpoint(logger *log.Logger) *Endpoint {
	s.lastID++
	endpointID := strconv.Itoa(s.lastID)

	e := &Endpoint{
		owner:  s,
		id:     endpointID,
		Errors: make(chan error, 10),
		logger: logger,
		cxnCh:  make(chan *IncomingConnection, 10),
		URL:    strings.TrimSuffix(s.baseURL, "/") + streamPathPrefix + endpointID,
	}
	s.endpoints[endpointID] = e

	return e
}

func (s *Server) serveHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == "HEAD" {
		w.WriteHeader(200)
		return
	}
	if !strings.HasPrefix(req.URL.Path, streamPathPrefix) {
		w.WriteHeader(400)
		return
	}
	endpointID := strings.TrimPrefix(req.URL.Path, streamPathPrefix)
	if e := s.endpoints[endpointID]; e != nil {
		e.serveHTTP(w, req)
		return
	}
	w.WriteHeader(404)
}
