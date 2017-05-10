package backoff

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/jlubawy/go-boilerpipe"
)

const (
	DefaultInitialInterval time.Duration = 500 * time.Millisecond
	DefaultMaxInterval     time.Duration = 20 * time.Second
	DefaultMultiplier      float64       = 1.5

	DefaultUserAgent string = "go-boilerpipe/" + boilerpipe.VERSION
)

var ErrMaxRetries = errors.New("maximum retry limit has been reached")

type BackoffClient struct {
	client  *http.Client
	retries int

	initialInterval time.Duration
	currentInterval time.Duration
	maxInterval     time.Duration

	userAgent string
}

func NewBackoffClient(client *http.Client) *BackoffClient {
	return &BackoffClient{
		client:  client,
		retries: 3,

		initialInterval: DefaultInitialInterval,
		currentInterval: DefaultInitialInterval,
		maxInterval:     DefaultMaxInterval,

		userAgent: DefaultUserAgent,
	}
}

func (c *BackoffClient) Do(req *http.Request) (*Response, error) {
	req.Header.Set("User-Agent", c.userAgent)

	if c.currentInterval > c.maxInterval {
		return nil, ErrMaxRetries
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		switch resp.StatusCode {
		case http.StatusServiceUnavailable:
			time.Sleep(time.Duration(c.currentInterval))
			c.currentInterval = time.Duration(DefaultMultiplier * float64(c.currentInterval))
			return &Response{resp, false}, nil

		default:
			return nil, errors.New(resp.Status)
		}
	}

	return &Response{resp, true}, nil
}

func (c *BackoffClient) Get(u *url.URL) (*Response, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

func (c *BackoffClient) SetUserAgent(userAgent string) {
	c.userAgent = userAgent
}

type Response struct {
	Resp *http.Response
	Done bool
}
