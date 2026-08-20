package version

import "regexp"

// Version is the product version stamped into release binaries. A plain go
// build or go test leaves it as "dev".
var Version = "dev"

var release = regexp.MustCompile(`^0\.\d+\.\d+$`)

// IsRelease reports whether Version is a 0.x.y product version.
func IsRelease() bool {
	return release.MatchString(Version)
}
