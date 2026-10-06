package jevtok

import (
	"unicode"
	"unicode/utf8"
)

// Qwen3.5 split, ported from OMP scan/qwen.rs (leftmost-first, greedy):
//
//	(?i:'s|'t|'re|'ve|'m|'ll|'d)|[^\r\n\p{L}\p{N}]?[\p{L}\p{M}]+|\p{N}| ?[^\s\p{L}\p{M}\p{N}]+[\r\n]*|\s*[\r\n]+|\s+(?!\S)|\s+
//
// Case-insensitivity is the scanner's simple fold, including U+017F for 's.

func isLetter(r rune) bool {
	if r < 128 {
		return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
	}
	return unicode.IsLetter(r)
}

func isNumber(r rune) bool {
	if r < 128 {
		return r >= '0' && r <= '9'
	}
	return unicode.IsNumber(r)
}

func isMark(r rune) bool {
	return r >= 128 && unicode.IsMark(r)
}

func isWS(r rune) bool { return unicode.IsSpace(r) }

func decodeAt(s string, i int) (rune, int, bool) {
	if i >= len(s) {
		return 0, 0, false
	}
	r, n := utf8.DecodeRuneInString(s[i:])
	return r, n, true
}

func contractionEnd(s string, pos int) int {
	c0, n0, ok := decodeAt(s, pos)
	if !ok || c0 != '\'' {
		return pos
	}
	c1, n1, ok := decodeAt(s, pos+n0)
	if !ok {
		return pos
	}
	one := pos + n0 + n1
	switch c1 {
	case 's', 'S', '\u017f', 't', 'T', 'm', 'M', 'd', 'D':
		return one
	case 'r', 'R', 'v', 'V':
		if c2, n2, ok := decodeAt(s, one); ok && (c2 == 'e' || c2 == 'E') {
			return one + n2
		}
	case 'l', 'L':
		if c2, n2, ok := decodeAt(s, one); ok && (c2 == 'l' || c2 == 'L') {
			return one + n2
		}
	}
	return pos
}

func lmRunEnd(s string, pos int) int {
	i := pos
	for {
		c, n, ok := decodeAt(s, i)
		if !ok || (!isLetter(c) && !isMark(c)) {
			return i
		}
		i += n
	}
}

// word is `[^\r\n\p{L}\p{N}]?[\p{L}\p{M}]+`. A mark is both prefix- and
// run-eligible, so a lone mark backtracks off the greedy prefix.
func word(s string, pos int) (int, bool) {
	c, n, ok := decodeAt(s, pos)
	if !ok {
		return 0, false
	}
	if c != '\r' && c != '\n' && !isLetter(c) && !isNumber(c) {
		e := lmRunEnd(s, pos+n)
		if e > pos+n {
			return e, true
		}
		if isMark(c) {
			return pos + n, true
		}
		return 0, false
	}
	e := lmRunEnd(s, pos)
	if e > pos {
		return e, true
	}
	return 0, false
}

// punct is ` ?[^\s\p{L}\p{M}\p{N}]+[\r\n]*`.
func punct(s string, pos int) (int, bool) {
	c, n, ok := decodeAt(s, pos)
	if !ok {
		return 0, false
	}
	start := pos
	if c == ' ' {
		start = pos + n
	}
	i := start
	for {
		c, n, ok := decodeAt(s, i)
		if !ok || isWS(c) || isLetter(c) || isMark(c) || isNumber(c) {
			break
		}
		i += n
	}
	if i == start {
		return 0, false
	}
	for {
		c, n, ok := decodeAt(s, i)
		if !ok || (c != '\r' && c != '\n') {
			break
		}
		i += n
	}
	return i, true
}

// wsEnd is `\s*[\r\n]+|\s+(?!\S)|\s+`. A newline run stops at its last
// newline; otherwise a non-final run gives back one codepoint for (?!\S).
func wsEnd(s string, pos int) (int, bool) {
	i := pos
	lastNL := -1
	lastLen := 0
	cps := 0
	for {
		c, n, ok := decodeAt(s, i)
		if !ok || !isWS(c) {
			break
		}
		i += n
		if c == '\r' || c == '\n' {
			lastNL = i
		}
		lastLen = n
		cps++
	}
	if cps == 0 {
		return 0, false
	}
	if lastNL >= 0 {
		return lastNL, true
	}
	if i == len(s) {
		return i, true
	}
	if cps > 1 {
		return i - lastLen, true
	}
	return i, true
}

func nextPiece(s string, pos int) int {
	if e := contractionEnd(s, pos); e > pos {
		return e
	}
	if e, ok := word(s, pos); ok {
		return e
	}
	c, n, ok := decodeAt(s, pos)
	if !ok {
		return pos + 1
	}
	if isNumber(c) {
		return pos + n
	}
	if e, ok := punct(s, pos); ok {
		return e
	}
	if e, ok := wsEnd(s, pos); ok {
		return e
	}
	return pos + n
}

func forEachPiece(s string, f func(string)) {
	for pos := 0; pos < len(s); {
		end := nextPiece(s, pos)
		if end <= pos || end > len(s) {
			panic("jevtok: scanner did not advance")
		}
		f(s[pos:end])
		pos = end
	}
}
