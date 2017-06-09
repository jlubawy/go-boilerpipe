package backoff

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

type httpError struct {
	resp *http.Response
}

func (err httpError) Error() string {
	return fmt.Sprintf("received error response %s", err.resp.Status)
}

func IsResponseError(err error) bool {
	_, ok := err.(httpError)
	return ok
}

var ErrRetriesExhausted = errors.New("retries exhausted")

type Config struct {
	ResponseChecker ResponseChecker
	Strategy        Strategy

	skipSleep bool
}

var DefaultConfig = &Config{
	ResponseChecker: DefaultResponseChecker,
	Strategy:        DefaulExponentialStrategy,
}

type NextFunc func() (*http.Response, error)

// Backoff attempts to complete a given HTTP request using the NextFunc closure
// until successful, a permanent error is received, or the retries specified by c.Strategy
// have been exhausted. A valid *http.Response will be returned if and only if err is nil,
// IsResponseError, or ErrRetriesExhausted.
func (c *Config) Backoff(next NextFunc) (*http.Response, error) {
	for retry := 1; ; retry++ {
		// Get the next response/error
		resp, err := next()
		if err != nil {
			return nil, err
		}

		// Check if the response is an error
		status := c.ResponseChecker.Check(resp)
		switch status {
		case StatusOK:
			return resp, nil

		case StatusPermanent:
			return resp, httpError{resp}

		case StatusTemporary:
			interval, ok := c.Strategy.Retry(retry)
			if !ok {
				return resp, ErrRetriesExhausted // retries exhausted
			}

			// Sleep for the returned interval if any
			if !c.skipSleep && interval >= time.Duration(0) {
				time.Sleep(interval)
			}
		}
	}
}

func (c *Config) SkipSleep(skipSleep bool) {
	c.skipSleep = skipSleep
}

func Backoff(next NextFunc) (*http.Response, error) {
	return DefaultConfig.Backoff(next)
}

type Status int

const (
	StatusOK Status = iota
	StatusPermanent
	StatusTemporary
)

type ResponseChecker interface {
	Check(resp *http.Response) Status
}

var DefaultResponseChecker = responseChecker{}

type responseChecker struct{}

func (responseChecker) Check(resp *http.Response) Status {
	if resp.StatusCode >= 400 {
		if resp.StatusCode == http.StatusServiceUnavailable {
			return StatusTemporary
		}
		return StatusPermanent
	}
	return StatusOK
}

type Strategy interface {
	Retry(retry int) (interval time.Duration, ok bool)
}

const (
	DefaultExponentialInterval    time.Duration = 500 * time.Millisecond
	DefaultExponentialMaxInterval time.Duration = 20 * time.Second
	DefaultExponentialMultiplier  float64       = 1.5
)

type ExponentialStrategy struct {
	Interval    time.Duration
	MaxInterval time.Duration
	Multiplier  float64

	currentInterval time.Duration
}

var DefaulExponentialStrategy = &ExponentialStrategy{
	Interval:    DefaultExponentialInterval,
	MaxInterval: DefaultExponentialMaxInterval,
	Multiplier:  DefaultExponentialMultiplier,
}

func (s *ExponentialStrategy) Retry(retry int) (interval time.Duration, ok bool) {
	if retry == 1 {
		interval = s.Interval
	} else {
		interval = time.Duration(s.Multiplier * float64(s.currentInterval))
	}

	s.currentInterval = interval
	ok = interval <= s.MaxInterval
	return
}
