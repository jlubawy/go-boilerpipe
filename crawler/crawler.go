package crawler

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/jlubawy/go-boilerpipe/crawler/backoff"

	"golang.org/x/net/html"
)

type CrawlResult struct {
	u     *url.URL
	hrefs []string
	errs  []error
}

func (r *CrawlResult) URL() *url.URL {
	return r.u
}

func (r *CrawlResult) Hrefs() []string {
	return r.hrefs
}

func (r *CrawlResult) Errors() []error {
	return r.errs
}

const (
	DefaultMaxThreads       = 3
	DefaultThrottleDuration = 2 * time.Second
)

type Crawler struct {
	start            uint
	end              uint
	throttleDuration time.Duration

	sem chan bool
}

func NewCrawler(start, end uint) *Crawler {
	if start >= end {
		panic("start must be less than end")
	}

	return &Crawler{
		start:            start,
		end:              end,
		throttleDuration: DefaultThrottleDuration,

		sem: make(chan bool, DefaultMaxThreads),
	}
}

func (c *Crawler) Crawl(client *http.Client, root *url.URL, pageFunc func(u *url.URL, page uint) *url.URL) (results chan *CrawlResult) {
	results = make(chan *CrawlResult)

	reqLeft := c.end - c.start
	done := make(chan bool)

	go func() {
		for page := c.start; page < c.end; page++ {
			// Wait until there is a request thread available
			c.sem <- true

			// Get the URL of the next page
			copyRoot := *root
			pageURL := pageFunc(&copyRoot, page)

			// Perform each request in a separate thread
			go func() {
				hrefs, errs := CrawlPage(client, pageURL)
				results <- &CrawlResult{
					u:     pageURL,
					hrefs: hrefs,
					errs:  errs,
				}

				<-c.sem
				done <- true
			}()

			time.Sleep(c.throttleDuration)
		}

		// Wait for all threads to finish
		for range done {
			reqLeft -= 1
			if reqLeft == 0 {
				close(results)
				return
			}
		}
	}()

	return results
}

// CrawlPage retrieves a page via HTTP and extracts all anchors from the response.
func CrawlPage(client *http.Client, u *url.URL) (hrefs []string, errs []error) {
	errs = make([]error, 0)

	for {
		resp, err := backoff.NewBackoffClient(client).Get(u)
		if err != nil {
			errs = append(errs, err)
			return
		}

		if resp.Done {
			hrefs1, err := ExtractAnchors(resp.Resp.Body)
			if err != nil {
				// Fatal error
				errs = append(errs, err)
				return
			} else {
				hrefs = hrefs1
			}
			return

		} else {
			// Non-fatal error
			errs = append(errs, errors.New(resp.Resp.Status))
		}
	}

	return
}

// ExtractAnchors extracts all anchors from a given io.Reader HTML document.
func ExtractAnchors(r io.Reader) (hrefs []string, err error) {
	hrefs = make([]string, 0)
	z := html.NewTokenizer(r)

	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			err1 := z.Err()
			if err1 != io.EOF {
				err = err1
				return
			}
			return // if EOF return with no error

		case html.StartTagToken:
			tn, _ := z.TagName()
			if len(tn) == 1 && tn[0] == 'a' {
				for {
					key, val, moreAttr := z.TagAttr()
					if string(key) == "href" {
						hrefs = append(hrefs, string(val))
						break
					}

					if !moreAttr {
						break
					}
				}
			}
		}
	}

	return
}
