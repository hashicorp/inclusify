package version

import (
	version "github.com/hashicorp/go-version"
)

// Current version of inclusify.
var Version = "0.1.2"

// SemVer is an instance of version.Version. This has the secondary
// benefit of verifying during tests and init time that our version is a
// proper semantic version, which should always be the case.
var SemVer *version.Version

func init() {
	SemVer = version.Must(version.NewVersion(Version))
}

// String returns the complete version string
func String() string {
	return SemVer.String()
}
