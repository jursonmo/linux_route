package linuxroute

import (
	"context"
	"strings"
	"testing"
)

type fakeManager struct {
	ops   []string
	addE  error
	delE  error
	list  []Route
	listE error
}

func (m *fakeManager) List(ctx context.Context) ([]Route, error) {
	if m.listE != nil {
		return nil, m.listE
	}
	return append([]Route(nil), m.list...), nil
}

func (m *fakeManager) Add(ctx context.Context, r Route) error {
	k, _ := r.Key()
	m.ops = append(m.ops, "add "+k)
	return m.addE
}

func (m *fakeManager) Delete(ctx context.Context, r Route) error {
	k, _ := r.Key()
	m.ops = append(m.ops, "del "+k)
	return m.delE
}

func TestControllerReconcile_OrderAndSave(t *testing.T) {
	ctx := context.Background()

	store := &MemoryStore{}
	if err := store.Save([]Route{
		{Dst: "default", Gateway: "10.0.0.1", Device: "eth0"},
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	mgr := &fakeManager{}
	c := Controller{Manager: mgr, Store: store}

	desired := []Route{
		{Dst: "192.168.2.0/24", Gateway: "10.0.0.2", Device: "eth0"},
	}
	got, err := c.Reconcile(ctx, desired)
	if err != nil {
		t.Fatalf("Reconcile() error: %v", err)
	}
	if len(got.Diff.ToDel) != 1 || len(got.Diff.ToAdd) != 1 {
		t.Fatalf("unexpected diff: %+v", got.Diff)
	}

	if len(mgr.ops) != 2 {
		t.Fatalf("ops len = %d, want 2; ops=%v", len(mgr.ops), mgr.ops)
	}
	if !strings.HasPrefix(mgr.ops[0], "del ") || !strings.HasPrefix(mgr.ops[1], "add ") {
		t.Fatalf("expected del then add, got ops=%v", mgr.ops)
	}

	after, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load() error: %v", err)
	}
	if len(after) != 1 || after[0].Dst != "192.168.2.0/24" {
		t.Fatalf("store not updated, got=%+v", after)
	}
}
