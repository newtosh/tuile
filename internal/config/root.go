package config

import (
	"fmt"
	"os"
)

// RefuseRootIfNeeded enforces R11 unless AllowRoot is set.
func RefuseRootIfNeeded(allowRoot bool) error {
	if allowRoot {
		return nil
	}
	if os.Geteuid() == 0 {
		return fmt.Errorf("refusing to run as root (R11); use a non-root user or --allow-root for development")
	}
	return nil
}
