package boilerpipe

import (
	"fmt"
	"go/build"
	"runtime"
)

const Version = "0.1.3"

var FullVersion string

func init() {
	goReleaseTag := build.Default.ReleaseTags[len(build.Default.ReleaseTags)-1]
	FullVersion = fmt.Sprintf("boilerpipe %s %s/%s/%s", Version, runtime.GOARCH, runtime.GOOS, goReleaseTag)
}
