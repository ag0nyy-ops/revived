package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"
)

// stringList is a repeatable string flag.
type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ", ") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

// Config holds everything needed to run one benchmark.
type Config struct {
	URL             string
	Method          string
	Requests        int
	Concurrency     int
	Duration        time.Duration
	Timeout         time.Duration
	KeepAlive       bool
	Gzip            bool
	ContinueOnError bool
	Headers         []string
	Cookies         []string
	ContentType     string
	Auth            string
	Body            []byte
	JSON            bool
	Quiet           bool
}

const usageText = `usage: gohttpbench [options] http[s]://host[:port]/path

options:
`

// parseArgs turns command line arguments into a validated Config.
func parseArgs(args []string, stderr io.Writer) (*Config, error) {
	fs := flag.NewFlagSet("gohttpbench", flag.ContinueOnError)
	fs.SetOutput(stderr)
	c := &Config{}
	var headers, cookies stringList
	var head bool
	var postFile, putFile string

	fs.IntVar(&c.Requests, "n", 1, "number of requests to perform")
	fs.IntVar(&c.Concurrency, "c", 1, "number of concurrent requests")
	fs.DurationVar(&c.Duration, "d", 0, "run for this long instead of a fixed request count (e.g. 10s)")
	fs.DurationVar(&c.Timeout, "t", 30*time.Second, "per-request timeout")
	fs.BoolVar(&c.KeepAlive, "k", false, "use HTTP keep-alive")
	fs.BoolVar(&c.Gzip, "z", false, "ask for gzip responses")
	fs.BoolVar(&c.ContinueOnError, "r", false, "keep going when requests fail (default: stop after the first error burst)")
	fs.BoolVar(&head, "i", false, "use HEAD instead of GET")
	fs.Var(&headers, "H", "extra header, e.g. 'Accept: text/html' (repeatable)")
	fs.Var(&cookies, "C", "cookie, e.g. 'session=1234' (repeatable)")
	fs.StringVar(&c.ContentType, "T", "text/plain", "content type for POST/PUT bodies")
	fs.StringVar(&c.Auth, "A", "", "basic auth as user:password")
	fs.StringVar(&postFile, "p", "", "file with data to POST")
	fs.StringVar(&putFile, "u", "", "file with data to PUT")
	fs.BoolVar(&c.JSON, "json", false, "print the report as JSON")
	fs.BoolVar(&c.Quiet, "q", false, "no progress output")
	fs.Usage = func() {
		fmt.Fprint(stderr, usageText)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return nil, errors.New("exactly one URL is required")
	}
	c.URL = fs.Arg(0)
	c.Headers, c.Cookies = headers, cookies

	c.Method = "GET"
	switch {
	case postFile != "" && putFile != "":
		return nil, errors.New("-p and -u cannot be combined")
	case head && (postFile != "" || putFile != ""):
		return nil, errors.New("-i cannot be combined with -p or -u")
	case head:
		c.Method = "HEAD"
	case postFile != "":
		c.Method = "POST"
	case putFile != "":
		c.Method = "PUT"
	}
	if f := postFile + putFile; f != "" {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading body file: %w", err)
		}
		c.Body = b
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) validate() error {
	u, err := url.Parse(c.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("invalid url %q: need http:// or https:// and a host", c.URL)
	}
	if c.Concurrency < 1 {
		return errors.New("-c must be at least 1")
	}
	if c.Duration < 0 {
		return errors.New("-d cannot be negative")
	}
	if c.Duration == 0 && c.Requests < 1 {
		return errors.New("-n must be at least 1")
	}
	if c.Duration == 0 && c.Concurrency > c.Requests {
		return errors.New("-c cannot be larger than -n")
	}
	for _, h := range c.Headers {
		if !strings.Contains(h, ":") {
			return fmt.Errorf("invalid header %q: expected 'Name: value'", h)
		}
	}
	for _, ck := range c.Cookies {
		if !strings.Contains(ck, "=") {
			return fmt.Errorf("invalid cookie %q: expected name=value", ck)
		}
	}
	if c.Auth != "" && !strings.Contains(c.Auth, ":") {
		return errors.New("-A expects user:password")
	}
	return nil
}
