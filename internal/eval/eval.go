package eval

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// ─── Lexer ───────────────────────────────────────────────────────────────────

type tokKind int

const (
	tokNum tokKind = iota
	tokIdent
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokPercent
	tokCaret
	tokLParen
	tokRParen
	tokComma
	tokEOF
)

type token struct {
	kind tokKind
	str  string
	num  float64
}

type lexer struct {
	src []rune
	pos int
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *lexer) advance() {
	if l.pos < len(l.src) {
		l.pos++
	}
}

func (l *lexer) skipWS() {
	for l.pos < len(l.src) && unicode.IsSpace(l.src[l.pos]) {
		l.pos++
	}
}

func (l *lexer) next() (token, error) {
	l.skipWS()
	if l.pos >= len(l.src) {
		return token{kind: tokEOF}, nil
	}
	ch := l.peek()
	switch {
	case unicode.IsDigit(ch) || (ch == '.' && l.pos+1 < len(l.src) && unicode.IsDigit(l.src[l.pos+1])):
		return l.readNum()
	case unicode.IsLetter(ch) || ch == '_':
		return l.readIdent()
	case ch == '+':
		l.advance()
		return token{kind: tokPlus, str: "+"}, nil
	case ch == '-':
		l.advance()
		return token{kind: tokMinus, str: "-"}, nil
	case ch == '*':
		l.advance()
		return token{kind: tokStar, str: "*"}, nil
	case ch == '/':
		l.advance()
		return token{kind: tokSlash, str: "/"}, nil
	case ch == '%':
		l.advance()
		return token{kind: tokPercent, str: "%"}, nil
	case ch == '^':
		l.advance()
		return token{kind: tokCaret, str: "^"}, nil
	case ch == '(':
		l.advance()
		return token{kind: tokLParen, str: "("}, nil
	case ch == ')':
		l.advance()
		return token{kind: tokRParen, str: ")"}, nil
	case ch == ',':
		l.advance()
		return token{kind: tokComma, str: ","}, nil
	default:
		l.advance()
		return token{}, fmt.Errorf("unexpected character: %c", ch)
	}
}

func (l *lexer) readNum() (token, error) {
	start := l.pos
	for l.pos < len(l.src) && (unicode.IsDigit(l.src[l.pos]) || l.src[l.pos] == '.') {
		l.pos++
	}
	if l.pos < len(l.src) && (l.src[l.pos] == 'e' || l.src[l.pos] == 'E') {
		l.pos++
		if l.pos < len(l.src) && (l.src[l.pos] == '+' || l.src[l.pos] == '-') {
			l.pos++
		}
		for l.pos < len(l.src) && unicode.IsDigit(l.src[l.pos]) {
			l.pos++
		}
	}
	s := string(l.src[start:l.pos])
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return token{}, fmt.Errorf("invalid number %q", s)
	}
	return token{kind: tokNum, str: s, num: v}, nil
}

func (l *lexer) readIdent() (token, error) {
	start := l.pos
	for l.pos < len(l.src) && (unicode.IsLetter(l.src[l.pos]) || unicode.IsDigit(l.src[l.pos]) || l.src[l.pos] == '_') {
		l.pos++
	}
	return token{kind: tokIdent, str: string(l.src[start:l.pos])}, nil
}

// ─── Parser ──────────────────────────────────────────────────────────────────

type parser struct {
	lex  *lexer
	cur  token
	vars map[string]float64
}

func newParser(expr string, vars map[string]float64) (*parser, error) {
	l := &lexer{src: []rune(expr)}
	tok, err := l.next()
	if err != nil {
		return nil, err
	}
	return &parser{lex: l, cur: tok, vars: vars}, nil
}

func (p *parser) consume() error {
	tok, err := p.lex.next()
	if err != nil {
		return err
	}
	p.cur = tok
	return nil
}

func (p *parser) parse() (float64, error) {
	v, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	if p.cur.kind != tokEOF {
		return 0, fmt.Errorf("unexpected %q", p.cur.str)
	}
	return v, nil
}

