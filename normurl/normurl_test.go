package normurl

import (
	"testing"
	"time"
)

var parseTestData = map[string]string{
	"http://powerbrokerconfidential.com/marcus-millichap-close-5430-west-sahara-sandyplace-llc/?utm_source=CALV+News+April+13%2C+2017&utm_campaign=CALV+September+26%2C+2016&utm_medium=email": "http://powerbrokerconfidential.com/marcus-millichap-close-5430-west-sahara-sandyplace-llc",
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
