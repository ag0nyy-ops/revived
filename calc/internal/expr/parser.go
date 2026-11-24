package expr

import "math"

// Env holds variables and previous results across evaluations.
type Env struct {
	Vars    map[string]float64
	History []float64 // oldest first
}

// NewEnv returns an empty environment.
func NewEnv() *Env { return &Env{Vars: map[string]float64{}} }

type parser struct {
	toks []token
	i    int
	env  *Env
}

// Eval evaluates one statement. A statement is an expression or an
// assignment ("name = expr"). The result is pushed onto the history.
func (e *Env) Eval(src string) (float64, error) {
	toks, err := lex(src)
	if err != nil {
		return 0, err
	}
	p := &parser{toks: toks, env: e}
	if p.peek().kind == tEOF {
		return 0, errAt(1, "empty expression")
	}
	var name string
	if t := p.peek(); t.kind == tIdent && p.toks[p.i+1].kind == tOp && p.toks[p.i+1].text == "=" {
		name = t.text
		if _, ok := constants[name]; ok || name == "ans" {
			return 0, errAt(t.pos, "cannot assign to %q", name)
		}
		if _, ok := functions[name]; ok {
			return 0, errAt(t.pos, "cannot assign to function %q", name)
		}
		p.i += 2
	}
	v, err := p.expr()
	if err != nil {
		return 0, err
	}
	if t := p.peek(); t.kind != tEOF {
		if t.kind == tRParen {
			return 0, errAt(t.pos, "unmatched )")
		}
		return 0, errAt(t.pos, "unexpected %q", t.text)
	}
	if math.IsNaN(v) {
		return 0, &Error{Msg: "result is not a number"}
	}
	if name != "" {
		e.Vars[name] = v
	}
	e.History = append(e.History, v)
	return v, nil
}

func (p *parser) peek() token { return p.toks[p.i] }
func (p *parser) next() token { t := p.toks[p.i]; p.i++; return t }

func (p *parser) isOp(s string) bool {
	t := p.peek()
	return t.kind == tOp && t.text == s
}

func (p *parser) expr() (float64, error) {
	l, err := p.term()
	if err != nil {
		return 0, err
	}
	for p.isOp("+") || p.isOp("-") {
		op := p.next().text
		r, err := p.term()
		if err != nil {
			return 0, err
		}
		if op == "+" {
			l += r
		} else {
			l -= r
		}
	}
	return l, nil
}

// startsOperand reports whether the next token can begin an implicitly
// multiplied operand, as in "3pi" or "2(3+4)".
func (p *parser) startsOperand() bool {
	switch p.peek().kind {
	case tNum, tIdent, tLParen, tAt:
		return true
	}
	return false
}

func (p *parser) term() (float64, error) {
	l, err := p.unary()
	if err != nil {
		return 0, err
	}
	for {
		switch {
		case p.isOp("*") || p.isOp("/") || p.isOp("%"):
			t := p.next()
			r, err := p.unary()
			if err != nil {
				return 0, err
			}
			switch t.text {
			case "*":
				l *= r
			case "/":
				if r == 0 {
					return 0, errAt(t.pos, "division by zero")
				}
				l /= r
			default:
				if r == 0 {
					return 0, errAt(t.pos, "modulo by zero")
				}
				l = math.Mod(l, r)
			}
		case p.startsOperand():
			r, err := p.power()
			if err != nil {
				return 0, err
			}
			l *= r
		default:
			return l, nil
		}
	}
}

func (p *parser) unary() (float64, error) {
	if p.isOp("-") {
		p.next()
		v, err := p.unary()
		return -v, err
	}
	if p.isOp("+") {
		p.next()
		return p.unary()
	}
	return p.power()
}

// power is right associative and binds tighter than unary minus, so
// -2^2 is -4 and 2^-2 is 0.25.
func (p *parser) power() (float64, error) {
	b, err := p.postfix()
	if err != nil {
		return 0, err
	}
	if p.isOp("^") {
		p.next()
		e, err := p.unary()
		if err != nil {
			return 0, err
		}
		return math.Pow(b, e), nil
	}
	return b, nil
}

func (p *parser) postfix() (float64, error) {
	v, err := p.primary()
	if err != nil {
		return 0, err
	}
	for p.isOp("!") {
		p.next()
		if v, err = factorial(v); err != nil {
			return 0, err
		}
	}
	return v, nil
}

func (p *parser) primary() (float64, error) {
	t := p.next()
	switch t.kind {
	case tNum:
		return t.num, nil
	case tAt:
		n := len(t.text)
		if n > len(p.env.History) {
			return 0, errAt(t.pos, "no result %d back (history has %d)", n, len(p.env.History))
		}
		return p.env.History[len(p.env.History)-n], nil
	case tLParen:
		v, err := p.expr()
		if err != nil {
			return 0, err
		}
		if c := p.next(); c.kind != tRParen {
			return 0, errAt(t.pos, "unclosed (")
		}
		return v, nil
	case tIdent:
		if fn, ok := functions[t.text]; ok {
			return p.call(t, fn)
		}
		if t.text == "ans" {
			if len(p.env.History) == 0 {
				return 0, errAt(t.pos, "no previous result")
			}
			return p.env.History[len(p.env.History)-1], nil
		}
		if v, ok := p.env.Vars[t.text]; ok {
			return v, nil
		}
		if v, ok := constants[t.text]; ok {
			return v, nil
		}
		return 0, errAt(t.pos, "unknown name %q", t.text)
	case tEOF:
		return 0, errAt(t.pos, "unexpected end of expression")
	}
	return 0, errAt(t.pos, "unexpected %q", t.text)
}

func (p *parser) call(name token, fn function) (float64, error) {
	if p.peek().kind != tLParen {
		return 0, errAt(name.pos, "function %q needs arguments in parentheses", name.text)
	}
	open := p.next()
	var args []float64
	if p.peek().kind != tRParen {
		for {
			v, err := p.expr()
			if err != nil {
				return 0, err
			}
			args = append(args, v)
			if p.peek().kind == tComma {
				p.next()
				continue
			}
			break
		}
	}
	if p.peek().kind != tRParen {
		return 0, errAt(open.pos, "unclosed (")
	}
	p.next()
	if len(args) < fn.min || (fn.max >= 0 && len(args) > fn.max) {
		return 0, errAt(name.pos, "%s: wrong number of arguments (%d)", name.text, len(args))
	}
	return fn.fn(args)
}
