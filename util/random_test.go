package util

import (
	"regexp"
	"testing"
)

func TestRandomHex(t *testing.T) {
	if RandomHex(0) != "" {
		t.Fatal("RandomHex(0) should be empty")
	}
	if got := RandomHex(16); len(got) != 32 {
		t.Fatalf("RandomHex(16) length = %d, want 32", len(got))
	}
	if RandomHex(16) == RandomHex(16) {
		t.Fatal("RandomHex should produce distinct values")
	}
}

func TestUUIDv4(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	u := UUIDv4()
	if !re.MatchString(u) {
		t.Fatalf("UUIDv4 = %q, want RFC 4122 v4 format", u)
	}
	if UUIDv4() == UUIDv4() {
		t.Fatal("UUIDv4 should produce distinct values")
	}
}
