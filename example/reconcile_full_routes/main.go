package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	linuxroute "github.com/jursonmo/linux_route"
)

// dryRunManager is a demo RouteManager: it does not touch the OS,
// it only records operations and maintains an in-memory "current routes set".
type dryRunManager struct {
	current map[string]linuxroute.Route
	ops     []string
}

func newDryRunManager(seed []linuxroute.Route) *dryRunManager {
	m := &dryRunManager{current: make(map[string]linuxroute.Route)}
	for _, r := range seed {
		n, err := r.Normalize()
		if err != nil {
			continue
		}
		k, _ := n.Key()
		m.current[k] = n
	}
	return m
}

func (m *dryRunManager) List(ctx context.Context) ([]linuxroute.Route, error) {
	out := make([]linuxroute.Route, 0, len(m.current))
	for _, r := range m.current {
		out = append(out, r)
	}
	return out, nil
}

func (m *dryRunManager) Add(ctx context.Context, r linuxroute.Route) error {
	n, err := r.Normalize()
	if err != nil {
		return err
	}
	k, _ := n.Key()
	m.current[k] = n
	m.ops = append(m.ops, "ADD "+k)
	return nil
}

func (m *dryRunManager) Delete(ctx context.Context, r linuxroute.Route) error {
	n, err := r.Normalize()
	if err != nil {
		return err
	}
	k, _ := n.Key()
	delete(m.current, k)
	m.ops = append(m.ops, "DEL "+k)
	return nil
}

func main() {
	ctx := context.Background()

	// 假设上一次我们“全量下发并应用”的路由如下（保存在 store 里）：
	oldBaseline := []linuxroute.Route{
		{Dst: "default", Gateway: "10.0.0.1", Device: "eth0"},
		{Dst: "10.10.0.0/16", Gateway: "10.0.0.2", Device: "eth0", Metric: 100},
		{Dst: "192.168.3.0/24", Gateway: "10.0.0.4", Device: "eth0"},
	}

	// 这次收到一组新的“全量路由”。
	desired := []linuxroute.Route{
		// default 变更了 next hop（在当前库的 full-key 语义下，会表现为：删旧 + 加新）
		{Dst: "default", Gateway: "10.0.0.254", Device: "eth0"},
		// 新增一条
		{Dst: "192.168.2.0/24", Gateway: "10.0.0.3", Device: "eth0"},
		// 不变的路由
		{Dst: "192.168.3.0/24", Gateway: "10.0.0.4", Device: "eth0"},
	}

	// store 用来保存“上次已应用的全量路由”，用于下一次 diff。
	store := &linuxroute.MemoryStore{}
	if err := store.Save(oldBaseline); err != nil {
		log.Fatalf("seed store: %v", err)
	}

	// manager 负责对 OS 路由表做增删查改；这个例子用 dry-run 版本演示调用顺序。
	manager := newDryRunManager(nil)

	ctrl := linuxroute.Controller{
		Manager: manager,
		Store:   store,
	}

	res, err := ctrl.Reconcile(ctx, desired)
	if err != nil {
		log.Fatalf("reconcile: %v", err)
	}

	fmt.Printf("ToDel=%d ToAdd=%d Unchanged=%d\n", len(res.Diff.ToDel), len(res.Diff.ToAdd), len(res.Diff.Unchanged))
	fmt.Println("Operations (delete first, then add):")
	fmt.Println("  " + strings.Join(manager.ops, "\n  "))

	newBaseline, _ := store.Load()
	fmt.Printf("Saved baseline routes: %d\n", len(newBaseline))
}
