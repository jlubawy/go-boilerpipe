// +build go1.6

package normurl

import (
	"testing"
)

type urlParts struct {
	hostname string
	port     string
	path     string
	scheme   string
}

var testData = map[string]urlParts{
	"http://lasvegassun.com/news": {
		hostname: "lasvegassun.com",
		port:     "",
		path:     "/news",
		scheme:   "http",
	},
	"https://lasvegassun.com:8080/news": {
		hostname: "lasvegassun.com",
		port:     "8080",
		path:     "/news",
		scheme:   "https",
	},
	"https://lasvegassun.com:8080/news?test=1": {
		hostname: "lasvegassun.com",
		port:     "8080",
		path:     "/news",
		scheme:   "https",
	},
	"https://lasvegassun.com:8080/news#fragment": {
		hostname: "lasvegassun.com",
		port:     "8080",
		path:     "/news",
		scheme:   "https",
	},
	"https://lasvegassun.com:8080/news?test=1#fragment": {
		hostname: "lasvegassun.com",
		port:     "8080",
		path:     "/news",
		scheme:   "https",
	},
}

func TestParts(t *testing.T) {
	for rawurl, parts := range testData {
		t.Log(rawurl)

		u, err := Parse(rawurl)
		if err != nil {
			t.Fatal(err)
		}

		if u.Hostname() != parts.hostname {
			t.Errorf("hostname mismatch (exp=%s, act=%s)", parts.hostname, u.Hostname())
		}

		if u.Port() != parts.port {
			t.Errorf("port mismatch (exp=%s, act=%s)", parts.port, u.Port())
		}

		if u.Path() != parts.path {
			t.Errorf("path mismatch (exp=%s, act=%s)", parts.path, u.Path())
		}

		if u.Scheme() != parts.scheme {
			t.Errorf("scheme mismatch (exp=%s, act=%s)", parts.scheme, u.Scheme())
		}
	}
}
