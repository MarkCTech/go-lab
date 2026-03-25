package platformrbac

import "testing"

func TestHasPermissionWildcard(t *testing.T) {
	if !HasPermission([]string{"operator"}, PermPlayersRead) {
		t.Fatal("operator should imply all permissions")
	}
}

func TestHasPermissionSupport(t *testing.T) {
	if !HasPermission([]string{"support"}, PermPlayersRead) {
		t.Fatal("support should read players")
	}
	if HasPermission([]string{"support"}, PermSecurityWrite) {
		t.Fatal("support must not get security.write")
	}
}

func TestHasPermissionSecurityAdmin(t *testing.T) {
	if !HasPermission([]string{"security_admin"}, PermAuditRead) {
		t.Fatal("security_admin should read audit")
	}
	if !HasPermission([]string{"security_admin"}, PermSecurityWrite) {
		t.Fatal("security_admin should write security")
	}
	if HasPermission([]string{"security_admin"}, PermSupportAck) {
		t.Fatal("security_admin should not get support ack by default")
	}
}
