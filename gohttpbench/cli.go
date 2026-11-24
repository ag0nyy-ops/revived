package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync/atomic"
)

// run is the testable entry point. It returns the process exit code:
// 0 success, 1 benchmark had failed requests, 2 usage error.
func run(args []string, stdout, stderr io.Writer) int {
	return runContext(context.Background(), args, stdout, stderr)
}

func runContext(parent context.Context, args []string, stdout, stderr io.Writer) int {
	c, err := parseArgs(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, "gohttpbench:", err)
		return 2
	}

	ctx, stop := signal.NotifyContext(parent, os.Interrupt)
	defer stop()

	var last atomic.Int64
	progress := func(done int) {
		if c.Quiet || c.JSON {
			return
		}
		step := c.Requests / 10
		if c.Duration > 0 || step < 1 {
			return
		}
		if done%step == 0 && int64(done) > last.Load() {
			last.Store(int64(done))
			fmt.Fprintf(stderr, "Completed %d requests\n", done)
		}
	}
	if !c.Quiet && !c.JSON {
		fmt.Fprintf(stderr, "Benchmarking %s (be patient)\n", c.URL)
	}

	results, elapsed := Run(ctx, c, progress)
	interrupted := parent.Err() == nil && ctx.Err() != nil && !(c.Duration > 0 && elapsed >= c.Duration)
	s := Summarize(results, elapsed)
	if c.JSON {
		if err := writeJSON(stdout, s, interrupted); err != nil {
			fmt.Fprintln(stderr, "gohttpbench:", err)
			return 2
		}
	} else {
		writeText(stdout, c, s, interrupted)
	}
	if s.Failed > 0 {
		return 1
	}
	return 0
}
