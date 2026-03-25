package auth

import (
	"testing"
)

func TestHashPasswordVerifyRoundTrip(t *testing.T) {
	const pw = "correct-horse-battery-staple"
	h, err := HashPassword(pw)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword(pw, h)
	if err != nil || !ok {
		t.Fatalf("verify same password: ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword("wrong", h)
	if err != nil || ok {
		t.Fatalf("verify wrong password should fail: ok=%v err=%v", ok, err)
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	_, err := VerifyPassword("password123!", "not-a-real-hash")
	if err == nil {
		t.Fatal("expected error for malformed hash")
	}
}

func TestPasswordPolicySummary(t *testing.T) {
	s := PasswordPolicySummary()
	if s == "" {
		t.Fatal("empty summary")
	}
}
