package scan

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type tokKind uint8

const (
	tIdent tokKind = iota
	tLifetime
	tStr
	tChar
	tNum
	tPunct
)

type token struct {
	kind tokKind
	text string
	line int
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentCont(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func lex(src string) []token {
	var toks []token
	line := 1
	i := 0
	n := len(src)
	for i < n {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case c == '/' && i+1 < n && src[i+1] == '/':
			for i < n && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && src[i+1] == '*':
			depth := 0
			for i < n {
				if src[i] == '/' && i+1 < n && src[i+1] == '*' {
					depth++
					i += 2
					continue
				}
				if src[i] == '*' && i+1 < n && src[i+1] == '/' {
					depth--
					i += 2
					if depth == 0 {
						break
					}
					continue
				}
				if src[i] == '\n' {
					line++
				}
				i++
			}
		case c == '"':
			start := line
			s, j, nl := readString(src, i+1)
			toks = append(toks, token{tStr, s, start})
			line += nl
			i = j
		case (c == 'r' || c == 'b' || c == 'c') && rawStart(src, i) > 0:
			start := line
			j := i + rawStart(src, i)
			hashes := 0
			for j < n && src[j] == '#' {
				hashes++
				j++
			}
			j++
			end := strings.Index(src[j:], "\""+strings.Repeat("#", hashes))
			if end < 0 {
				end = n - j
			}
			s := src[j : j+end]
			line += strings.Count(s, "\n")
			toks = append(toks, token{tStr, s, start})
			i = j + end + 1 + hashes
		case (c == 'b' || c == 'c') && i+1 < n && src[i+1] == '"':
			start := line
			s, j, nl := readString(src, i+2)
			toks = append(toks, token{tStr, s, start})
			line += nl
			i = j
		case c == 'b' && i+1 < n && src[i+1] == '\'':
			j := readChar(src, i+2)
			toks = append(toks, token{tChar, src[i:j], line})
			i = j
		case c == '\'':
			if j, ok := charLit(src, i); ok {
				toks = append(toks, token{tChar, src[i:j], line})
				i = j
				continue
			}
			j := i + 1
			for j < n {
				r, sz := utf8.DecodeRuneInString(src[j:])
				if !isIdentCont(r) {
					break
				}
				j += sz
			}
			toks = append(toks, token{tLifetime, src[i:j], line})
			i = j
		case c >= '0' && c <= '9':
			j := i
			for j < n && (isIdentCont(rune(src[j])) || (src[j] == '.' && j+1 < n && src[j+1] >= '0' && src[j+1] <= '9')) {
				j++
			}
			toks = append(toks, token{tNum, src[i:j], line})
			i = j
		default:
			r, sz := utf8.DecodeRuneInString(src[i:])
			if isIdentStart(r) {
				j := i
				for j < n {
					r2, sz2 := utf8.DecodeRuneInString(src[j:])
					if !isIdentCont(r2) {
						break
					}
					j += sz2
				}
				text := src[i:j]
				if text == "r" && j < n && src[j] == '#' && j+1 < n && isIdentStart(rune(src[j+1])) {
					k := j + 1
					for k < n && isIdentCont(rune(src[k])) {
						k++
					}
					text = src[j+1 : k]
					j = k
				}
				toks = append(toks, token{tIdent, text, line})
				i = j
				continue
			}
			toks = append(toks, token{tPunct, src[i : i+sz], line})
			i += sz
		}
	}
	return toks
}

func rawStart(src string, i int) int {
	j := i
	if src[j] == 'b' || src[j] == 'c' {
		j++
	}
	if j >= len(src) || src[j] != 'r' {
		return 0
	}
	j++
	k := j
	for k < len(src) && src[k] == '#' {
		k++
	}
	if k < len(src) && src[k] == '"' {
		if i > 0 && isIdentCont(rune(src[i-1])) {
			return 0
		}
		return j - i
	}
	return 0
}

func readString(src string, i int) (string, int, int) {
	var b strings.Builder
	nl := 0
	for i < len(src) {
		c := src[i]
		if c == '\\' && i+1 < len(src) {
			if src[i+1] == '\n' {
				nl++
			}
			b.WriteByte(c)
			b.WriteByte(src[i+1])
			i += 2
			continue
		}
		if c == '"' {
			return b.String(), i + 1, nl
		}
		if c == '\n' {
			nl++
		}
		b.WriteByte(c)
		i++
	}
	return b.String(), i, nl
}

func readChar(src string, i int) int {
	for i < len(src) {
		if src[i] == '\\' {
			i += 2
			continue
		}
		if src[i] == '\'' {
			return i + 1
		}
		i++
	}
	return i
}

func charLit(src string, i int) (int, bool) {
	j := i + 1
	if j >= len(src) {
		return 0, false
	}
	if src[j] == '\\' {
		return readChar(src, j), true
	}
	_, sz := utf8.DecodeRuneInString(src[j:])
	if j+sz < len(src) && src[j+sz] == '\'' {
		return j + sz + 1, true
	}
	return 0, false
}
