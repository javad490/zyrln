package tun

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"zyrln/relay/route"
	"zyrln/relay/tunnel"
)

// Config controls how outbound TCP flows are dialed from the TUN engine.
type Config struct {
	Tunnel     *tunnel.TunnelClient
	DirectOnly bool
	Timeout    time.Duration
}

func (c Config) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 45 * time.Second
}

// NeedsTunnel reports whether target should use the Apps Script tunnel path.
func NeedsTunnel(cfg Config, target string) bool {
	host := route.HostFromConnectTarget(strings.TrimSpace(target))
	switch route.ConnectRouteForHost(host) {
	case route.RouteDirectFragment, route.RouteDomesticPlain:
		return false
	default:
		return !cfg.DirectOnly && cfg.Tunnel != nil
	}
}

// DialPlainBackend dials target directly (fragment, domestic, or plain protected TCP).
func DialPlainBackend(_ context.Context, target string) (net.Conn, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("empty target")
	}
	host := route.HostFromConnectTarget(target)

	switch route.ConnectRouteForHost(host) {
	case route.RouteDirectFragment:
		conn, ok := route.DialFragment(target)
		if !ok {
			return nil, fmt.Errorf("direct dial failed for %s", target)
		}
		return conn, nil
	case route.RouteDomesticPlain:
		conn, ok := route.DialPlainDirect(target)
		if !ok {
			return nil, fmt.Errorf("domestic dial failed for %s", target)
		}
		return conn, nil
	default:
		conn, ok := route.DialPlainDirect(target)
		if !ok {
			return nil, fmt.Errorf("plain dial failed for %s", target)
		}
		return conn, nil
	}
}
