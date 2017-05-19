// +build go1.8

package normurl

import (
	"strings"
)

func IsChild(root, ref *URL) bool {
	if root.Hostname() != ref.Hostname() {
		return false
	}

	if !strings.HasPrefix(ref.gu.Path, root.gu.Path) {
		return false
	}

	return !ref.Equal(root)
}

func (u *URL) Hostname() string {
	return u.gu.Hostname()
}

func (u *URL) MarshalBinary() (text []byte, err error) {
	return u.gu.MarshalBinary()
}

func (u *URL) Port() string {
	return u.gu.Port()
}

func (u *URL) Path() string {
	return u.gu.Path
}

func (u *URL) Scheme() string {
	return u.gu.Scheme
}

func (u *URL) UnmarshalBinary(text []byte) error {
	return u.gu.UnmarshalBinary(text)
}
