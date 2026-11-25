package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
)

type compiled struct {
	rule Rule
	fn   action
}

// Process streams CSV from r to w applying the configured rules.
func Process(cfg *Config, r io.Reader, w io.Writer) error {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.ReuseRecord = true
	cw := csv.NewWriter(w)
	if cfg.Delimiter != "" {
		d := []rune(cfg.Delimiter)[0]
		cr.Comma, cw.Comma = d, d
	}

	var rules []compiled
	for i := range cfg.Rules {
		fn, err := newAction(&cfg.Rules[i], cfg)
		if err != nil {
			return err
		}
		rules = append(rules, compiled{cfg.Rules[i], fn})
	}

	var header []string
	var resolved []int // column index per rule
	var sampleIdx = -1
	line := 0

	resolve := func(c Column) (int, error) {
		if c.ByIdx {
			return c.Index, nil
		}
		for i, h := range header {
			if h == c.Name {
				return i, nil
			}
		}
		return 0, fmt.Errorf("column %q not found in header", c.Name)
	}
	setup := func() error {
		resolved = make([]int, len(rules))
		for i, ru := range rules {
			idx, err := resolve(ru.rule.Column)
			if err != nil {
				return err
			}
			resolved[i] = idx
		}
		if cfg.Sampling != nil {
			idx, err := resolve(cfg.Sampling.Column)
			if err != nil {
				return err
			}
			sampleIdx = idx
		}
		return nil
	}

	if !cfg.HasHeader() {
		if err := setup(); err != nil {
			return err
		}
	}

	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		line++
		if line == 1 && cfg.HasHeader() {
			header = append([]string(nil), rec...)
			if err := setup(); err != nil {
				return err
			}
			if err := cw.Write(rec); err != nil {
				return err
			}
			continue
		}
		if cfg.Sampling != nil {
			if sampleIdx >= len(rec) {
				return fmt.Errorf("line %d: sampling column missing", line)
			}
			h := fnv.New32()
			h.Write([]byte(rec[sampleIdx]))
			if h.Sum32()%cfg.Sampling.Mod != 0 {
				continue
			}
		}
		for i, ru := range rules {
			idx := resolved[i]
			if idx >= len(rec) {
				return fmt.Errorf("line %d: column %s missing (row has %d fields)", line, ru.rule.Column, len(rec))
			}
			v, err := ru.fn(rec[idx])
			if err != nil {
				return fmt.Errorf("line %d, column %s: %w", line, ru.rule.Column, err)
			}
			rec[idx] = v
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
