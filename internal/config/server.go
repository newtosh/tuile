package config

import "time"

// Server holds HTTP serve configuration.
type Server struct {
	Listen          string
	AllowedOrigins  []string
	TLSCert         string
	TLSKey          string
	AllowRoot       bool
	TokenTTL        time.Duration
	BootstrapSecret string
}

// DefaultServer returns loopback-only defaults (R9 local transport).
func DefaultServer() Server {
	return Server{
		Listen:         "127.0.0.1:7710",
		AllowedOrigins: nil, // default-deny for browser WS (R10)
		TokenTTL:       24 * time.Hour,
	}
}
