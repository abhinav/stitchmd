package rawhtml

import (
	"github.com/yuin/goldmark/v2"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/util"
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
