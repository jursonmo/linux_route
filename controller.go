package linuxroute

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

// Controller ties together a RouteManager and a RouteStore
// to support "receive full routes -> apply delta" workflows.
type Controller struct {
	Manager RouteManager
	Store   RouteStore
}

type ReconcileResult struct {
	Diff DiffResult
}

func NewController(manager RouteManager, store RouteStore) *Controller {
	return &Controller{
		Manager: manager,
		Store:   store,
	}
}

func NewControllerWithFileStore(path string) *Controller {
	return NewController(&IPRouteManager{}, &FileStore{Path: path})
}

// Reconcile compares desiredRoutes with previously saved routes, then:
// - deletes routes that should no longer exist
// - adds routes that are missing
// - saves desiredRoutes as the new baseline (only if apply succeeded)
func (c Controller) Reconcile(ctx context.Context, desiredRoutes []Route) (ReconcileResult, error) {
	if c.Manager == nil {
		return ReconcileResult{}, fmt.Errorf("manager is nil")
	}
	if c.Store == nil {
		return ReconcileResult{}, fmt.Errorf("store is nil")
	}

	oldRoutes, err := c.Store.Load()
	if err != nil {
		return ReconcileResult{}, fmt.Errorf("load old routes: %w", err)
	}

	diff, err := DiffRoutes(oldRoutes, desiredRoutes)
	if err != nil {
		return ReconcileResult{}, err
	}

	// Track the actual set applied to RouteManager, so Store stays consistent
	// with what really succeeded.
	applied := make(map[string]Route, len(diff.Unchanged)+len(diff.ToDel)+len(diff.ToAdd))
	for _, r := range diff.Unchanged {
		k, _ := r.Key() // already normalized/validated by DiffRoutes
		applied[k] = r
	}
	for _, r := range diff.ToDel {
		k, _ := r.Key()
		applied[k] = r
	}

	// Deletes first to avoid "file exists" / conflicts.
	var applyErrs []error
	for _, r := range diff.ToDel {
		if err := c.Manager.Delete(ctx, r); err != nil {
			applyErrs = append(applyErrs, fmt.Errorf("delete route (%+v): %w", r, err))
			continue
		}
		k, _ := r.Key()
		delete(applied, k)
	}
	for _, r := range diff.ToAdd {
		if err := c.Manager.Add(ctx, r); err != nil {
			applyErrs = append(applyErrs, fmt.Errorf("add route (%+v): %w", r, err))
			continue
		}
		k, _ := r.Key()
		applied[k] = r
	}

	appliedRoutes := make([]Route, 0, len(applied))
	for _, r := range applied {
		appliedRoutes = append(appliedRoutes, r)
	}
	sort.Slice(appliedRoutes, func(i, j int) bool {
		ki, _ := appliedRoutes[i].Key()
		kj, _ := appliedRoutes[j].Key()
		return ki < kj
	})

	if err := c.Store.Save(appliedRoutes); err != nil {
		applyErrs = append(applyErrs, fmt.Errorf("save applied routes: %w", err))
	}

	return ReconcileResult{Diff: diff}, errors.Join(applyErrs...)
}
