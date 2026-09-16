package eletrocromo

import (
	"reflect"
	"testing"
)

func TestNormalizeDNSServer(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"", "", false},
		{"   ", "", false},
		{"8.8.8.8", "8.8.8.8:53", true},
		{"8.8.8.8:53", "8.8.8.8:53", true},
		{"1.1.1.1:5353", "1.1.1.1:5353", true},
		// Bare IPv6 must get brackets + port (old code left this unusable).
		{"2001:4860:4860::8888", "[2001:4860:4860::8888]:53", true},
		{"[2001:4860:4860::8888]:53", "[2001:4860:4860::8888]:53", true},
		{"[2606:4700:4700::1111]:53", "[2606:4700:4700::1111]:53", true},
		// Loopback / Android stubs skipped.
		{"127.0.0.1", "", false},
		{"127.0.0.1:53", "", false},
		{"::1", "", false},
		{"[::1]:53", "", false},
		{"localhost", "", false},
		{"localhost:53", "", false},
	}
	for _, tt := range cases {
		got, ok := normalizeDNSServer(tt.in)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("normalizeDNSServer(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestDNSServersFromEnv(t *testing.T) {
	t.Setenv("ELETROCROMO_DNS", "8.8.8.8, 2001:4860:4860::8888, 127.0.0.1, 1.1.1.1:53")
	got := dnsServersFromEnv()
	want := []string{"8.8.8.8:53", "[2001:4860:4860::8888]:53", "1.1.1.1:53"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}

	t.Setenv("ELETROCROMO_DNS", "")
	if dnsServersFromEnv() != nil {
		t.Fatal("empty env should yield nil")
	}
}

func TestDNSDialNetworks(t *testing.T) {
	cases := []struct {
		server string
		want   []string
	}{
		{"8.8.8.8:53", []string{"udp4", "tcp4"}},
		{"[2001:4860:4860::8888]:53", []string{"udp6", "tcp6"}},
		{"dns.example:53", []string{"udp", "tcp"}},
	}
	for _, tt := range cases {
		got := dnsDialNetworks(tt.server)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("dnsDialNetworks(%q) = %v want %v", tt.server, got, tt.want)
		}
	}
}
