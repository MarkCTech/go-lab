package main

import "testing"

func TestIsUnversionedLegacyAPIPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/healthz", false},
		{"/api/v1/users", false},
		{"/api/v1/auth/token", false},
		{"/api/v1/auth/login", false},
		{"/api/v1/auth/register", false},
		{"/api/v1/auth/csrf", false},
		{"/api", true},
		{"/api/users", true},
		{"/api/foo/bar", true},
	}
	for _, tt := range tests {
		if got := isUnversionedLegacyAPIPath(tt.path); got != tt.want {
			t.Errorf("%q: got %v want %v", tt.path, got, tt.want)
		}
	}
}
