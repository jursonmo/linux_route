//go:build linux

package linuxroute

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// IPRouteManager implements RouteManager using the open-source netlink library
// (github.com/vishvananda/netlink).
type IPRouteManager struct {
	// IPPath is kept for backward compatibility, but unused in the netlink implementation.
	IPPath string
}

func (m IPRouteManager) List(ctx context.Context) ([]Route, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	nlRoutes, err := netlink.RouteListFiltered(netlink.FAMILY_ALL, &netlink.Route{}, 0)
	if err != nil {
		return nil, err
	}

	routes := make([]Route, 0, len(nlRoutes))
	for _, nr := range nlRoutes {
		r, err := fromNetlinkRoute(nr)
		if err != nil {
			continue
		}
		routes = append(routes, r)
	}

	return routes, nil
}

func (m IPRouteManager) Add(ctx context.Context, r Route) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	nlr, err := toNetlinkRoute(r)
	if err != nil {
		return err
	}
	return netlink.RouteReplace(&nlr)
}

func (m IPRouteManager) Delete(ctx context.Context, r Route) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	nlr, err := toNetlinkRoute(r)
	if err != nil {
		return err
	}
	err = netlink.RouteDel(&nlr)
	if err != nil {
		if errors.Is(err, syscall.ESRCH) || errors.Is(err, syscall.ENOENT) {
			return nil
		}
		s := strings.ToLower(err.Error())
		if strings.Contains(s, "no such process") || strings.Contains(s, "not found") {
			return nil
		}
		return err
	}
	return nil
}

func toNetlinkRoute(r Route) (netlink.Route, error) {
	n, err := r.Normalize()
	if err != nil {
		return netlink.Route{}, err
	}

	var nr netlink.Route

	if n.Dst != "default" {
		_, ipNet, err := net.ParseCIDR(n.Dst)
		if err != nil {
			return netlink.Route{}, fmt.Errorf("invalid dst %q: %w", n.Dst, err)
		}
		nr.Dst = ipNet
	}

	if n.Gateway != "" {
		gw := net.ParseIP(n.Gateway)
		if gw == nil {
			return netlink.Route{}, fmt.Errorf("invalid gateway %q", n.Gateway)
		}
		nr.Gw = gw
	}

	if n.Device != "" {
		link, err := netlink.LinkByName(n.Device)
		if err != nil {
			return netlink.Route{}, fmt.Errorf("link %q: %w", n.Device, err)
		}
		nr.LinkIndex = link.Attrs().Index
	}

	if n.Table != 0 {
		nr.Table = n.Table
	}
	if n.Metric != 0 {
		nr.Priority = n.Metric
	}
	if n.Src != "" {
		src := net.ParseIP(n.Src)
		if src == nil {
			return netlink.Route{}, fmt.Errorf("invalid src %q", n.Src)
		}
		nr.Src = src
	}

	if n.Scope != "" {
		sc, ok := parseScope(n.Scope)
		if !ok {
			return netlink.Route{}, fmt.Errorf("unsupported scope %q", n.Scope)
		}
		nr.Scope = sc
	}

	if n.Type != "" {
		rt, ok := parseRouteType(n.Type)
		if !ok {
			return netlink.Route{}, fmt.Errorf("unsupported type %q", n.Type)
		}
		nr.Type = rt
	} else {
		nr.Type = unix.RTN_UNICAST
	}

	if n.Proto != "" {
		p, ok := parseProtocol(n.Proto)
		if !ok {
			return netlink.Route{}, fmt.Errorf("unsupported proto %q", n.Proto)
		}
		nr.Protocol = netlink.RouteProtocol(p)
	}

	return nr, nil
}

func fromNetlinkRoute(nr netlink.Route) (Route, error) {
	r := Route{
		Table:  nr.Table,
		Metric: nr.Priority,
	}

	if nr.Dst == nil {
		r.Dst = "default"
	} else {
		r.Dst = nr.Dst.String()
	}

	if len(nr.Gw) != 0 {
		r.Gateway = nr.Gw.String()
	}

	if len(nr.Src) != 0 {
		r.Src = nr.Src.String()
	}

	if nr.LinkIndex != 0 {
		link, err := netlink.LinkByIndex(nr.LinkIndex)
		if err == nil && link != nil && link.Attrs() != nil {
			r.Device = link.Attrs().Name
		}
	}

	r.Scope = scopeString(nr.Scope)
	r.Type = routeTypeString(nr.Type)
	if nr.Protocol != 0 {
		r.Proto = protocolString(int(nr.Protocol))
	}

	return r.Normalize()
}

func parseScope(s string) (netlink.Scope, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "global", "universe":
		return netlink.SCOPE_UNIVERSE, true
	case "site":
		return netlink.SCOPE_SITE, true
	case "link":
		return netlink.SCOPE_LINK, true
	case "host":
		return netlink.SCOPE_HOST, true
	case "nowhere":
		return netlink.SCOPE_NOWHERE, true
	default:
		return 0, false
	}
}

func scopeString(sc netlink.Scope) string {
	switch sc {
	case netlink.SCOPE_UNIVERSE:
		return "global"
	case netlink.SCOPE_SITE:
		return "site"
	case netlink.SCOPE_LINK:
		return "link"
	case netlink.SCOPE_HOST:
		return "host"
	case netlink.SCOPE_NOWHERE:
		return "nowhere"
	default:
		return ""
	}
}

func parseRouteType(s string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "unicast":
		return unix.RTN_UNICAST, true
	case "blackhole":
		return unix.RTN_BLACKHOLE, true
	case "unreachable":
		return unix.RTN_UNREACHABLE, true
	case "prohibit":
		return unix.RTN_PROHIBIT, true
	default:
		return 0, false
	}
}

func routeTypeString(t int) string {
	switch t {
	case unix.RTN_BLACKHOLE:
		return "blackhole"
	case unix.RTN_UNREACHABLE:
		return "unreachable"
	case unix.RTN_PROHIBIT:
		return "prohibit"
	default:
		return ""
	}
}

func parseProtocol(p string) (int, bool) {
	p = strings.ToLower(strings.TrimSpace(p))
	if p == "" {
		return 0, true
	}
	if n, err := strconv.Atoi(p); err == nil {
		if n < 0 || n > 255 {
			return 0, false
		}
		return n, true
	}
	switch p {
	case "kernel":
		return unix.RTPROT_KERNEL, true
	case "boot":
		return unix.RTPROT_BOOT, true
	case "static":
		return unix.RTPROT_STATIC, true
	case "dhcp":
		return unix.RTPROT_DHCP, true
	default:
		return 0, false
	}
}

func protocolString(p int) string {
	switch p {
	case unix.RTPROT_KERNEL:
		return "kernel"
	case unix.RTPROT_BOOT:
		return "boot"
	case unix.RTPROT_STATIC:
		return "static"
	case unix.RTPROT_DHCP:
		return "dhcp"
	default:
		return strconv.Itoa(p)
	}
}
