package main

import (
	"github.com/Nameer-kp/go-load-balancer/backend"
	"github.com/Nameer-kp/go-load-balancer/helpers"
	"log"
	"net/http"
	"net/url"
	"sync/atomic"
)

type ServerPool struct {
	backends []*backend.Backend
	current  uint64
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}
func (s *ServerPool) GetNextPeer() *backend.Backend {
	next := s.NextIndex()
	l := len(s.backends) + next
	for i := next; i < l; i++ {
		idx := i % len(s.backends)
		if s.backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.backends[idx]
		}
	}
	return nil
}
func (s *ServerPool) HealthCheck() {
	for _, b := range s.backends {
		status := "up"
		alive := helpers.IsBackendAlive(b.URL)
		b.SetAlive(alive)
		if !alive {
			status = "down"
		}
		log.Printf("%s [%s]\n", b.URL, status)
	}
}
func (s *ServerPool) AddBackend(backend *backend.Backend) {
	s.backends = append(s.backends, backend)
}
func (s *ServerPool) MarkBackendStatus(backendUrl *url.URL, alive bool) {
	for _, b := range s.backends {
		if b.URL.String() == backendUrl.String() {
			b.SetAlive(alive)
		}
	}
}
func lb(w http.ResponseWriter, r *http.Request) {
	attempts := helpers.GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}
	peer := serverpool.GetNextPeer()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}
