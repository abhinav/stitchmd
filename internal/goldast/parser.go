package goldast

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/stitchmd/internal/pos"
)

// File is a parsed Markdown file.
type File struct {
	// AST is the parsed Markdown file.
	AST ast.Node

	// Source is the original Markdown source.
	Source []byte

	// Info holds position information about the file.
	Info *pos.Info
}

// Position turns the given Pos into a Position
// using this file's position information.
func (f *File) Position(offset int) pos.Position {
	return f.Info.Position(offset)
}

// Parse parses a Markdown document using the provided Goldmark parser.
func Parse(p parser.Parser, filename string, src []byte, opts ...parser.ParseOption) *File {
	n := p.Parse(text.NewReader(src), opts...)

	return &File{
		AST:    n,
		Source: src,
		Info:   pos.FromContent(filename, src),
	}
}

// DefaultParser returns the default Goldmark parser we should use in the
// application.
func DefaultParser() parser.Parser {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	).Parser()
}
