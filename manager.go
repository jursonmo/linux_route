package linuxroute

import "context"

// RouteManager performs CRUD against the OS routing table.
//
// For the "full routes input" use-case, you typically:
// - Load old routes from RouteStore
// - Diff
// - Apply deletes/adds through RouteManager
// - Save desired routes back to RouteStore
type RouteManager interface {
	// List returns current routes on the system.
	List(ctx context.Context) ([]Route, error)
	Add(ctx context.Context, r Route) error
	Delete(ctx context.Context, r Route) error
}
