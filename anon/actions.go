package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// action transforms one cell value.
type action func(string) (string, error)

func newAction(r *Rule, c *Config) (action, error) {
	switch r.Action {
	case "nothing":
		return func(s string) (string, error) { return s, nil }, nil
	case "hash":
		salt := r.Salt
		if salt == "" {
			salt = c.Salt
		}
		if salt == "" {
			return nil, fmt.Errorf("hash needs a salt (rule or top-level), random salts make output unjoinable")
		}
		return hashAction(salt), nil
	case "outcode":
		return outcode, nil
	case "year", "date":
		return dateAction(r)
	case "range":
		return rangeAction(r)
	case "mask":
		return maskAction(r)
	case "redact":
		rep := r.Char
		if rep == "" {
			rep = "REDACTED"
		}
		return func(s string) (string, error) {
			if s == "" {
				return s, nil
			}
			return rep, nil
		}, nil
	case "ip":
		return ipAction, nil
	case "":
		return nil, fmt.Errorf("missing action")
	}
	return nil, fmt.Errorf("unknown action %q", r.Action)
}

func hashAction(salt string) action {
	key := []byte(salt)
	return func(s string) (string, error) {
		if s == "" {
			return s, nil
		}
		m := hmac.New(sha256.New, key)
		m.Write([]byte(s))
		return hex.EncodeToString(m.Sum(nil)), nil
	}
}

var postcodeRe = regexp.MustCompile(`^\s*([A-Za-z]{1,2}[0-9][A-Za-z0-9]?)\s*[0-9][A-Za-z]{2}\s*$`)

func outcode(s string) (string, error) {
	if strings.TrimSpace(s) == "" {
		return s, nil
	}
	m := postcodeRe.FindStringSubmatch(s)
	if m == nil {
		return "", fmt.Errorf("%q is not a UK postcode", s)
	}
	return strings.ToUpper(m[1]), nil
}

var dateTokens = strings.NewReplacer(
	"YYYY", "2006", "YY", "06", "MMMM", "January", "MMM", "Jan",
	"MM", "01", "DD", "02", "hh", "15", "mm", "04", "ss", "05",
)

// dateAction parses with Format (tokens YYYY MM DD hh mm ss MMM MMMM) and
// writes OutputFormat, which defaults to the year alone for "year".
func dateAction(r *Rule) (action, error) {
	if r.Format == "" {
		return nil, fmt.Errorf("date and year need a format, e.g. YYYY-MM-DD")
	}
	in := dateTokens.Replace(r.Format)
	out := "2006"
	if r.Action == "date" {
		if r.OutputFormat == "" {
			return nil, fmt.Errorf("date needs an outputFormat, e.g. YYYY-MM")
		}
		out = dateTokens.Replace(r.OutputFormat)
	}
	return func(s string) (string, error) {
		if s == "" {
			return s, nil
		}
		t, err := time.Parse(in, s)
		if err != nil {
			return "", fmt.Errorf("%q does not match date format %s", s, r.Format)
		}
		return t.Format(out), nil
	}, nil
}

func rangeAction(r *Rule) (action, error) {
	if len(r.Ranges) == 0 {
		return nil, fmt.Errorf("range needs at least one range")
	}
	for i, g := range r.Ranges {
		if (g.Gt != nil && g.Gte != nil) || (g.Lt != nil && g.Lte != nil) {
			return nil, fmt.Errorf("range %d: use only one of gt/gte and one of lt/lte", i)
		}
		if g.Gt == nil && g.Gte == nil && g.Lt == nil && g.Lte == nil {
			return nil, fmt.Errorf("range %d: needs a bound", i)
		}
	}
	return func(s string) (string, error) {
		if s == "" {
			return s, nil
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return "", fmt.Errorf("%q is not a number", s)
		}
		for _, g := range r.Ranges {
			if (g.Gt == nil || v > *g.Gt) && (g.Gte == nil || v >= *g.Gte) &&
				(g.Lt == nil || v < *g.Lt) && (g.Lte == nil || v <= *g.Lte) {
				return g.Output, nil
			}
		}
		return "", fmt.Errorf("%q is outside every range", s)
	}, nil
}

func maskAction(r *Rule) (action, error) {
	if r.Keep < 0 {
		return nil, fmt.Errorf("keep must not be negative")
	}
	ch := "*"
	if r.Char != "" {
		ch = r.Char
	}
	return func(s string) (string, error) {
		rs := []rune(s)
		if len(rs) <= r.Keep {
			return strings.Repeat(ch, len(rs)), nil
		}
		return strings.Repeat(ch, len(rs)-r.Keep) + string(rs[len(rs)-r.Keep:]), nil
	}, nil
}

// ipAction zeroes the host part: last octet of IPv4, last 80 bits of IPv6.
func ipAction(s string) (string, error) {
	if s == "" {
		return s, nil
	}
	a, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil {
		return "", fmt.Errorf("%q is not an IP address", s)
	}
	bits := 24
	if a.Is6() && !a.Is4In6() {
		bits = 48
	}
	a = a.Unmap()
	if a.Is4() {
		bits = 24
	}
	p, _ := a.Prefix(bits)
	return p.Masked().Addr().String(), nil
}
