//go:build android

package eletrocromo

import (
	"context"
	"net"
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
				// Prefer family-matched networks; see dnsDialNetworks.
				for _, nw := range dnsDialNetworks(s) {
					c, err := d.DialContext(ctx, nw, s)
					if err == nil {
						return c, nil
					}
					last = err
				}
			}
			if last != nil {
				return nil, last
			}
			return d.DialContext(ctx, network, address)
		},
	}
}
