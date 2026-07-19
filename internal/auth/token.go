package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Scope names credential capabilities (KTD1).
type Scope string

const (
	ScopeAgentRead    Scope = "agent:read"
	ScopeAgentWrite   Scope = "agent:write"
	ScopeHumanView    Scope = "human:view"
	ScopeHumanControl Scope = "human:control"
)

// AgentScopes are minted for automated clients.
var AgentScopes = []Scope{ScopeAgentRead, ScopeAgentWrite}

// HumanScopes include observe and takeover.
var HumanScopes = []Scope{ScopeHumanView, ScopeHumanControl}

// SessionScopes grants full access to a session (returned at create time).
var SessionScopes = []Scope{
	ScopeAgentRead, ScopeAgentWrite, ScopeHumanView, ScopeHumanControl,
}

// Claims binds a bearer token to a session.
type Claims struct {
	SessionID string
	Scopes    []Scope
	ExpiresAt time.Time
}

// Store holds active bearer tokens in memory.
type Store struct {
	mu      sync.RWMutex
	byToken map[string]Claims
	bySess  map[string][]string
}

// NewStore creates an empty token store.
func NewStore() *Store {
	return &Store{
		byToken: make(map[string]Claims),
		bySess:  make(map[string][]string),
	}
}

// Mint issues a new bearer token for sessionID.
func (s *Store) Mint(sessionID string, scopes []Scope, ttl time.Duration) (string, error) {
	if sessionID == "" {
		return "", errors.New("session id required")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("mint token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	claims := Claims{
		SessionID: sessionID,
		Scopes:    append([]Scope(nil), scopes...),
		ExpiresAt: time.Now().Add(ttl),
	}

	s.mu.Lock()
	s.byToken[token] = claims
	s.bySess[sessionID] = append(s.bySess[sessionID], token)
	s.mu.Unlock()

	return token, nil
}

// Validate checks a bearer token and returns claims.
func (s *Store) Validate(token string) (Claims, error) {
	if token == "" {
		return Claims{}, ErrUnauthorized
	}

	s.mu.RLock()
	claims, ok := s.byToken[token]
	s.mu.RUnlock()

	if !ok {
		return Claims{}, ErrUnauthorized
	}
	if time.Now().After(claims.ExpiresAt) {
		s.Revoke(token)
		return Claims{}, ErrUnauthorized
	}
	return claims, nil
}

// HasScope reports whether claims include scope.
func HasScope(claims Claims, want Scope) bool {
	for _, sc := range claims.Scopes {
		if sc == want {
			return true
		}
	}
	return false
}

// Revoke removes a single token.
func (s *Store) Revoke(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	claims, ok := s.byToken[token]
	if !ok {
		return
	}
	delete(s.byToken, token)
	tokens := s.bySess[claims.SessionID]
	for i, t := range tokens {
		if t == token {
			s.bySess[claims.SessionID] = append(tokens[:i], tokens[i+1:]...)
			break
		}
	}
}

// RevokeSession removes all tokens for a session.
func (s *Store) RevokeSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, token := range s.bySess[sessionID] {
		delete(s.byToken, token)
	}
	delete(s.bySess, sessionID)
}

// BootstrapSecret is a server-wide secret for session creation.
type BootstrapSecret string

// NewBootstrapSecret generates a random bootstrap secret.
func NewBootstrapSecret() (BootstrapSecret, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return BootstrapSecret(base64.RawURLEncoding.EncodeToString(raw)), nil
}

// ConstantTimeEqual compares bootstrap secrets safely.
func (b BootstrapSecret) ConstantTimeEqual(other string) bool {
	return subtle.ConstantTimeCompare([]byte(b), []byte(other)) == 1
}

// ErrUnauthorized is returned for invalid or expired credentials.
var ErrUnauthorized = errors.New("unauthorized")

// ErrForbidden is returned when credentials lack required scope.
var ErrForbidden = errors.New("forbidden")
