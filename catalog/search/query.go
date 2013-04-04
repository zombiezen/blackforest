package search

import (
	"strconv"
	"strings"
)

// FindTerms identifies the byte index pairs of the terms in query q.
func FindTerms(q string, text string) []int {
	ast, err := parseQuery(q)
	if err != nil {
		return []int{}
	}
	terms := extractTerms(ast)
	if len(terms) == 0 {
		return []int{}
	}
	foldedTerms := make([][]rune, len(terms))
	for i, term := range terms {
		foldedTerms[i] = fold([]rune(term))
	}

	runes := fold([]rune(text))
	pairs := make([]int, 0)
	tokenStart := -1
	for i, r := range runes {
		if !isTokenizeRune(r) {
			if tokenStart >= 0 {
				tok := runes[tokenStart:i]
				for _, term := range foldedTerms {
					if runesEq(tok, term) {
						pairs = append(pairs, tokenStart, i)
						break
					}
				}
				tokenStart = -1
			}
		} else if tokenStart == -1 {
			tokenStart = i
		}
	}
	if tokenStart >= 0 {
		tok := runes[tokenStart:len(runes)]
		for _, term := range foldedTerms {
			if runesEq(tok, term) {
				pairs = append(pairs, tokenStart, len(runes))
				break
			}
		}
	}
	return pairs
}

func runesEq(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type queryAST interface {
	String() string
	isQueryAST()
}

func extractTerms(ast queryAST) []string {
	switch ast := ast.(type) {
	case token:
		return []string{string(ast)}
	case queryAnd:
		var terms []string
		for _, q := range ast {
			terms = append(terms, extractTerms(q)...)
		}
		return terms
	case queryOr:
		var terms []string
		for _, q := range ast {
			terms = append(terms, extractTerms(q)...)
		}
		return terms
	}
	return []string{}
}

func parseQuery(q string) (ast queryAST, err error) {
	p := queryParser{
		items: lexQuery(q),
	}
	return p.parseQuery(), nil
}

type queryParser struct {
	items []item
	pos   int
}

func (p *queryParser) next() item {
	item := p.peek()
	if p.pos < len(p.items) {
		p.pos++
	}
	return item
}

func (p *queryParser) peek() item {
	if p.pos >= len(p.items) {
		return item{}
	}
	return p.items[p.pos]
}

func (p *queryParser) backup() {
	p.pos--
}

func (p *queryParser) parseQuery() queryAST {
	qa := make(queryAnd, 0, 1)
	for {
		c := p.parseChoice()
		if c == nil {
			break
		}
		qa = append(qa, c)
	}
	if len(qa) == 0 {
		return nil
	} else if len(qa) == 1 {
		return qa[0]
	}
	return qa
}

func (p *queryParser) parseChoice() queryAST {
	qo := make(queryOr, 1)
	qo[0] = p.parsePrimary()
	if qo[0] == nil {
		return nil
	}
	for p.peek().kind == orItem {
		p.next()
		primary := p.parsePrimary()
		if primary == nil {
			return nil
		}
		qo = append(qo, primary)
	}
	if len(qo) == 1 {
		return qo[0]
	}
	return qo
}

func (p *queryParser) parsePrimary() queryAST {
	if p.peek().kind == notItem {
		p.next()
		atom := p.parseAtom()
		if atom == nil {
			return nil
		}
		return queryNot{atom}
	}
	return p.parseAtom()
}

func (p *queryParser) parseAtom() queryAST {
	switch item := p.next(); item.kind {
	default:
		p.backup()
		return nil
	case tokenItem:
		return token(item.value)
	case tagItem:
		item = p.next()
		if item.kind != tokenItem {
			return nil
		}
		return tagAtom(item.value)
	case lparenItem:
		q := p.parseQuery()
		if q == nil {
			return nil
		}
		if p.next().kind != rparenItem {
			return nil
		}
		return q
	}
	panic("unreachable")
}

type queryAnd []queryAST

func (queryAnd) isQueryAST() {}

func (q queryAnd) String() string {
	parts := make([]string, len(q))
	for i := range q {
		parts[i] = q[i].String()
	}
	return "(" + strings.Join(parts, ") (") + ")"
}

type queryOr []queryAST

func (queryOr) isQueryAST() {}

func (q queryOr) String() string {
	parts := make([]string, len(q))
	for i := range q {
		parts[i] = q[i].String()
	}
	return "(" + strings.Join(parts, ") OR (") + ")"
}

type queryNot struct {
	ast queryAST
}

func (queryNot) isQueryAST() {}

func (q queryNot) String() string {
	return "-(" + q.ast.String() + ")"
}

type token string

func (token) isQueryAST() {}

func (t token) String() string {
	return string(t)
}

func (t token) GoString() string {
	return "search.token(" + strconv.Quote(string(t)) + ")"
}

type tagAtom string

func (tagAtom) isQueryAST() {}

func (t tagAtom) String() string {
	return tagPrefix + string(t)
}

func (t tagAtom) GoString() string {
	return "search.tagAtom(" + strconv.Quote(string(t)) + ")"
}

type parseError struct {
	Input string
	Pos   int
	Msg   string
}

func (e *parseError) Error() string {
	return e.Msg
}
