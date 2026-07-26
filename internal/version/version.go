// Package version holds build metadata injected at compile time via
// -ldflags. It contains no business logic.
package version

// Version, Commit, and BuildDate are overridden at build time, e.g.:
//
//	go build -ldflags "-X github.com/paladindigitalgh/palladium-oss/internal/version.Version=1.2.3"
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)
