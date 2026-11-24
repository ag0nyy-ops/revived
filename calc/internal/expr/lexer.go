// Package expr lexes, parses and evaluates arithmetic expressions.
package expr

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type tokKind int

const (
	tEOF tokKind = iota
	tNum
	tIdent
	tOp
	tLParen
	tRParen
	tComma
	tAt
)

type token struct {
	kind tokKind
	text string
	num  float64
	pos  int // 1-based column
}

// Error is a syntax or evaluation error with a column.
type Error struct {
	Pos int
	Msg string
}

func (e *Error) Error() string {
	if e.Pos > 0 {
		return fmt.Sprintf("column %d: %s", e.Pos, e.Msg)
	}
	return e.Msg
}

func errAt(pos int, f string, a ...any) *Error {
	return &Error{Pos: pos, Msg: fmt.Sprintf(f, a...)}
}

func lex(src string) ([]token, error) {
	rs := []rune(src)
	var out []token
	for i := 0; i < len(rs); {
		r := rs[i]
		pos := i + 1
		switch {
		case unicode.IsSpace(r):
			i++
		case unicode.IsDigit(r) || (r == '.' && i+1 < len(rs) && unicode.IsDigit(rs[i+1])):
			j := i
			for j < len(rs) && (unicode.IsDigit(rs[j]) || rs[j] == '.' || rs[j] == '_') {
				j++
			}
			if j < len(rs) && (rs[j] == 'e' || rs[j] == 'E') {
				k := j + 1
				if k < len(rs) && (rs[k] == '+' || rs[k] == '-') {
					k++
				}
				if k < len(rs) && unicode.IsDigit(rs[k]) {
					for k < len(rs) && unicode.IsDigit(rs[k]) {
						k++
					}
					j = k
				}
			}
			text := strings.ReplaceAll(string(rs[i:j]), "_", "")
			v, err := strconv.ParseFloat(text, 64)
			if err != nil {
				return nil, errAt(pos, "invalid number %q", text)
			}
			out = append(out, token{tNum, text, v, pos})
			i = j
		case unicode.IsLetter(r) || r == '_':
			j := i
			for j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsDigit(rs[j]) || rs[j] == '_') {
				j++
			}
			out = append(out, token{kind: tIdent, text: string(rs[i:j]), pos: pos})
			i = j
		case r == '@':
			j := i
			for j < len(rs) && rs[j] == '@' {
				j++
			}
			out = append(out, token{kind: tAt, text: string(rs[i:j]), pos: pos})
			i = j
		case strings.ContainsRune("+-*/^%!=", r):
			out = append(out, token{kind: tOp, text: string(r), pos: pos})
			i++
		case r == '×':
			out = append(out, token{kind: tOp, text: "*", pos: pos})
			i++
		case r == '÷':
			out = append(out, token{kind: tOp, text: "/", pos: pos})
			i++
		case r == '(':
			out = append(out, token{kind: tLParen, text: "(", pos: pos})
			i++
		case r == ')':
			out = append(out, token{kind: tRParen, text: ")", pos: pos})
			i++
		case r == ',':
			out = append(out, token{kind: tComma, text: ",", pos: pos})
			i++
		default:
			return nil, errAt(pos, "unexpected character %q", r)
		}
	}
	out = append(out, token{kind: tEOF, pos: len(rs) + 1})
	return out, nil
}
