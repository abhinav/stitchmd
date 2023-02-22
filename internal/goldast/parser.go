package goldast

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/stitchmd/internal/pos"
)

// File is a parsed Markdown file.
type File struct {
	// AST is the parsed Markdown file.
	AST *Any

	// Source is the original Markdown source.
	Source []byte

	// Info holds position information about the file.
	Info *pos.Info

	// Pos is the starting position for this document.
	Pos pos.Pos
}

// Position turns the given Pos into a Position
// using this file's position information.
func (f *File) Position(p pos.Pos) pos.Position {
	return f.Info.Position(p)
}

// Parse parses a Markdown document using the provided Goldmark parser.
func Parse(p parser.Parser, filename string, src []byte, opts ...parser.ParseOption) (*File, error) {
	n, err := Wrap(p.Parse(text.NewReader(src), opts...))
	if err != nil {
		// This is not typically possible because we'll always have
		// position information for the top-level object.
		return nil, err
	}

	return &File{
		AST:    n,
		Source: src,
		Pos:    n.Pos(),
		Info:   pos.FromContent(filename, src),
	}, nil
}

// DefaultParser returns the default Goldmark parser we should use in the
// application.
func DefaultParser() parser.Parser {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	).Parser()
}
