package main

import (
	"os"
	"testing"
)

func TestSmoketestAllowedHosts_Default(t *testing.T) {
	t.Setenv("SMOKETEST_ALLOW_HOSTS", "")
	got := smoketestAllowedHosts()
	if len(got) != 3 {
		t.Fatalf("default: got %v", got)
	}
}

func TestSmoketestAllowedHosts_Star(t *testing.T) {
	t.Setenv("SMOKETEST_ALLOW_HOSTS", "*")
	if smoketestAllowedHosts() != nil {
		t.Fatal("expected nil for *")
	}
}

func TestSmoketestAllowedHosts_Custom(t *testing.T) {
	t.Setenv("SMOKETEST_ALLOW_HOSTS", " Staging.EXAMPLE.com , api.ci ")
	got := smoketestAllowedHosts()
	if len(got) != 2 || got[0] != "staging.example.com" || got[1] != "api.ci" {
		t.Fatalf("got %v", got)
	}
}

func TestHostMatchesAllowlist(t *testing.T) {
	allowed := []string{"localhost", "127.0.0.1", "::1"}
	if !hostMatchesAllowlist("localhost", allowed) {
		t.Fatal("localhost")
	}
	if !hostMatchesAllowlist("LOCALHOST", allowed) {
		t.Fatal("case")
	}
	if !hostMatchesAllowlist("::1", allowed) {
		t.Fatal("::1")
	}
	if hostMatchesAllowlist("prod.example.com", allowed) {
		t.Fatal("should reject")
	}
}

func TestMain(m *testing.M) {
	// Tests in this package must not inherit SMOKETEST_ALLOW_HOSTS from the outer environment.
	_ = os.Unsetenv("SMOKETEST_ALLOW_HOSTS")
	os.Exit(m.Run())
}
