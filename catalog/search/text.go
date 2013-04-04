package search

import (
	"strings"
	"unicode"
)

// fold changes every rune in s to its least equivalent folded case, according
// to unicode.SimpleFold and returns s.
func fold(s []rune) []rune {
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			// the only characters in ASCII that need folding are lowercase
			s[i] = r - 'a' + 'A'
		case r < 128:
			// do nothing
		default:
			rr := unicode.SimpleFold(r)
			for rr > r {
				rr = unicode.SimpleFold(rr)
			}
			s[i] = rr
		}
	}
	return s
}

// foldString allocates a new rune slice for s and calls fold.
func foldString(s string) string {
	return string(fold([]rune(s)))
}

func tokenize(s []rune) [][]rune {
	return runeFieldsFunc(s, isTokenSep)
}

func tokenizeString(s string) []string {
	return strings.FieldsFunc(s, isTokenSep)
}

var tokenizeRanges = []*unicode.RangeTable{unicode.Letter, unicode.Number}

func isTokenSep(r rune) bool {
	return !unicode.IsOneOf(tokenizeRanges, r)
}

func stripTokenSep(r []rune) []rune {
	for i := len(r) - 1; i >= 0; i-- {
		if isTokenSep(r[i]) {
			copy(r[i:], r[i+1:])
			r = r[:len(r)-1]
		}
	}
	return r
}

// runeFieldsFunc splits the rune slice s at each run of Unicode code points c
// satisfying f(c) and returns an array of slices of s. If all code points in s
// satisfy f(c) or the string is empty, an empty slice is returned.
func runeFieldsFunc(s []rune, f func(rune) bool) [][]rune {
	// borrowed from strings.FieldsFunc in standard library

	n := 0
	inField := false
	for _, rune := range s {
		wasInField := inField
		inField = !f(rune)
		if inField && !wasInField {
			n++
		}
	}

	a := make([][]rune, n)
	na := 0
	fieldStart := -1
	for i, rune := range s {
		if f(rune) {
			if fieldStart >= 0 {
				a[na] = s[fieldStart:i]
				na++
				fieldStart = -1
			}
		} else if fieldStart == -1 {
			fieldStart = i
		}
	}
	if fieldStart >= 0 {
		a[na] = s[fieldStart:]
	}

	return a
}
