package main

import (
	"math"
	"sort"
	"time"
)

// Failure categories.
const (
	FailNone    = ""
	FailConnect = "connect"
	FailRead    = "read"
	FailLength  = "length"
	FailStatus  = "status"
	FailTimeout = "timeout"
)

// Result is the outcome of one request.
type Result struct {
	Duration time.Duration
	Status   int
	Bytes    int64
	Fail     string
}

// Summary aggregates a finished (or interrupted) run.
type Summary struct {
	Completed   int             `json:"completed"`
	Failed      int             `json:"failed"`
	Failures    map[string]int  `json:"failures"`
	StatusCodes map[int]int     `json:"status_codes"`
	Elapsed     float64         `json:"elapsed_seconds"`
	Bytes       int64           `json:"bytes_received"`
	RPS         float64         `json:"requests_per_second"`
	MBPerSec    float64         `json:"megabytes_per_second"`
	Latency     Latency         `json:"latency_ms"`
	Percentiles map[int]float64 `json:"percentiles_ms"`
}

// Latency holds basic distribution numbers in milliseconds.
type Latency struct {
	Min    float64 `json:"min"`
	Mean   float64 `json:"mean"`
	Stddev float64 `json:"stddev"`
	Median float64 `json:"median"`
	Max    float64 `json:"max"`
}

// PercentileSteps are the points reported in the percentile table.
var PercentileSteps = []int{50, 66, 75, 80, 90, 95, 98, 99, 100}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

// percentile returns the nearest-rank percentile of sorted durations.
func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	rank := int(math.Ceil(float64(p) / 100 * float64(len(sorted))))
	if rank < 1 {
		rank = 1
	}
	if rank > len(sorted) {
		rank = len(sorted)
	}
	return sorted[rank-1]
}

// Summarize folds results into a Summary. Latency statistics only include
// requests that received a response, so connection errors do not skew them.
func Summarize(results []Result, elapsed time.Duration) Summary {
	s := Summary{
		Failures:    map[string]int{},
		StatusCodes: map[int]int{},
		Percentiles: map[int]float64{},
		Elapsed:     elapsed.Seconds(),
	}
	var durs []time.Duration
	for _, r := range results {
		if r.Fail != FailNone {
			s.Failed++
			s.Failures[r.Fail]++
		} else {
			s.Completed++
		}
		if r.Status != 0 {
			s.StatusCodes[r.Status]++
		}
		s.Bytes += r.Bytes
		if r.Fail == FailConnect || r.Fail == FailTimeout {
			continue
		}
		durs = append(durs, r.Duration)
	}
	if elapsed > 0 {
		s.RPS = float64(len(results)) / elapsed.Seconds()
		s.MBPerSec = float64(s.Bytes) / 1e6 / elapsed.Seconds()
	}
	if len(durs) == 0 {
		return s
	}
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
	var sum float64
	for _, d := range durs {
		sum += ms(d)
	}
	mean := sum / float64(len(durs))
	var sq float64
	for _, d := range durs {
		sq += (ms(d) - mean) * (ms(d) - mean)
	}
	s.Latency = Latency{
		Min:    ms(durs[0]),
		Mean:   mean,
		Stddev: math.Sqrt(sq / float64(len(durs))),
		Median: ms(percentile(durs, 50)),
		Max:    ms(durs[len(durs)-1]),
	}
	for _, p := range PercentileSteps {
		s.Percentiles[p] = ms(percentile(durs, p))
	}
	return s
}
