package crawler

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/jlubawy/go-boilerpipe/backoff"

	"golang.org/x/net/context"
	"golang.org/x/net/html"
)

const (
	DefaultMaxConcurrentPages = 3
	DefaultThrottleDuration   = 2 * time.Second
)

type Crawler struct {
	Client    *http.Client
	Extractor Extractor
	Pager     Pager

	LastPage uint
	Done     func(resp *http.Response) bool

	MaxConcurrentPages uint
	ThrottleDuration   time.Duration
}

func NewCrawler(client *http.Client, throttleDuration time.Duration) *Crawler {
	return &Crawler{
		Client: client,
		Pager:  DefaultPager,

		LastPage: math.MaxUint64,
		Done: func(resp *http.Response) bool {
			return resp.StatusCode == http.StatusNotFound
		},

		ThrottleDuration: throttleDuration,
	}
}

func (c *Crawler) Crawl(ctx context.Context, rootURL string, startPage uint) (chan *Result, context.CancelFunc) {
	if c.Extractor == nil {
		c.Extractor = htmlExtractor{}
	}
	if c.MaxConcurrentPages == 0 {
		c.MaxConcurrentPages = DefaultMaxConcurrentPages
	}

	results := make(chan *Result)
	newCtx, cancelFunc := context.WithCancel(ctx)

	var (
		sem = make(chan struct{}, c.MaxConcurrentPages)
		wg  sync.WaitGroup
	)

	go func() {
		done := make(chan struct{})
		doneFunc := func(resp *http.Response) bool {
			if c.Done(resp) {
				done <- struct{}{}
				cancelFunc()
				return true
			}
			return false
		}

		t := time.NewTimer(c.ThrottleDuration)
		for page := startPage; page <= c.LastPage; page++ {
			select {
			case <-newCtx.Done():
				goto DONE
			case <-done:
				goto DONE

			case <-t.C:
				// Throttle based on MaxConcurrentPages
				sem <- struct{}{}
				wg.Add(1)

				go func(page uint) {
					defer func() {
						wg.Done()
						<-sem
					}()

					pageURL := c.Pager.NextPage(rootURL, page)

					result := &Result{
						Page:    page,
						PageURL: pageURL,
						URLs:    make([]string, 0),
					}

					req, err := http.NewRequest(http.MethodGet, pageURL, nil)
					if err != nil {
						result.Error = err
					} else {
						req.Cancel = newCtx.Done()
						resp, err := backoff.Backoff(func() (*http.Response, error) {
							t.Reset(c.ThrottleDuration)
							return c.Client.Do(req)
						})
						if err != nil {
							if newCtx.Err() != nil {
								return
							}
							if backoff.IsResponseError(err) || err == backoff.ErrRetriesExhausted {
								if doneFunc(resp) {
									return
								}
							}

							result.Error = err
						} else {
							defer resp.Body.Close()

							if doneFunc(resp) {
								return
							}

							hrefs, err := c.Extractor.Extract(resp.Body)
							if err != nil {
								result.Error = err
							} else {
								result.URLs = append(result.URLs, hrefs...)
							}
						}
					}

					results <- result
				}(page)
			}
		}

	DONE:
		wg.Wait()
		close(results)
	}()

	return results, cancelFunc
}

type Extractor interface {
	Extract(r io.Reader) ([]string, error)
}

type htmlExtractor struct{}

// Extract extracts all anchor URLs from a given HTML document.
// Any errors parsing the document are returned.
func (htmlExtractor) Extract(r io.Reader) ([]string, error) {
	anchors := make([]string, 0)

	// Tokenize the io.Reader
	z := html.NewTokenizer(r)
	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			err := z.Err()
			if err == io.EOF {
				goto DONE // if EOF return with no error
			}
			return nil, err

		case html.StartTagToken:
			tn, _ := z.TagName()
			if len(tn) == 1 && tn[0] == 'a' {
				for {
					key, val, moreAttr := z.TagAttr()
					if string(key) == "href" {
						anchors = append(anchors, string(val))
						break
					}

					if !moreAttr {
						break
					}
				}
			}
		}
	}

DONE:
	return anchors, nil
}

type Pager interface {
	NextPage(rootURL string, page uint) string
}

type pager struct{}

var DefaultPager pager

func (pager) NextPage(rootURL string, page uint) string {
	return fmt.Sprintf("%s?page=%d", rootURL, page)
}

func NextPage(rootURL string, page uint) string {
	return DefaultPager.NextPage(rootURL, page)
}

type Result struct {
	Page    uint
	PageURL string
	URLs    []string
	Error   error
}
