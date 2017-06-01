package crawler

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"strconv"
	"testing"

	"golang.org/x/net/context"
)

type TestExtractor struct{}

func (TestExtractor) Extract(r io.Reader) ([]string, error) {
	d, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return []string{string(d)}, nil
}

var extractTestURLs = []string{
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/31",
	"2017/may/30",
	"2017/may/30",
	"2017/may/31",
	"2009/apr/21",
	"2010/dec/22",
}

func TestCrawler(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.FormValue("page")
		n, err := strconv.ParseInt(pageStr, 10, 64)
		if err != nil {
			t.Fatal(err)
		}

		page := int(n)
		if page-1 >= len(extractTestURLs) {
			t.Logf("pg=%d, len=%d", page, len(extractTestURLs))
			w.WriteHeader(http.StatusNotFound)
		} else {
			rawurl := path.Join(r.URL.String(), extractTestURLs[page-1], pageStr)
			fmt.Fprint(w, rawurl)
		}
	}))
	defer ts.Close()

	c := NewCrawler(http.DefaultClient, 0)
	c.Extractor = TestExtractor{}
	c.Done = func(resp *http.Response) bool {
		return resp.StatusCode != http.StatusOK
	}

	results, _ := c.Crawl(context.Background(), ts.URL, 1)
	for res := range results {
		if err := res.Error; err != nil {
			t.Logf("Page %-2d: %s\n", res.Page, err)
		}

		for _, u := range res.URLs {
			t.Logf("Page %-2d: %s\n", res.Page, u)
		}
	}
}
