package version

import "runtime/debug"

// Version is set at link time via -ldflags; otherwise resolved from module
// build info (go install module@version) or "dev" for local working-tree builds.
var Version = "dev"

func init() {
	Version = resolve(Version, moduleVersion())
}

func resolve(explicit, module string) string {
	if explicit != "" && explicit != "dev" {
		return explicit
	}
	if module != "" && module != "(devel)" {
		return module
	}
	return "dev"
}

func moduleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}
