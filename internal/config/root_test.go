package config

import (
	"os"
	"testing"
)

func TestRefuseRootWhenEUIDZero(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("not running as root")
	}
	if err := RefuseRootIfNeeded(false); err == nil {
		t.Fatal("expected root refusal")
	}
	if err := RefuseRootIfNeeded(true); err != nil {
		t.Fatalf("allow-root: %v", err)
	}
}
