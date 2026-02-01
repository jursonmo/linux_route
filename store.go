package linuxroute

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// RouteStore persists the "last applied" full route set.
// It is used for diffing when a new full route set arrives.
type RouteStore interface {
	Load() ([]Route, error)
	Save(routes []Route) error
}

// FileStore stores routes as JSON on disk (atomic write).
type FileStore struct {
	Path string
}

func (s FileStore) Load() ([]Route, error) {
	if s.Path == "" {
		return nil, fmt.Errorf("filestore path is empty")
	}
	b, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if len(b) == 0 {
		return nil, nil
	}
	var routes []Route
	if err := json.Unmarshal(b, &routes); err != nil {
		return nil, err
	}
	return routes, nil
}

func (s FileStore) Save(routes []Route) error {
	if s.Path == "" {
		return fmt.Errorf("filestore path is empty")
	}

	// Normalize before persisting to avoid key churn across runs.
	norm := make([]Route, 0, len(routes))
	for i, r := range routes {
		n, err := r.Normalize()
		if err != nil {
			return fmt.Errorf("routes[%d]: %w", i, err)
		}
		norm = append(norm, n)
	}

	b, err := json.MarshalIndent(norm, "", "  ")
	if err != nil {
		return err
	}

	b = append(b, '\n')

	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}

	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
