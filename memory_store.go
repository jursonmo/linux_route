package linuxroute

import "sync"

// MemoryStore is an in-memory RouteStore (useful for tests or embedding).
type MemoryStore struct {
	mu     sync.Mutex
	routes []Route
}

func (s *MemoryStore) Load() ([]Route, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Route, 0, len(s.routes))
	out = append(out, s.routes...)
	return out, nil
}

func (s *MemoryStore) Save(routes []Route) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes = append([]Route(nil), routes...)
	return nil
}
