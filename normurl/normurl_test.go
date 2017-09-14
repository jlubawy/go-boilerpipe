package normurl

import (
	"testing"
	"time"
)

var parseTestData = map[string]string{
	"http://powerbrokerconfidential.com/marcus-millichap-close-5430-west-sahara-sandyplace-llc/?utm_source=CALV+News+April+13%2C+2017&utm_campaign=CALV+September+26%2C+2016&utm_medium=email": "http://powerbrokerconfidential.com/marcus-millichap-close-5430-west-sahara-sandyplace-llc",
	"http://www.google.com/path/":  "http://www.google.com/path",
	"https://www.google.com/path/": "https://www.google.com/path",
	"www.google.com/path/":         "http://www.google.com/path", // implicit scheme
	"www.google.com/":              "http://www.google.com/",     // implicit scheme, keeps slash when there is no path
}

func TestParse(t *testing.T) {
	for rawurl, exp := range parseTestData {
		act, err := Parse(rawurl)
		if err != nil {
			t.Error(err)
		}
		if act.String() != exp {
			t.Errorf("expected '%s' but got '%s'", exp, act)
		}
	}
}

var dateTestData = map[string]string{
	"scheme://host/path/2016/nov/16?query#fragment":      "2016-11-16T00:00:00+00:00",
	"scheme://host/path/2017-01?query#fragment":          "2017-01-01T00:00:00+00:00", // no day specified
	"scheme://host/path/2016-12-15-title?query#fragment": "2016-12-15T00:00:00+00:00",
}

func TestDate(t *testing.T) {
	for rawurl, dateStr := range dateTestData {
		expTime, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			t.Error(err)
		}

		u, err := Parse(rawurl)
		if err != nil {
			t.Error(err)
		}

		actTime, exists := u.Date()
		if !exists {
			t.Errorf("time should exists but doesn't (%s)", actTime)
		}

		if !expTime.Equal(actTime) {
			t.Errorf("expected time '%s' does not equal actual time '%s'", expTime, actTime)
		}
	}
}

var rootTestData = map[string]string{
	"https://vegasinc.lasvegassun.com/business/real-estate/?page=1": "lasvegassun.com",
	"https://lasvegassun.com/business/real-estate/?page=1":          "lasvegassun.com",
	"https://.lasvegassun.com/business/real-estate/?page=1":         "lasvegassun.com",
	".lasvegassun.com/business/real-estate/?page=1":                 "lasvegassun.com",
}

func TestRoot(t *testing.T) {
	for rawurl, exp := range rootTestData {
		u, err := Parse(rawurl)
		if err != nil {
			t.Error(err)
		}
		act := u.Root()
		if exp != act {
			t.Errorf("expected '%s' but got '%s'", exp, act)
		}
	}
}

var isChildTestData = []struct {
	Root       string
	Ref        string
	ExpIsChild bool
}{
	{
		"https://vegasinc.lasvegassun.com/business/real-estate/",
		"https://vegasinc.lasvegassun.com/business/real-estate/2017/sep/06/life-is-good-for-home-sellers-not-so-much-for-buye/",
		true,
	},
	{
		"https://vegasinc.lasvegassun.com/business/real-estate/",
		"https://lasvegassun.com/business/real-estate/2017/sep/06/life-is-good-for-home-sellers-not-so-much-for-buye/",
		true,
	},
}

func TestIsChild(t *testing.T) {
	for i, d := range isChildTestData {
		root, err := Parse(d.Root)
		if err != nil {
			t.Error(err)
		}

		ref, err := Parse(d.Ref)
		if err != nil {
			t.Error(err)
		}

		if IsChild(root, ref) != d.ExpIsChild {
			t.Errorf("%d was mismatch", i)
		}
	}
}
