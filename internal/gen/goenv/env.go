// Package goenv builds an environment for a child go command.
package goenv

import "strings"

// Merge returns base plus kv, with each kv key replacing any earlier value.
func Merge(base []string, kv ...string) []string {
	drop := make(map[string]struct{}, len(kv))
	for _, p := range kv {
		k, _, _ := strings.Cut(p, "=")
		drop[k] = struct{}{}
	}
	out := make([]string, 0, len(base)+len(kv))
	for _, e := range base {
		k, _, _ := strings.Cut(e, "=")
		if _, ok := drop[k]; ok {
			continue
		}
		out = append(out, e)
	}
	return append(out, kv...)
}
