package main

import (
	"strings"
	"unicode"
)

// ref https://github.com/Flet/github-slugger/blob/3461c4350868329c8530904d170358bca1d31448/script/generate-regex.js
var _removeRanges = []*unicode.RangeTable{
	unicode.No, // Number, other
	unicode.Pd, // Punctuation, dash
	unicode.Pe, // Punctuation, close
	unicode.Pi, // Punctuation, initial quote
	unicode.Pf, // Punctuation, final quote
	unicode.Po, // Punctuation, other
	unicode.Ps, // Punctuation, open
	unicode.S,  // Symbol
	unicode.Cc, // Control
	unicode.Co, // Private use
	unicode.Cf, // Format
	unicode.Z,  // Separator
}

func shouldRemove(r rune) bool {
	switch r {
	case ' ', '-':
		return false
	}
	if unicode.Is(unicode.Other_Alphabetic, r) {
		return false
	}
	return unicode.In(r, _removeRanges...)
}

// slugify turns a heading title into a slug.
// The slug is intended to match the GitHub heading slugging algorithm.
func slugify(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if shouldRemove(r) {
			continue
		}
		switch r {
		case ' ', '-':
			sb.WriteRune('-')
		default:
			sb.WriteRune(unicode.ToLower(r))
		}
	}
	return sb.String()
}
