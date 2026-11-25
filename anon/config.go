package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Column is a reference to a CSV column, by header name or by 0-based index.
type Column struct {
	Name  string
	Index int
	ByIdx bool
}

func (c *Column) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		c.Name = s
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("column must be a header name or a 0-based index")
	}
	if n < 0 {
		return fmt.Errorf("column index must not be negative")
	}
	c.Index, c.ByIdx = n, true
	return nil
}

func (c Column) String() string {
	if c.ByIdx {
		return fmt.Sprintf("#%d", c.Index)
	}
	return c.Name
}

// Rule describes the action applied to one column.
type Rule struct {
	Column Column `json:"column"`
	Action string `json:"action"`
	// date
	Format       string `json:"format,omitempty"`
	OutputFormat string `json:"outputFormat,omitempty"`
	// range
	Ranges []Range `json:"ranges,omitempty"`
	// mask
	Keep int `json:"keep,omitempty"`
	// redact / mask
	Char string `json:"char,omitempty"`
	// hash
	Salt string `json:"salt,omitempty"`
}

type Range struct {
	Gt     *float64 `json:"gt,omitempty"`
	Gte    *float64 `json:"gte,omitempty"`
	Lt     *float64 `json:"lt,omitempty"`
	Lte    *float64 `json:"lte,omitempty"`
	Output string   `json:"output"`
}

type Sampling struct {
	Mod    uint32 `json:"mod"`
	Column Column `json:"column"`
}

type Config struct {
	Delimiter string    `json:"delimiter,omitempty"`
	Header    *bool     `json:"header,omitempty"`
	Salt      string    `json:"salt,omitempty"`
	Sampling  *Sampling `json:"sampling,omitempty"`
	Rules     []Rule    `json:"rules"`
}

func (c *Config) HasHeader() bool { return c.Header == nil || *c.Header }

func LoadConfig(path, envSalt string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var c Config
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if c.Salt == "" {
		c.Salt = envSalt
	}
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &c, nil
}

func (c *Config) validate() error {
	if len([]rune(c.Delimiter)) > 1 {
		return fmt.Errorf("delimiter must be a single character")
	}
	if c.Sampling != nil && c.Sampling.Mod == 0 {
		return fmt.Errorf("sampling.mod must be greater than 0")
	}
	for i := range c.Rules {
		r := &c.Rules[i]
		if !c.HasHeader() && !r.Column.ByIdx {
			return fmt.Errorf("rule %d: column names need a header row, use an index", i)
		}
		if _, err := newAction(r, c); err != nil {
			return fmt.Errorf("rule %d (%s): %w", i, r.Column, err)
		}
	}
	return nil
}
