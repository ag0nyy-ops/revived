package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
)

// writeText prints an ab-style human readable report.
func writeText(w io.Writer, c *Config, s Summary, interrupted bool) {
	u, _ := url.Parse(c.URL)
	port := u.Port()
	if port == "" {
		port = map[string]string{"http": "80", "https": "443"}[u.Scheme]
	}
	if interrupted {
		fmt.Fprintln(w, "\n(interrupted: partial results)")
	}
	fmt.Fprintf(w, "\nServer Hostname:   %s\n", u.Hostname())
	fmt.Fprintf(w, "Server Port:       %s\n", port)
	fmt.Fprintf(w, "Document Path:     %s\n\n", u.RequestURI())
	fmt.Fprintf(w, "Concurrency Level: %d\n", c.Concurrency)
	fmt.Fprintf(w, "Time taken:        %.3f seconds\n", s.Elapsed)
	fmt.Fprintf(w, "Complete requests: %d\n", s.Completed)
	fmt.Fprintf(w, "Failed requests:   %d\n", s.Failed)
	for _, k := range sortedKeys(s.Failures) {
		fmt.Fprintf(w, "   %-9s %d\n", k+":", s.Failures[k])
	}
	if len(s.StatusCodes) > 0 {
		codes := make([]int, 0, len(s.StatusCodes))
		for code := range s.StatusCodes {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		fmt.Fprint(w, "Status codes:     ")
		for _, code := range codes {
			fmt.Fprintf(w, " %d=%d", code, s.StatusCodes[code])
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "Bytes received:    %d\n", s.Bytes)
	fmt.Fprintf(w, "Requests/sec:      %.2f\n", s.RPS)
	if total := s.Completed + s.Failed; total > 0 {
		fmt.Fprintf(w, "Time per request:  %.3f ms (mean, per connection)\n", s.Latency.Mean)
		fmt.Fprintf(w, "Time per request:  %.3f ms (mean, across all concurrent requests)\n", s.Elapsed*1000/float64(total))
	}
	fmt.Fprintf(w, "Transfer rate:     %.2f MB/sec\n\n", s.MBPerSec)
	fmt.Fprintln(w, "Latency (ms):      min     mean   +/-sd   median      max")
	l := s.Latency
	fmt.Fprintf(w, "                %7.3f  %7.3f  %6.3f  %7.3f  %7.3f\n\n", l.Min, l.Mean, l.Stddev, l.Median, l.Max)
	if len(s.Percentiles) > 0 {
		fmt.Fprintln(w, "Percentage of requests served within a certain time (ms)")
		for _, p := range PercentileSteps {
			note := ""
			if p == 100 {
				note = " (longest request)"
			}
			fmt.Fprintf(w, " %4d%%  %.3f%s\n", p, s.Percentiles[p], note)
		}
	}
}

// writeJSON prints the summary as indented JSON.
func writeJSON(w io.Writer, s Summary, interrupted bool) error {
	out := struct {
		Summary
		Interrupted bool `json:"interrupted"`
	}{s, interrupted}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