func (p *parser) parseExpr() (float64, error) { return p.parseAddSub() }

func (p *parser) parseAddSub() (float64, error) {
	v, err := p.parseMulDiv()
	if err != nil {
		return 0, err
	}
	for p.cur.kind == tokPlus || p.cur.kind == tokMinus {
		op := p.cur.kind
		if err := p.consume(); err != nil {
			return 0, err
		}
		r, err := p.parseMulDiv()
		if err != nil {
			return 0, err
		}
		if op == tokPlus {
			v += r
		} else {
			v -= r
		}
	}
	return v, nil
}

func (p *parser) parseMulDiv() (float64, error) {
	v, err := p.parsePow()
	if err != nil {
		return 0, err
	}
	for p.cur.kind == tokStar || p.cur.kind == tokSlash || p.cur.kind == tokPercent {
		op := p.cur.kind
		if err := p.consume(); err != nil {
			return 0, err
		}
		r, err := p.parsePow()
		if err != nil {
			return 0, err
		}
		switch op {
		case tokStar:
			v *= r
		case tokSlash:
			if r == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			v /= r
		case tokPercent:
			if r == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			v = math.Mod(v, r)
		}
	}
	return v, nil
}

func (p *parser) parsePow() (float64, error) {
	base, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	if p.cur.kind == tokCaret {
		if err := p.consume(); err != nil {
			return 0, err
		}
		exp, err := p.parsePow() // right-associative
		if err != nil {
			return 0, err
		}
		return math.Pow(base, exp), nil
	}
	return base, nil
}

