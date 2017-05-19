// +build go1.6

package normurl

import (
	"regexp"
	"strings"
)

const (
	posScheme   = 1
	posHostname = 2
	posPort     = 3
	posPath     = 4
	posQuery    = 7
)

// From http://stackoverflow.com/a/27755
var reURL = regexp.MustCompile(`^(.*):\/\/([A-Za-z0-9\-\.]+):?([0-9]+)?(.*)$`)

func getURLParts(u *URL) []string {
	return reURL.FindStringSubmatch(u.gu.String())
}

func (u *URL) Hostname() string {
	parts := getURLParts(u)
	if len(parts) > posHostname {
		return parts[posHostname]
	}
	return ""
}

//func (u *URL) MarshalBinary() (text []byte, err error) {
//	return u.gu.MarshalBinary()
//}

func (u *URL) Port() string {
	parts := getURLParts(u)
	if len(parts) > posPort {
		return parts[posPort]
	}
	return ""
}

func (u *URL) Path() string {
	parts := getURLParts(u)
	if len(parts) > posPath {
		path := parts[posPath]

		qi := strings.Index(path, "?")
		pi := strings.Index(path, "#")

		if qi == -1 && pi == -1 {
			return path
		} else {
			if qi == -1 {
				return path[0:pi]
			} else {
				return path[0:qi]
			}
		}
	}
	return ""
}

func (u *URL) Scheme() string {
	parts := getURLParts(u)
	if len(parts) > posScheme {
		return parts[posScheme]
	}
	return ""
}

//func (u *URL) UnmarshalBinary(text []byte) error {
//	return u.gu.UnmarshalBinary(text)
//}
