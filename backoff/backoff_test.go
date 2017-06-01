package backoff

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type LoggerStrategy struct {
	Strategy Strategy
	Log      func(retry int, interval time.Duration, ok bool)
}

func (s *LoggerStrategy) Retry(retry int) (interval time.Duration, ok bool) {
	interval, ok = s.Strategy.Retry(retry)
	if s.Log != nil {
		s.Log(retry, interval, ok)
	}
	return
}

var exponentialBackoffIntervals = []time.Duration{
	500000000,   // 0 (should not happen)
	500000000,   // 1
	750000000,   // 2
	1125000000,  // 3
	1687500000,  // 4
	2531250000,  // 5
	3796875000,  // 6
	5695312500,  // 7
	8542968750,  // 8
	12814453125, // 9
	19221679687, // 10
	28832519530, // 11 (should fail for ErrRetriesExhausted here)
}

func TestExponentialBackoffIntervals(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	config := &Config{
		ResponseChecker: DefaultResponseChecker,
		Strategy: &LoggerStrategy{
			Strategy: DefaulExponentialStrategy,
			Log: func(retry int, interval time.Duration, ok bool) {
				t.Logf("Retry %2d: interval=%11d, ok=%t", retry, interval, ok)

				if retry < 1 {
					t.Fatal("retry cannot be less than 1")
				}

				if retry >= len(exponentialBackoffIntervals) {
					if ok {
						t.Fatal("expected ok == false")
					}
				} else {
					if exponentialBackoffIntervals[retry] != interval {
						t.Fatalf("expected interval %d, but got %d", exponentialBackoffIntervals[retry], interval)
					}
				}
			},
		},
	}
	config.SkipSleep(true)
	_, err := config.Backoff(func() (*http.Response, error) {
		return http.Get(ts.URL)
	})
	t.Log(err)
	if err != ErrRetriesExhausted {
		t.Fatal("expected ErrRetriesExhausted")
	}
}

func TestExponentialBackoffSuccess(t *testing.T) {
	const SuccessCount = 3
	i := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if i >= SuccessCount {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		i += 1
	}))
	defer ts.Close()

	DefaultConfig.SkipSleep(true)
	resp, err := Backoff(func() (*http.Response, error) {
		return http.Get(ts.URL)
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode == http.StatusOK {
		t.Logf("received %s", resp.Status)
	} else {
		t.Fatalf("expected interval %d, but got %d", http.StatusText(http.StatusOK), resp.Status)
	}
}
