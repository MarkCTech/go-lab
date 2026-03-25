package main

import (
	"fmt"
	"os"
	"strings"
)

// SMOKETEST_ALLOW_HOSTS controls which request hosts -base may use. Comma-separated hostnames or IPs
// (no port), compared case-insensitively for names. If unset, only loopback is allowed: localhost,
// 127.0.0.1, ::1. Set to * to disable the check (not recommended). This binary mutates the API database.
func smoketestAllowedHosts() []string {
	raw := strings.TrimSpace(os.Getenv("SMOKETEST_ALLOW_HOSTS"))
	if raw == "" {
		return []string{"localhost", "127.0.0.1", "::1"}
	}
	if raw == "*" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, strings.ToLower(p))
		}
	}
	if len(out) == 0 {
		return []string{"localhost", "127.0.0.1", "::1"}
	}
	return out
}

func hostMatchesAllowlist(host string, allowed []string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if i := strings.IndexByte(h, '%'); i >= 0 {
		h = h[:i]
	}
	for _, a := range allowed {
		if h == a {
			return true
		}
	}
	return false
}

func mustHostAllowed(host string) {
	host = strings.TrimSpace(host)
	if host == "" {
		fmt.Fprintf(os.Stderr, "smoketest: -base URL has no host\n")
		os.Exit(1)
	}
	allowed := smoketestAllowedHosts()
	if allowed == nil {
		fmt.Fprintf(os.Stderr, "smoketest: warning: SMOKETEST_ALLOW_HOSTS=* (host allowlist disabled)\n")
		return
	}
	if hostMatchesAllowlist(host, allowed) {
		return
	}
	fmt.Fprintf(os.Stderr, "smoketest: host %q is not allowed. Allowed: %s\n", host, strings.Join(allowed, ", "))
	fmt.Fprintf(os.Stderr, "smoketest: this command mutates database state; use only against disposable APIs.\n")
	fmt.Fprintf(os.Stderr, "smoketest: set SMOKETEST_ALLOW_HOSTS to a comma-separated list including this host, or '*' to disable (not recommended).\n")
	os.Exit(1)
}
