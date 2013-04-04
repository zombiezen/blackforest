package search

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Inspired by the text/template/parse lexer

const eof = -1

type itemKind int

const (
	invalidItem itemKind = iota
	eofItem
	termItem
	orItem
	notItem
	tagItem
	lparenItem
	rparenItem
)

type item struct {
	kind  itemKind
	value string
}

func (item item) String() string {
	return item.value
}

type stateFn func(*queryLexer) stateFn

type queryLexer struct {
	input string
	pos   int
	width int
	start int
	items chan item
	state stateFn
}

func lexQuery(query string) []item {
	l := queryLexer{
		input: query,
		items: make(chan item, 2),
		state: lexDefault,
	}
	items := make([]item, 0)
	for {
		item := l.nextItem()
		items = append(items, item)
		if item.kind == eofItem || item.kind == invalidItem {
			break
		}
	}
	return items
}

func (l *queryLexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *queryLexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *queryLexer) backup() {
	l.pos -= l.width
}

func (l *queryLexer) emit(k itemKind) {
	l.items <- item{kind: k, value: l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *queryLexer) ignore() {
	l.start = l.pos
}

func (l *queryLexer) nextItem() item {
	for {
		select {
		case item := <-l.items:
			return item
		default:
			l.state = l.state(l)
		}
	}
	panic("not reachable")
}

const (
	tagPrefix  = "tag:"
	orOperator = "OR"
)

func lexDefault(l *queryLexer) stateFn {
	// skip leading whitespace
	for unicode.IsSpace(l.next()) {
	}
	l.backup()
	l.ignore()

	switch c := l.next(); {
	case c == eof:
		l.emit(eofItem)
		return nil
	case c == '(':
		l.emit(lparenItem)
		return lexDefault
	case c == ')':
		l.emit(rparenItem)
		return lexDefault
	case c == '-':
		l.emit(notItem)
		return lexDefault
	}

	l.backup()
	return lexTerm
}

func lexTerm(l *queryLexer) stateFn {
	for {
		r := l.next()
		if r == eof || unicode.IsSpace(r) || r == '(' || r == ')' {
			break
		}
	}
	l.backup()

	if strings.HasPrefix(l.input[l.start:], tagPrefix) {
		end := l.pos
		l.pos = l.start + len(tagPrefix)
		l.emit(tagItem)
		l.pos = end
		l.emit(termItem)
	} else if l.input[l.start:l.pos] == orOperator {
		l.emit(orItem)
	} else {
		l.emit(termItem)
	}
	return lexDefault
}
