package auth

import (
	"testing"
	"time"
)

func TestMintValidateRevoke(t *testing.T) {
	store := NewStore()
	token, err := store.Mint("sess-1", AgentScopes, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := store.Validate(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.SessionID != "sess-1" {
		t.Fatalf("session id = %q", claims.SessionID)
	}
	if !HasScope(claims, ScopeAgentRead) {
		t.Fatal("expected agent:read scope")
	}

	store.RevokeSession("sess-1")
	if _, err := store.Validate(token); err == nil {
		t.Fatal("expected revoked token to fail")
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	store := NewStore()
	token, err := store.Mint("sess-2", AgentScopes, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, err := store.Validate(token); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestBootstrapSecretConstantTime(t *testing.T) {
	sec, err := NewBootstrapSecret()
	if err != nil {
		t.Fatal(err)
	}
	if !sec.ConstantTimeEqual(string(sec)) {
		t.Fatal("expected secret to match itself")
	}
	if sec.ConstantTimeEqual("wrong") {
		t.Fatal("expected mismatch")
	}
}
