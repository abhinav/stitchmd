package stitch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark/ast"
)

func TestLinkItem_accessors(t *testing.T) {
	t.Parallel()

	link := ast.NewLink()
	item := LinkItem{
		Text:   "foo",
		Target: "foo.md",
		Depth:  3,
		AST:    link,
	}

	assert.Equal(t, 3, item.ItemDepth())
	assert.True(t, item.Node() == link)
}

func TestTextItem_accessors(t *testing.T) {
	t.Parallel()

	node := ast.NewText()
	item := TextItem{
		Text:  "foo",
		Depth: 3,
		AST:   node,
	}

	assert.Equal(t, 3, item.ItemDepth())
	assert.True(t, item.Node() == node)
}
