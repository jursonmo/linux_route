
### 这个项目的功能简单，代码几乎都是AI生成的。包括这个readme.md, 都是AI写的。

### 这个库解决什么问题？

当你的控制平面/配置中心每次下发的是一份**全量路由列表**（desired routes），而你希望在机器上以**幂等、可回滚、最小变更**的方式把系统路由表收敛到目标状态时，可以用本库的统一流程：

- **保存基线**：把“上一次已经成功应用的全量路由”持久化（`RouteStore`）
- **计算差量**：新全量到来时，用 `DiffRoutes(old, desired)` 得到要删/要加/不变（集合语义）
- **应用差量**：按“**先删后加**”的顺序调用 `RouteManager`（避免冲突/重复）
- **提交新基线**：只有当应用成功后，才把 desired 保存为下一次对比的基线（`Controller.Reconcile` 已内置）

### 重要语义：full-key（集合）对比

`Route.Key()` 使用的是“**full-key**”语义（`dst + gw + dev + table + metric + src + scope + type + proto`）。这意味着：

- **支持同一目的网段多条路由**（例如不同 gateway / metric），它们会被视作不同条目
- 当你把某条路由的 gateway/metric 等字段改掉时，diff 的结果会表现为：**删旧 + 加新**

### reconcile_full_routes（推荐先看这个）

`example/reconcile_full_routes` 演示了完整调用链，且使用 `dryRunManager`：

- **不修改系统路由表**，只记录“将要执行”的操作
- 因此可以在 macOS/Windows/Linux 任意平台跑通，理解流程

#### 运行

在仓库根目录执行：

```bash
go run ./example/reconcile_full_routes
```

你会看到类似输出（以示例代码的路由为准）：

```text
ToDel=2 ToAdd=2 Unchanged=1
Operations (delete first, then add):
  DEL dst=10.10.0.0/16|gw=10.0.0.2|dev=eth0|table=0|metric=100|src=|scope=|type=|proto=
  DEL dst=default|gw=10.0.0.1|dev=eth0|table=0|metric=0|src=|scope=|type=|proto=
  ADD dst=192.168.2.0/24|gw=10.0.0.3|dev=eth0|table=0|metric=0|src=|scope=|type=|proto=
  ADD dst=default|gw=10.0.0.254|dev=eth0|table=0|metric=0|src=|scope=|type=|proto=
Saved baseline routes: 3
```

### 在 Linux 上真实修改系统路由表（IPRouteManager）

本库在 Linux 下提供了 `IPRouteManager`（基于 `github.com/vishvananda/netlink`）来对系统路由表做 `List/Add/Delete`：

- 文件：`ip_manager_linux.go`（仅 Linux 编译）
- 非 Linux 平台会得到明确错误：`IPRouteManager is supported only on linux`

一个最小化的落地示例（伪代码/模板，可直接拷到你的服务里）：

```go
package main

import (
  "context"
  "log"

  linuxroute "github.com/jursonmo/linux_route"
)

func main() {
  ctx := context.Background()

  ctrl := linuxroute.Controller{
    Manager: linuxroute.IPRouteManager{},
    Store: linuxroute.FileStore{Path: "/var/lib/linux-route/baseline.json"},
  }

  desired := []linuxroute.Route{
    {Dst: "default", Gateway: "10.0.0.1", Device: "eth0"},
    {Dst: "10.10.0.0/16", Gateway: "10.0.0.2", Device: "eth0", Metric: 100},
  }

  if _, err := ctrl.Reconcile(ctx, desired); err != nil {
    log.Fatal(err)
  }
}
```

#### Linux 权限与注意事项

- **需要足够权限修改路由表**：通常需要 root 或 `CAP_NET_ADMIN`
- `IPRouteManager.Add` 内部用的是 `netlink.RouteReplace`：更贴近“幂等写入”的期望
- `Controller.Reconcile` 默认采用“先删后加”，降低 “file exists / 冲突” 类错误概率

### Route 数据格式（给 FileStore / JSON 的约定）

`FileStore` 会把 routes 以 JSON 数组形式持久化，并在保存前做 `Normalize()`，让字段更稳定（例如 CIDR 规范化、IP 格式化、大小写等）。

字段要点：

- **dst**：必填；支持 `"default"` 或 CIDR（如 `"10.0.0.0/24"`、`"2001:db8::/64"`）
- **gateway/src**：可选 IP
- **device**：可选但推荐（避免歧义）
- **table/metric**：可选；0 表示“未指定/默认”
- **scope/type/proto**：可选（为了更完整的路由表达与兼容）

示例 `baseline.json`：

```json
[
  { "dst": "default", "gateway": "10.0.0.1", "device": "eth0" },
  { "dst": "10.10.0.0/16", "gateway": "10.0.0.2", "device": "eth0", "metric": 100 }
]
```

### 下一步建议

- **先跑 `reconcile_full_routes`**：确认你理解 full-key 与 diff 的行为
- **再替换为真实的 `IPRouteManager + FileStore`**：把基线落盘，接入你自己的“全量路由下发”来源

