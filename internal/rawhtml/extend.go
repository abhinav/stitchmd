package rawhtml

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
)

// Extender installs the rawhtml extension into a Goldmark Markdown.
type Extender struct{}

var _ goldmark.Extender = (*Extender)(nil)

// Extend extends the given Goldmark Markdown with the rawhtml extension.
//
// This implements the goldmark.Extender interface.
func (e *Extender) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&Transformer{}, 10),
		),
	)
}
