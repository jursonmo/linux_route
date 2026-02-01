//go:build !linux

package linuxroute

import (
	"context"
	"fmt"
)

// IPRouteManager is not supported on non-Linux platforms.
type IPRouteManager struct {
	IPPath string
}

func (m IPRouteManager) List(ctx context.Context) ([]Route, error) {
	return nil, fmt.Errorf("IPRouteManager is supported only on linux")
}

func (m IPRouteManager) Add(ctx context.Context, r Route) error {
	return fmt.Errorf("IPRouteManager is supported only on linux")
}

func (m IPRouteManager) Delete(ctx context.Context, r Route) error {
	return fmt.Errorf("IPRouteManager is supported only on linux")
}
