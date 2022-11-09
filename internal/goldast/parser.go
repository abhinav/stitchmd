package goldast

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Parse parses a Markdown document using the provided Goldmark parser.
func Parse(p parser.Parser, src []byte, opts ...parser.ParseOption) (*Node[ast.Node], error) {
	return Wrap(p.Parse(text.NewReader(src), opts...))
}
