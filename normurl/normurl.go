package normurl

import (
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type NormalizeOptions struct {
	KeepFragments bool
}

var defaultNormalizeOptions = NormalizeOptions{
	KeepFragments: false,
}

func Normalize(u *url.URL) *URL {
	return NormalizeWithOptions(u, &defaultNormalizeOptions)
}

func NormalizeWithOptions(u *url.URL, options *NormalizeOptions) *URL {
	// Remove blacklisted query keys
	values := u.Query()
	for _, key := range DefaultQueryKeyBlacklist.Keys() {
		values.Del(key)
	}
	u.RawQuery = values.Encode()

	if !options.KeepFragments {
		// Remove any fragments
		u.Fragment = ""
	}

	// Clean the path
	u.Path = path.Clean(u.Path)

	return NewURL(u)
}

type QueryKeyBlacklist struct {
	m map[string]bool
}

func NewQueryKeyBlacklist(keys []string) *QueryKeyBlacklist {
	bl := &QueryKeyBlacklist{
		m: make(map[string]bool),
	}
	for _, key := range keys {
		bl.m[key] = true
	}
	return bl
}

func (bl *QueryKeyBlacklist) Add(key string) *QueryKeyBlacklist {
	bl.m[key] = true
	return bl
}

func (bl *QueryKeyBlacklist) Del(key string) *QueryKeyBlacklist {
	delete(bl.m, key)
	return bl
}

func (bl *QueryKeyBlacklist) Keys() []string {
	keys := make([]string, 0, len(bl.m))
	for key := range bl.m {
		keys = append(keys, key)
	}
	return keys
}

var DefaultQueryKeyBlacklist = NewQueryKeyBlacklist([]string{
	"email_subscriber",
	"utm_campaign",
	"utm_medium",
	"utm_source",
})

type URL struct {
	gu *url.URL
}

func NewURL(u *url.URL) *URL {
	return &URL{
		gu: u,
	}
}

func Parse(rawurl string) (*URL, error) {
	gu, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	return Normalize(gu), nil
}

func ParseRequestURI(rawurl string) (*URL, error) {
	gu, err := url.ParseRequestURI(rawurl)
	if err != nil {
		return nil, err
	}
	return Normalize(gu), nil
}

func (u *URL) EscapedPath() string {
	return u.gu.EscapedPath()
}

func (u1 *URL) Equal(u2 *URL) bool {
	return u1.String() == u2.String()
}

func (u *URL) IsAbs() bool {
	return u.gu.IsAbs()
}

func (u *URL) Parse(ref string) (*URL, error) {
	gu, err := u.gu.Parse(ref)
	if err != nil {
		return nil, err
	}
	return Normalize(gu), nil
}

func (u *URL) Query() url.Values {
	return u.gu.Query()
}

func (u *URL) RequestURI() string {
	return u.gu.RequestURI()
}

func (u *URL) ResolveReference(ref *URL) *URL {
	return Normalize(u.gu.ResolveReference(ref.gu))
}

func (u *URL) String() string {
	return strings.ToLower(u.gu.String())
}

func (u *URL) ValueSet(key, value string) *URL {
	values := u.gu.Query()
	values.Set(key, value)
	u.gu.RawQuery = values.Encode()
	return u
}

type ParseDateFunc func(u *URL) (t time.Time, exists bool)

type dateRegexp struct {
	re *regexp.Regexp
	i  int
}

var monthStrings = map[string]time.Month{
	"jan": time.January,
	"feb": time.February,
	"mar": time.March,
	"apr": time.April,
	"may": time.May,
	"jun": time.June,
	"jul": time.July,
	"aug": time.August,
	"sep": time.September,
	"oct": time.October,
	"nov": time.November,
	"dec": time.December,
}

func parseMonth(s string) (time.Month, bool) {
	s = strings.ToLower(s)
	m, exists := monthStrings[s]
	return m, exists
}

// Order of these regexps matters, since some grab all yyyy-mm-dd, while others only grab yyyy-mm
var dateRegexps = []dateRegexp{
	{
		re: regexp.MustCompile(`\/([0-9]{4})\/([a-zA-Z]{3})\/([0-9]{2})[\/]*`), // scheme://host/path/2016/nov/16?query#fragment
		i:  3,
	},
	{
		re: regexp.MustCompile(`\/([0-9]{4})-([0-9]{2})-([0-9]{2})`), // scheme://host/path/2016-12-15-title?query#fragment
		i:  3,
	},
	{
		re: regexp.MustCompile(`\/([0-9]{4})-([0-9]{2})[\/]*`), // scheme://host/path/2017-01?query#fragment
		i:  2,
	},
}

func (u *URL) Date() (t time.Time, exists bool) {
	s := u.String()

	for _, v := range dateRegexps {
		ss := v.re.FindStringSubmatch(s)
		if len(ss) > 1 {
			ss = ss[1:]
		}

		var year int
		var month time.Month
		day := 1

		if len(ss) == v.i {
			y, err := strconv.ParseUint(ss[0], 10, 64)
			if err != nil {
				continue
			}
			year = int(y)

			m, err := strconv.ParseUint(ss[1], 10, 64)
			if err != nil {
				m, exists := parseMonth(ss[1])
				if exists {
					month = m
				} else {
					continue
				}
			} else {
				month = time.Month(m)
			}

			if len(ss) > 2 {
				d, err := strconv.ParseUint(ss[2], 10, 64)
				if err != nil {
					continue
				}
				day = int(d)
			}

			// Date found
			t = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
			exists = true
			return
		}
	}

	// No date found
	return
}
