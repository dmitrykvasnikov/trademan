package signal

import (
	"fmt"
	"strconv"
)

// parse turns a rule's source text into an expression tree, checking that every
// candle reference falls within a body of the given length. It reports the first
// problem it meets — a stray character, a reference past the body, a missing
// operator or bracket — with the position it is at, so a mistyped rule says where
// it went wrong rather than just failing.
func parse(src string, body int) (expr, error) {
	toks, err := lex(src)
	if err != nil {
		return nil, err
	}

	p := &parser{toks: toks, body: body}
	e, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if tok := p.peek(); tok.kind != tokEnd {
		return nil, fmt.Errorf("signal: unexpected %s at position %d", describe(tok), tok.pos)
	}
	return e, nil
}

// tokenKind enumerates the pieces a rule is built from.
type tokenKind int

const (
	tokEnd tokenKind = iota
	tokLParen
	tokRParen
	tokAnd
	tokOr
	tokNot
	tokRef
	tokNumber
	tokOp
)

// token is one lexed piece: the operator or reference text where it matters, and
// always the position it started at for error messages.
type token struct {
	kind tokenKind
	text string
	ref  ref
	num  float64
	pos  int
}

// describe names a token for an error message.
func describe(t token) string {
	switch t.kind {
	case tokEnd:
		return "end of rule"
	case tokLParen:
		return "'('"
	case tokRParen:
		return "')'"
	case tokAnd:
		return "AND"
	case tokOr:
		return "OR"
	case tokNot:
		return "NOT"
	case tokRef:
		return fmt.Sprintf("%q", t.ref.text)
	case tokNumber:
		return strconv.FormatFloat(t.num, 'g', -1, 64)
	case tokOp:
		return fmt.Sprintf("%q", t.text)
	}
	return "?"
}

// lex splits src into tokens. Whitespace separates but is otherwise ignored, so
// a rule can be spaced however reads best.
func lex(src string) ([]token, error) {
	var toks []token

	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '(':
			toks = append(toks, token{kind: tokLParen, pos: i})
			i++
		case c == ')':
			toks = append(toks, token{kind: tokRParen, pos: i})
			i++
		case c == '<' || c == '>' || c == '=' || c == '!':
			op, n, err := lexOp(src, i)
			if err != nil {
				return nil, err
			}
			toks = append(toks, token{kind: tokOp, text: op, pos: i})
			i += n
		case c >= '0' && c <= '9' || c == '.':
			num, n, err := lexNumber(src, i)
			if err != nil {
				return nil, err
			}
			toks = append(toks, token{kind: tokNumber, num: num, pos: i})
			i += n
		case isLetter(c):
			word, n := lexWord(src, i)
			tok, err := classify(word, i)
			if err != nil {
				return nil, err
			}
			toks = append(toks, tok)
			i += n
		default:
			return nil, fmt.Errorf("signal: unexpected character %q at position %d", string(c), i)
		}
	}

	return append(toks, token{kind: tokEnd, pos: len(src)}), nil
}

// lexOp reads a comparison operator. A lone '=' or '!' is rejected: equality is
// '==' and inequality '!=', and NOT is spelled out as a word rather than '!'.
func lexOp(src string, i int) (op string, n int, err error) {
	if i+1 < len(src) {
		switch src[i : i+2] {
		case "<=", ">=", "==", "!=":
			return src[i : i+2], 2, nil
		}
	}
	switch src[i] {
	case '<', '>':
		return src[i : i+1], 1, nil
	}
	return "", 0, fmt.Errorf("signal: %q at position %d is not a comparison operator (use ==, !=, <, >, <= or >=)", string(src[i]), i)
}

// lexNumber reads a run of digits and dots and parses it as a number.
func lexNumber(src string, i int) (val float64, n int, err error) {
	start := i
	for i < len(src) && (src[i] >= '0' && src[i] <= '9' || src[i] == '.') {
		i++
	}
	text := src[start:i]
	val, err = strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("signal: %q at position %d is not a number", text, start)
	}
	return val, i - start, nil
}

// lexWord reads a run of letters and digits: a keyword or a candle reference.
func lexWord(src string, i int) (word string, n int) {
	start := i
	for i < len(src) && (isLetter(src[i]) || src[i] >= '0' && src[i] <= '9') {
		i++
	}
	return src[start:i], i - start
}

// classify decides whether a word is a keyword or a candle reference. A
// reference is a field letter (O, C, H, L) followed by a 1-based candle number,
// so H2 is the second candle's high; anything else is rejected by name.
func classify(word string, pos int) (token, error) {
	switch word {
	case "AND":
		return token{kind: tokAnd, pos: pos}, nil
	case "OR":
		return token{kind: tokOr, pos: pos}, nil
	case "NOT":
		return token{kind: tokNot, pos: pos}, nil
	}

	if len(word) >= 2 && isField(word[0]) {
		if index, err := strconv.Atoi(word[1:]); err == nil && index >= 1 {
			return token{kind: tokRef, ref: ref{field: word[0], index: index, text: word}, pos: pos}, nil
		}
	}
	return token{}, fmt.Errorf("signal: %q at position %d is not a value (like H1) or a keyword (AND, OR, NOT)", word, pos)
}

func isLetter(c byte) bool { return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' }

// isField reports whether c names one of a candle's four prices.
func isField(c byte) bool { return c == 'O' || c == 'C' || c == 'H' || c == 'L' }

// parser is a recursive-descent parser over the lexed tokens. Precedence runs
// from OR (loosest) through AND to NOT, with comparisons and bracketed groups at
// the bottom.
type parser struct {
	toks []token
	pos  int
	body int
}

func (p *parser) peek() token { return p.toks[p.pos] }

func (p *parser) next() token {
	t := p.toks[p.pos]
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return t
}

func (p *parser) parseOr() (expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokOr {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = orExpr{left, right}
	}
	return left, nil
}

func (p *parser) parseAnd() (expr, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokAnd {
		p.next()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = andExpr{left, right}
	}
	return left, nil
}

func (p *parser) parseNot() (expr, error) {
	if p.peek().kind == tokNot {
		p.next()
		child, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return notExpr{child}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (expr, error) {
	switch tok := p.peek(); tok.kind {
	case tokLParen:
		p.next()
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tokRParen {
			return nil, fmt.Errorf("signal: missing ')' at position %d", p.peek().pos)
		}
		p.next()
		return inner, nil
	case tokRef, tokNumber:
		return p.parseComparison()
	default:
		return nil, fmt.Errorf("signal: expected a value or '(' at position %d, got %s", tok.pos, describe(tok))
	}
}

func (p *parser) parseComparison() (expr, error) {
	left, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokOp {
		return nil, fmt.Errorf("signal: expected a comparison operator at position %d, got %s", p.peek().pos, describe(p.peek()))
	}
	op := p.next().text

	right, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	return comparison{left: left, op: op, right: right}, nil
}

func (p *parser) parseOperand() (operand, error) {
	switch tok := p.next(); tok.kind {
	case tokRef:
		if tok.ref.index > p.body {
			return nil, fmt.Errorf("signal: %s references candle %d, but the body is only %d long", tok.ref.text, tok.ref.index, p.body)
		}
		return tok.ref, nil
	case tokNumber:
		return number(tok.num), nil
	default:
		return nil, fmt.Errorf("signal: expected a value at position %d, got %s", tok.pos, describe(tok))
	}
}
