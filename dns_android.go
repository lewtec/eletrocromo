//go:build android

package eletrocromo

import (
	"context"
	"net"
	"os"
	"strings"
	"time"
)

// configureDNSForPlatform sets a PreferGo resolver that does not depend on
// Android netd listening on [::1]:53 (pure-Go UDP to that address is refused).
// DNS servers come from ELETROCROMO_DNS (comma-separated) or public fallbacks.
func configureDNSForPlatform() {
	servers := dnsServersFromEnv()
	if len(servers) == 0 {
		servers = []string{"8.8.8.8:53", "1.1.1.1:53", "9.9.9.9:53"}
	}
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			var last error
			for _, s := range servers {
				// Prefer udp4 to avoid IPv6 localhost stub issues.
				c, err := d.DialContext(ctx, "udp4", s)
				if err == nil {
					return c, nil
				}
				last = err
				c, err = d.DialContext(ctx, "tcp4", s)
				if err == nil {
					return c, nil
				}
				last = err
			}
			if last != nil {
				return nil, last
			}
			return d.DialContext(ctx, network, address)
		},
	}
}

func dnsServersFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("ELETROCROMO_DNS"))
	if raw == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.Contains(p, ":") {
			p = p + ":53"
		}
		// Skip Android stub resolvers that pure Go cannot use.
		if strings.HasPrefix(p, "[::1]") || strings.HasPrefix(p, "127.0.0.1") || strings.HasPrefix(p, "::1") {
			continue
		}
		out = append(out, p)
	}
	return out
}
