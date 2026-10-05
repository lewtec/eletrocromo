package eletrocromo

import (
	"net"
	"os"
	"strings"
)

// dnsServersFromEnv parses ELETROCROMO_DNS (comma-separated host or host:port).
// Bare IPv4/IPv6/names get port 53 via net.JoinHostPort. Loopback stubs that
// pure-Go resolvers cannot use on Android are skipped.
func dnsServersFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("ELETROCROMO_DNS"))
	if raw == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		addr, ok := normalizeDNSServer(p)
		if !ok {
			continue
		}
		out = append(out, addr)
	}
	return out
}

// normalizeDNSServer returns a dialable host:port (IPv6 bracketed) and false
// when the entry is empty or a loopback/stub resolver address.
func normalizeDNSServer(p string) (string, bool) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", false
	}
	// Already host:port or [ipv6]:port.
	if _, _, err := net.SplitHostPort(p); err == nil {
		if isUnusableDNSHost(p) {
			return "", false
		}
		return p, true
	}
	// Bare host: add default DNS port (JoinHostPort brackets IPv6).
	addr := net.JoinHostPort(p, "53")
	if isUnusableDNSHost(addr) {
		return "", false
	}
	return addr, true
}

func isUnusableDNSHost(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return true
	}
	// Android netd / local stubs: pure-Go UDP to these is refused.
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

// dnsDialNetworks picks dial networks for a host:port DNS server.
// IPv4-only hosts stay on *4 (avoids Android IPv6 localhost stub issues);
// IPv6 hosts use *6; unresolved names try udp then tcp without forcing a family.
func dnsDialNetworks(server string) []string {
	host, _, err := net.SplitHostPort(server)
	if err != nil {
		return []string{"udp4", "tcp4"}
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return []string{"udp", "tcp"}
	}
	if ip.To4() != nil {
		return []string{"udp4", "tcp4"}
	}
	return []string{"udp6", "tcp6"}
}
