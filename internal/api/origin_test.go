package api

import "testing"

func TestOriginAllowedDefaultDeny(t *testing.T) {
	if OriginAllowed("https://evil.example", nil) {
		t.Fatal("expected default deny with nil allowlist")
	}
	if OriginAllowed("", []string{"https://app.example"}) {
		t.Fatal("expected empty origin rejected")
	}
}

func TestOriginAllowedMatch(t *testing.T) {
	list := []string{"https://app.example", "http://localhost:3000"}
	if !OriginAllowed("https://app.example", list) {
		t.Fatal("expected allowed origin")
	}
	if OriginAllowed("https://other.example", list) {
		t.Fatal("expected disallowed origin")
	}
}
