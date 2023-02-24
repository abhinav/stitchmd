// Package goldast provides utilities for working with Goldmark ASTs.
//
// It provides a position tracking mechanism via the [Info] and [Position]
// types, so that we can report errors with line and column numbers.
package goldast

import (
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// File is a parsed Markdown file.
type File struct {
	// AST is the parsed Markdown file.
	AST ast.Node

	// Source is the original Markdown source.
	Source []byte

	// Info holds position information about the file.
	Info *Info
}

// Position turns the given Pos into a Position
// using this file's position information.
func (f *File) Position(offset int) Position {
	return f.Info.Position(offset)
}

// Parse parses a Markdown document using the provided Goldmark parser.
func Parse(p parser.Parser, filename string, src []byte, opts ...parser.ParseOption) *File {
	n := p.Parse(text.NewReader(src), opts...)

	return &File{
		AST:    n,
		Source: src,
		Info:   infoFromContent(filename, src),
	}
}

// DefaultParser returns the default Goldmark parser we should use in the
// application.
func DefaultParser() parser.Parser {
	return goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			meta.New(),
		),
	).Parser()
}
