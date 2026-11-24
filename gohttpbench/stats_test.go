package main

import (
	"math"
	"testing"
	"time"
)

func TestPercentileNearestRank(t *testing.T) {
	var d []time.Duration
	for i := 1; i <= 100; i++ {
		d = append(d, time.Duration(i)*time.Millisecond)
	}
	for p, want := range map[int]time.Duration{50: 50 * time.Millisecond, 99: 99 * time.Millisecond, 100: 100 * time.Millisecond} {
		if got := percentile(d, p); got != want {
			t.Errorf("p%d = %v, want %v", p, got, want)
		}
	}
	if percentile(nil, 50) != 0 {
		t.Error("empty input should give 0")
	}
}

func TestSummarizeKeepsSubMillisecondLatency(t *testing.T) {
	res := []Result{
		{Duration: 200 * time.Microsecond, Status: 200, Bytes: 10},
		{Duration: 400 * time.Microsecond, Status: 200, Bytes: 10},
	}
	s := Summarize(res, time.Second)
	if math.Abs(s.Latency.Min-0.2) > 1e-9 || math.Abs(s.Latency.Mean-0.3) > 1e-9 {
		t.Fatalf("latency = %+v", s.Latency)
	}
	if s.RPS != 2 || s.Bytes != 20 {
		t.Fatalf("rps=%v bytes=%d", s.RPS, s.Bytes)
	}
}

func TestSummarizeFailures(t *testing.T) {
	res := []Result{
		{Duration: time.Millisecond, Status: 200},
		{Duration: time.Millisecond, Status: 500, Fail: FailStatus},
		{Duration: time.Hour, Fail: FailConnect},
	}
	s := Summarize(res, time.Second)
	if s.Completed != 1 || s.Failed != 2 {
		t.Fatalf("completed=%d failed=%d", s.Completed, s.Failed)
	}
	if s.Failures[FailStatus] != 1 || s.Failures[FailConnect] != 1 {
		t.Fatalf("failures = %v", s.Failures)
	}
	if s.Latency.Max != 1 {
		t.Fatalf("connect failure leaked into latency: %v", s.Latency.Max)
	}
	if s.StatusCodes[500] != 1 {
		t.Fatalf("status codes = %v", s.StatusCodes)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	s := Summarize(nil, 0)
	if s.Completed != 0 || len(s.Percentiles) != 0 {
		t.Fatalf("%+v", s)
	}
}
