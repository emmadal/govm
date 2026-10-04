package internal

import "runtime/debug"

// Version is set at build time with
// -ldflags "-X github.com/emmadal/govm/internal.Version=v1.2.3".
var Version = ""

// GetVersion returns the version of govm: the build-time version when set,
// the module version for `go install` builds, and "dev" otherwise.
func GetVersion() string {
	if Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
