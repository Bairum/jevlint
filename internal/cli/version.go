package cli

import (
	"runtime/debug"
)

// version is overridden at build time with
// -ldflags "-X github.com/codegirl-007/jevlint/internal/cli.version=vX.Y.Z".
var version = "dev"

// Version returns the build version. Release builds set it with ldflags;
// source and `go install` builds fall back to the module version in build info.
func Version() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return version
}
