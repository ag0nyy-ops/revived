package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// newClient builds an HTTP client sized for the configured concurrency.
func newClient(c *Config) *http.Client {
	tr := &http.Transport{
		DisableKeepAlives:   !c.KeepAlive,
		DisableCompression:  !c.Gzip,
		MaxIdleConns:        c.Concurrency,
		MaxIdleConnsPerHost: c.Concurrency,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		DialContext:         (&net.Dialer{Timeout: c.Timeout}).DialContext,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   c.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func newRequest(ctx context.Context, c *Config) (*http.Request, error) {
	var body io.Reader
	if c.Body != nil {
		body = bytes.NewReader(c.Body)
	}
	req, err := http.NewRequestWithContext(ctx, c.Method, c.URL, body)
	if err != nil {
		return nil, err
	}
	if c.Body != nil {
		req.Header.Set("Content-Type", c.ContentType)
	}
	if c.Gzip {
		req.Header.Set("Accept-Encoding", "gzip")
	}
	if c.Auth != "" {
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.Auth)))
	}
	for _, h := range c.Headers {
		name, val, _ := strings.Cut(h, ":")
		if strings.EqualFold(strings.TrimSpace(name), "Host") {
			req.Host = strings.TrimSpace(val)
			continue
		}
		req.Header.Add(strings.TrimSpace(name), strings.TrimSpace(val))
	}
	if len(c.Cookies) > 0 {
		req.Header.Set("Cookie", strings.Join(c.Cookies, "; "))
	}
	return req, nil
}

// classify maps a transport error to a failure category.
func classify(err error, read bool) string {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return FailTimeout
	}
	if read {
		return FailRead
	}
	return FailConnect
}

// doOne performs a single request and records its outcome.
func doOne(ctx context.Context, client *http.Client, c *Config, expected *atomic.Int64) Result {
	req, err := newRequest(ctx, c)
	if err != nil {
		return Result{Fail: FailConnect}
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return Result{Duration: time.Since(start), Fail: classify(err, false)}
	}
	defer resp.Body.Close()
	n, err := io.Copy(io.Discard, resp.Body)
	r := Result{Duration: time.Since(start), Status: resp.StatusCode, Bytes: n}
	switch {
	case err != nil:
		r.Fail = classify(err, true)
	case resp.StatusCode >= 400 || resp.StatusCode < 200:
		r.Fail = FailStatus
	case c.Method != "HEAD":
		// The first successful body length is the reference; later
		// responses that differ are counted as length failures.
		if !expected.CompareAndSwap(-1, n) && expected.Load() != n {
			r.Fail = FailLength
		}
	}
	return r
}

// Run executes the benchmark until the request budget or duration is used up,
// or ctx is cancelled. It always returns the results gathered so far.
func Run(ctx context.Context, c *Config, progress func(done int)) ([]Result, time.Duration) {
	if c.Duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Duration)
		defer cancel()
	}
	runCtx, stop := context.WithCancel(ctx)
	defer stop()

	client := newClient(c)
	defer client.CloseIdleConnections()
	var expected atomic.Int64
	expected.Store(-1)

	var (
		mu      sync.Mutex
		results []Result
		issued  atomic.Int64
		wg      sync.WaitGroup
	)
	start := time.Now()
	for i := 0; i < c.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for runCtx.Err() == nil {
				if c.Duration == 0 && issued.Add(1) > int64(c.Requests) {
					return
				}
				r := doOne(runCtx, client, c, &expected)
				if runCtx.Err() != nil && r.Fail != FailNone && r.Status == 0 {
					return // cut off by cancellation; not a real failure
				}
				mu.Lock()
				results = append(results, r)
				n := len(results)
				mu.Unlock()
				if progress != nil {
					progress(n)
				}
				if !c.ContinueOnError && (r.Fail == FailConnect || r.Fail == FailTimeout) {
					stop()
				}
			}
		}()
	}
	wg.Wait()
	return results, time.Since(start)
}
