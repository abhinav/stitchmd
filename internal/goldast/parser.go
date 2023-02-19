package goldast

import (
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/mdreduce/internal/pos"
)

// File is a parsed Markdown file.
type File struct {
	// Name of the file, as passed to Parse.
	Name string

	// AST is the parsed Markdown file.
	AST *Any

	// Source is the original Markdown source.
	Source []byte

	// Positioner maps Pos values in the AST to file positions.
	Positioner *pos.Converter

	// Pos is the starting position for this document.
	Pos pos.Pos
}

// Pos turns the given Pos into a Position
// using this file's position information.
func (f *File) Position(p pos.Pos) pos.Position {
	return f.Positioner.Position(p)
}

// Parse parses a Markdown document using the provided Goldmark parser.
func Parse(p parser.Parser, filename string, src []byte, opts ...parser.ParseOption) (*File, error) {
	n, err := Wrap(p.Parse(text.NewReader(src), opts...))
	if err != nil {
		return nil, err
	}

	return &File{
		Name:       filename,
		AST:        n,
		Source:     src,
		Pos:        n.Pos(),
		Positioner: pos.FromContent(filename, src),
	}, nil
}
