package linuxroute

import "testing"

func TestRouteNormalize(t *testing.T) {
	r, err := (Route{
		Dst:     "10.0.0.1/24",
		Gateway: "192.168.1.1",
		Device:  "eth0",
		Table:   100,
		Metric:  10,
		Src:     "10.0.0.5",
	}).Normalize()
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
	if r.Dst != "10.0.0.0/24" {
		t.Fatalf("Dst not normalized, got %q", r.Dst)
	}
}

func TestDiffRoutes(t *testing.T) {
	oldRoutes := []Route{
		{Dst: "10.0.0.0/24", Gateway: "192.168.1.1", Device: "eth0", Table: 100},
		{Dst: "default", Gateway: "10.0.0.1", Device: "eth0"},
	}
	desiredRoutes := []Route{
		{Dst: "10.0.0.0/24", Gateway: "192.168.1.1", Device: "eth0", Table: 100}, // unchanged
		{Dst: "192.168.2.0/24", Gateway: "10.0.0.2", Device: "eth0"},             // add
	}

	res, err := DiffRoutes(oldRoutes, desiredRoutes)
	if err != nil {
		t.Fatalf("DiffRoutes() error: %v", err)
	}

	if len(res.Unchanged) != 1 {
		t.Fatalf("Unchanged len = %d, want 1", len(res.Unchanged))
	}
	if len(res.ToAdd) != 1 {
		t.Fatalf("ToAdd len = %d, want 1", len(res.ToAdd))
	}
	if len(res.ToDel) != 1 {
		t.Fatalf("ToDel len = %d, want 1", len(res.ToDel))
	}

	if res.ToDel[0].Dst != "default" {
		t.Fatalf("ToDel[0].Dst = %q, want %q", res.ToDel[0].Dst, "default")
	}
}
