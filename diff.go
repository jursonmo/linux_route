package linuxroute

import (
	"fmt"
	"sort"
)

// DiffResult is the plan computed from oldRoutes -> desiredRoutes.
type DiffResult struct {
	// ToAdd are routes in desired but not in old (after normalization).
	ToAdd []Route
	// ToDel are routes in old but not in desired (after normalization).
	ToDel []Route
	// Unchanged are routes present in both sets (after normalization).
	Unchanged []Route
}

// DiffRoutes computes a set-diff between oldRoutes and desiredRoutes.
//
// It uses Route.Key() (full-key) semantics:
// - ToDel: routes in old but not in desired
// - ToAdd: routes in desired but not in old
func DiffRoutes(oldRoutes, desiredRoutes []Route) (DiffResult, error) {
	oldMap := make(map[string]Route, len(oldRoutes))
	for i, r := range oldRoutes {
		n, err := r.Normalize()
		if err != nil {
			return DiffResult{}, fmt.Errorf("oldRoutes[%d]: %w", i, err)
		}
		k, _ := n.Key() // already normalized/validated
		oldMap[k] = n
	}

	newMap := make(map[string]Route, len(desiredRoutes))
	for i, r := range desiredRoutes {
		n, err := r.Normalize()
		if err != nil {
			return DiffResult{}, fmt.Errorf("desiredRoutes[%d]: %w", i, err)
		}
		k, _ := n.Key()
		newMap[k] = n
	}

	var res DiffResult
	for k, oldR := range oldMap {
		if _, ok := newMap[k]; ok {
			res.Unchanged = append(res.Unchanged, oldR)
		} else {
			res.ToDel = append(res.ToDel, oldR)
		}
	}
	for k, newR := range newMap {
		if _, ok := oldMap[k]; !ok {
			res.ToAdd = append(res.ToAdd, newR)
		}
	}

	// Make output deterministic.
	sortRoutes := func(rr []Route) {
		sort.Slice(rr, func(i, j int) bool {
			ki, _ := rr[i].Key()
			kj, _ := rr[j].Key()
			return ki < kj
		})
	}

	sortRoutes(res.ToDel)
	sortRoutes(res.ToAdd)
	sortRoutes(res.Unchanged)

	return res, nil
}
