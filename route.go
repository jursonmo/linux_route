package linuxroute

import (
	"fmt"
	"net"
	"strings"
)

// Route describes a Linux route entry in a mostly "ip route" compatible form.
//
// Notes:
// - Dst is required. Use "default" or a CIDR (e.g. "10.0.0.0/24", "2001:db8::/64").
// - Gateway/Src are optional IPs.
// - Device (dev) is optional but recommended for unambiguous routes.
// - Table/Metric are optional; 0 means "unspecified/default".
type Route struct {
	Dst     string `json:"dst"`
	Gateway string `json:"gateway,omitempty"`
	Device  string `json:"device,omitempty"`
	Table   int    `json:"table,omitempty"`
	Metric  int    `json:"metric,omitempty"`
	Src     string `json:"src,omitempty"`

	// Extra fields for forward compatibility / completeness.
	Scope string `json:"scope,omitempty"`
	Type  string `json:"type,omitempty"`
	Proto string `json:"proto,omitempty"`
}

// Normalize canonicalizes fields so diffing is stable.
// It returns a copy of r.
func (r Route) Normalize() (Route, error) {
	out := r

	out.Dst = strings.TrimSpace(out.Dst)
	out.Gateway = strings.TrimSpace(out.Gateway)
	out.Device = strings.TrimSpace(out.Device)
	out.Src = strings.TrimSpace(out.Src)
	out.Scope = strings.TrimSpace(out.Scope)
	out.Type = strings.TrimSpace(out.Type)
	out.Proto = strings.TrimSpace(out.Proto)

	out.Dst = strings.ToLower(out.Dst)
	if out.Dst == "" {
		return Route{}, fmt.Errorf("route.dst is required")
	}
	if out.Dst != "default" {
		_, ipNet, err := net.ParseCIDR(out.Dst)
		if err != nil {
			return Route{}, fmt.Errorf("invalid route.dst %q: %w", out.Dst, err)
		}
		out.Dst = ipNet.String()
	}

	if out.Gateway != "" {
		ip := net.ParseIP(out.Gateway)
		if ip == nil {
			return Route{}, fmt.Errorf("invalid route.gateway %q", out.Gateway)
		}
		out.Gateway = ip.String()
	}

	if out.Src != "" {
		ip := net.ParseIP(out.Src)
		if ip == nil {
			return Route{}, fmt.Errorf("invalid route.src %q", out.Src)
		}
		out.Src = ip.String()
	}

	out.Scope = strings.ToLower(out.Scope)
	out.Type = strings.ToLower(out.Type)
	out.Proto = strings.ToLower(out.Proto)

	if out.Table < 0 {
		return Route{}, fmt.Errorf("route.table must be >= 0")
	}
	if out.Metric < 0 {
		return Route{}, fmt.Errorf("route.metric must be >= 0")
	}

	return out, nil
}

// Key returns a deterministic identity string for a route.
// Two routes with the same key are considered the same entry for diffing purposes.
//
// This is intentionally a "full key" (set semantics):
// it supports multiple routes to the same destination (e.g. different gateways/metrics)
// by treating them as distinct entries.
func (r Route) Key() (string, error) {
	n, err := r.Normalize()
	if err != nil {
		return "", err
	}

	// Keep format stable and explicit.
	return fmt.Sprintf(
		"dst=%s|gw=%s|dev=%s|table=%d|metric=%d|src=%s|scope=%s|type=%s|proto=%s",
		n.Dst, n.Gateway, n.Device, n.Table, n.Metric, n.Src, n.Scope, n.Type, n.Proto,
	), nil
}
