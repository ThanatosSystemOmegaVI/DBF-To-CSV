package version

import "testing"

func TestStringReturnsDefaultVersion(t *testing.T) {
	if got := String(); got != "dev" {
		t.Fatalf("String() = %q, want %q", got, "dev")
	}
}

func TestStringUsesConfiguredVersion(t *testing.T) {
	oldVersion := version
	version = "1.2.3"
	defer func() { version = oldVersion }()

	if got := String(); got != "1.2.3" {
		t.Fatalf("String() = %q, want %q", got, "1.2.3")
	}
}