func (p *parser) parseUnary() (float64, error) {
	if p.cur.kind == tokMinus {
		if err := p.consume(); err != nil {
			return 0, err
		}
		v, err := p.parseUnary()
		return -v, err
	}
	if p.cur.kind == tokPlus {
		if err := p.consume(); err != nil {
			return 0, err
		}
		return p.parseUnary()
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (float64, error) {
	switch p.cur.kind {
	case tokNum:
		v := p.cur.num
		return v, p.consume()
	case tokLParen:
		if err := p.consume(); err != nil {
			return 0, err
		}
		v, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if p.cur.kind != tokRParen {
			return 0, fmt.Errorf("expected ')'")
		}
		return v, p.consume()
	case tokIdent:
		name := p.cur.str
		if err := p.consume(); err != nil {
			return 0, err
		}
		if p.cur.kind == tokLParen {
			return p.callBuiltin(name)
		}
		v, ok := p.vars[strings.ToLower(name)]
		if !ok {
			return 0, fmt.Errorf("undefined: %s", name)
		}
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected %q", p.cur.str)
	}
}

func (p *parser) callBuiltin(name string) (float64, error) {
	if err := p.consume(); err != nil { // consume '('
		return 0, err
	}
	var args []float64
	if p.cur.kind != tokRParen {
		v, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		args = append(args, v)
		for p.cur.kind == tokComma {
			if err := p.consume(); err != nil {
				return 0, err
			}
			v, err := p.parseExpr()
			if err != nil {
				return 0, err
			}
			args = append(args, v)
		}
	}
	if p.cur.kind != tokRParen {
		return 0, fmt.Errorf("expected ')' after args")
	}
	if err := p.consume(); err != nil {
		return 0, err
	}
	return evalBuiltin(strings.ToLower(name), args)
}

func nargs(name string, args []float64, n int) error {
	if len(args) != n {
		return fmt.Errorf("%s takes %d arg(s), got %d", name, n, len(args))
	}
	return nil
}

func evalBuiltin(name string, args []float64) (float64, error) {
	switch name {
	case "sqrt":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Sqrt(args[0]), nil
	case "abs":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Abs(args[0]), nil
	case "floor":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Floor(args[0]), nil
	case "ceil":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Ceil(args[0]), nil
	case "round":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Round(args[0]), nil
	case "sin":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Sin(args[0]), nil
	case "cos":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Cos(args[0]), nil
	case "tan":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Tan(args[0]), nil
	case "asin":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Asin(args[0]), nil
	case "acos":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Acos(args[0]), nil
	case "atan":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Atan(args[0]), nil
	case "atan2":
		if err := nargs(name, args, 2); err != nil {
			return 0, err
		}
		return math.Atan2(args[0], args[1]), nil
	case "log", "ln":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Log(args[0]), nil
	case "log2":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Log2(args[0]), nil
	case "log10":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Log10(args[0]), nil
	case "exp":
		if err := nargs(name, args, 1); err != nil {
			return 0, err
		}
		return math.Exp(args[0]), nil
	case "pow":
		if err := nargs(name, args, 2); err != nil {
			return 0, err
		}
		return math.Pow(args[0], args[1]), nil
	case "max":
		if len(args) == 0 {
			return 0, fmt.Errorf("max requires at least 1 arg")
		}
		m := args[0]
		for _, a := range args[1:] {
			if a > m {
				m = a
			}
		}
		return m, nil
	case "min":
		if len(args) == 0 {
			return 0, fmt.Errorf("min requires at least 1 arg")
		}
		m := args[0]
		for _, a := range args[1:] {
			if a < m {
				m = a
			}
		}
		return m, nil
	case "sum":
		v := 0.0
		for _, a := range args {
			v += a
		}
		return v, nil
	default:
		return 0, fmt.Errorf("unknown function: %s", name)
	}
}

// ─── Evaluator ───────────────────────────────────────────────────────────────

// Evaluator evaluates lines sequentially, maintaining variable state.
type Evaluator struct {
	vars       map[string]float64
	pendingSum float64
}

func New() *Evaluator {
	e := &Evaluator{}
	e.Reset()
	return e
}

func (e *Evaluator) Reset() {
	e.vars = map[string]float64{
		"pi":  math.Pi,
		"e":   math.E,
		"tau": 2 * math.Pi,
		"phi": (1 + math.Sqrt(5)) / 2,
	}
	e.pendingSum = 0
}

// EvalLine evaluates a single line, updating variables.
// Returns (result, errMsg). Both empty means no output (blank/comment line).
func (e *Evaluator) EvalLine(line string) (result, errMsg string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
		return "", ""
	}

	if strings.ToLower(trimmed) == "sum-total" {
		total := e.pendingSum
		e.pendingSum = 0
		e.vars["last"] = total
		return FormatNum(total), ""
	}

	if varName, exprStr, ok := detectAssignment(trimmed); ok {
		p, err := newParser(exprStr, e.vars)
		if err != nil {
			return "", err.Error()
		}
		v, err := p.parse()
		if err != nil {
			return "", err.Error()
		}
		e.vars[strings.ToLower(varName)] = v
		e.vars["last"] = v
		e.pendingSum += v
		return varName + " = " + FormatNum(v), ""
	}

	p, err := newParser(trimmed, e.vars)
	if err != nil {
		return "", err.Error()
	}
	v, err := p.parse()
	if err != nil {
		return "", err.Error()
	}
	e.vars["last"] = v
	e.pendingSum += v
	return FormatNum(v), ""
}

func detectAssignment(line string) (varName, expr string, ok bool) {
	for i := 0; i < len(line); i++ {
		if line[i] != '=' {
			continue
		}
		if i+1 < len(line) && line[i+1] == '=' {
			continue // ==
		}
		if i > 0 && (line[i-1] == '!' || line[i-1] == '<' || line[i-1] == '>') {
			continue // !=, <=, >=
		}
		lhs := strings.TrimSpace(line[:i])
		rhs := strings.TrimSpace(line[i+1:])
		if isIdent(lhs) && rhs != "" {
			return lhs, rhs, true
		}
	}
	return "", "", false
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			return false
		}
		if i > 0 && !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

func FormatNum(v float64) string {
	if math.IsNaN(v) {
		return "NaN"
	}
	if math.IsInf(v, 1) {
		return "∞"
	}
	if math.IsInf(v, -1) {
		return "-∞"
	}
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	s := strconv.FormatFloat(v, 'f', 10, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}
