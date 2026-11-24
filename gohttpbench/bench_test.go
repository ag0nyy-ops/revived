package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func cfg(url string, mod func(*Config)) *Config {
	c := &Config{URL: url, Method: "GET", Requests: 20, Concurrency: 4, Timeout: 5 * time.Second, ContentType: "text/plain"}
	if mod != nil {
		mod(c)
	}
	return c
}

func TestRunCountsRequests(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		io.WriteString(w, "hello")
	}))
	defer srv.Close()
	res, _ := Run(context.Background(), cfg(srv.URL, nil), nil)
	s := Summarize(res, time.Second)
	if hits.Load() != 20 || s.Completed != 20 || s.Failed != 0 {
		t.Fatalf("hits=%d summary=%+v", hits.Load(), s)
	}
	if s.Bytes != 100 {
		t.Fatalf("bytes=%d", s.Bytes)
	}
}

func TestRunSendsHeadersBodyAuthAndCookies(t *testing.T) {
	var got atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		u, p, _ := r.BasicAuth()
		got.Store(strings.Join([]string{r.Method, string(b), r.Header.Get("Content-Type"), r.Header.Get("X-A"), r.Header.Get("Cookie"), u + ":" + p, r.Host}, "|"))
	}))
	defer srv.Close()
	c := cfg(srv.URL, func(c *Config) {
		c.Requests, c.Concurrency = 1, 1
		c.Method, c.Body, c.ContentType = "POST", []byte("data"), "application/json"
		c.Headers = []string{"X-A: 1", "Host: example.test"}
		c.Cookies = []string{"a=1", "b=2"}
		c.Auth = "bob:pw"
	})
	Run(context.Background(), c, nil)
	want := "POST|data|application/json|1|a=1; b=2|bob:pw|example.test"
	if got.Load() != want {
		t.Fatalf("got %v\nwant %s", got.Load(), want)
	}
}

func TestRunCategorisesStatusFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()
	res, _ := Run(context.Background(), cfg(srv.URL, nil), nil)
	s := Summarize(res, time.Second)
	if s.Failures[FailStatus] != 20 || s.StatusCodes[503] != 20 {
		t.Fatalf("%+v", s)
	}
}

func TestRunDetectsLengthMismatch(t *testing.T) {
	var n atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, strings.Repeat("x", int(n.Add(1))))
	}))
	defer srv.Close()
	res, _ := Run(context.Background(), cfg(srv.URL, func(c *Config) { c.Requests, c.Concurrency = 5, 1 }), nil)
	if Summarize(res, time.Second).Failures[FailLength] != 4 {
		t.Fatalf("%+v", res)
	}
}

func TestConnectErrorStopsUnlessContinue(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	c := cfg(url, func(c *Config) { c.Requests, c.Concurrency = 50, 1 })
	res, _ := Run(context.Background(), c, nil)
	if len(res) != 1 || res[0].Fail != FailConnect {
		t.Fatalf("expected a single connect failure, got %d results", len(res))
	}
	c.ContinueOnError = true
	res, _ = Run(context.Background(), c, nil)
	if len(res) != 50 {
		t.Fatalf("got %d results with -r", len(res))
	}
}

func TestDurationMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	c := cfg(srv.URL, func(c *Config) { c.Duration = 200 * time.Millisecond; c.KeepAlive = true })
	res, el := Run(context.Background(), c, nil)
	if len(res) == 0 || el < 150*time.Millisecond || el > 2*time.Second {
		t.Fatalf("results=%d elapsed=%v", len(res), el)
	}
	if f := Summarize(res, el).Failed; f != 0 {
		t.Fatalf("deadline cut-off counted as %d failures", f)
	}
}

func TestInterruptReturnsPartialResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	c := cfg(srv.URL, func(c *Config) { c.Requests = 1_000_000; c.Concurrency = 2 })
	res, el := Run(ctx, c, nil)
	if len(res) == 0 || len(res) >= 1_000_000 || el > 2*time.Second {
		t.Fatalf("results=%d elapsed=%v", len(res), el)
	}
}

func TestCLIReportsAndExitCodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") }))
	defer srv.Close()
	var out, errb bytes.Buffer
	if code := run([]string{"-q", "-n", "10", "-c", "2", srv.URL + "/p"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	for _, want := range []string{"Complete requests: 10", "Document Path:     /p", "50%"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("report missing %q:\n%s", want, out.String())
		}
	}
	out.Reset()
	if code := run([]string{"--help"}, &out, &errb); code != 0 {
		t.Fatalf("help exit %d", code)
	}
	if code := run([]string{"nonsense"}, &out, &errb); code != 2 {
		t.Fatalf("bad url exit %d", code)
	}
}

func TestCLIJSONAndFailureExit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer srv.Close()
	var out, errb bytes.Buffer
	code := run([]string{"--json", "-n", "4", srv.URL}, &out, &errb)
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	var v struct {
		Failed      int
		Failures    map[string]int
		Interrupted bool
	}
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatalf("bad json: %v\n%s", err, out.String())
	}
	if v.Failed != 4 || v.Failures["status"] != 4 || v.Interrupted {
		t.Fatalf("%+v", v)
	}
	if errb.Len() != 0 {
		t.Fatalf("json mode wrote to stderr: %s", errb.String())
	}
}
