package api

import "strings"

// OriginAllowed implements default-deny Origin allowlist matching (R10).
func OriginAllowed(origin string, allowlist []string) bool {
	if len(allowlist) == 0 {
		return false
	}
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return false
	}
	for _, allowed := range allowlist {
		if strings.EqualFold(strings.TrimSpace(allowed), origin) {
			return true
		}
	}
	return false
}
