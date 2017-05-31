package normurl

import (
	"strings"
)

func (u *URL) Hostname() string {
	i := strings.Index(u.gu.Host, ":")
	if i == -1 {
		return u.gu.Host
	} else {
		return u.gu.Host[0:i]
	}
}

func (u *URL) MarshalBinary() (text []byte, err error) {
	return []byte(u.String()), nil
}

func (u *URL) Port() string {
	i := strings.Index(u.gu.Host, ":")
	if i == -1 {
		return ""
	} else {
		return u.gu.Host[i+1:]
	}
}

func (u *URL) UnmarshalBinary(text []byte) error {
	u1, err := Parse(string(text))
	if err != nil {
		return err
	}
	*u = *u1
	return nil
}
