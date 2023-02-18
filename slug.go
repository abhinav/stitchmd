package main

import (
	"strings"
	"unicode"
)

// ref https://github.com/Flet/github-slugger/blob/3461c4350868329c8530904d170358bca1d31448/script/generate-regex.js
var _slugRemoveRanges = []*unicode.RangeTable{
	unicode.No, // Number, other

	unicode.Pd, // Punctuation, dash
	unicode.Pe, // Punctuation, close
	unicode.Pi, // Punctuation, initial quote
	unicode.Pf, // Punctuation, final quote
	unicode.Po, // Punctuation, other
	unicode.Ps, // Punctuation, open
	// The group 'Punctuation, connector'
	// is specifically not included here.
	// From experimentation, and from the source above,
	// GitHub does not remove those from titles when IDs.

	unicode.S,  // Symbol
	unicode.Cc, // Control
	unicode.Co, // Private use
	unicode.Cf, // Format
	unicode.Z,  // Separator
}

// titleSlug turns a heading title into a slug.
// The slug is intended to match the GitHub heading slugging algorithm.
func titleSlug(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r == ' ', r == '-':
			sb.WriteRune('-')
			continue

		case unicode.Is(unicode.Other_Alphabetic, r):
			// do nothing

		case unicode.In(r, _slugRemoveRanges...):
			// Should be removed.
			continue
		}

		sb.WriteRune(unicode.ToLower(r))
	}
	return sb.String()
}
