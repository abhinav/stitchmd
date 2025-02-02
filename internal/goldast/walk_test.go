package goldast

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
)

func TestWalk(t *testing.T) {
	t.Parallel()

	parser := goldmark.New().Parser()

	f := Parse(parser, "foo.md", []byte("\n# Foo\nhello world."))

	var nodes []ast.Node
	err := Walk(f.AST, func(n ast.Node) error {
		nodes = append(nodes, n)
		return nil
	})
	require.NoError(t, err)

	posc := f.Info
	require.Len(t, nodes, 5)
	if n := nodes[0]; assert.IsType(t, new(ast.Document), n) {
		assert.Equal(t, "foo.md:1:1", posc.Position(OffsetOf(n)).String(), "document")
	}
	if n := nodes[1]; assert.IsType(t, new(ast.Heading), n) {
		assert.Equal(t, "foo.md:2:3", posc.Position(OffsetOf(n)).String(), "heading")
		assert.Equal(t, "Foo", string(Text(f.Source, n)))
	}
	if n := nodes[2]; assert.IsType(t, new(ast.Text), n) {
		assert.Equal(t, "foo.md:2:3", posc.Position(OffsetOf(n)).String(), "heading text")
		assert.Equal(t, "Foo", string(Text(f.Source, n)))
	}
	if n := nodes[3]; assert.IsType(t, new(ast.Paragraph), n) {
		assert.Equal(t, "foo.md:3:1", posc.Position(OffsetOf(n)).String(), "paragraph")
	}
	if n := nodes[4]; assert.IsType(t, new(ast.Text), n) {
		assert.Equal(t, "foo.md:3:1", posc.Position(OffsetOf(n)).String(), "paragraph text")
		assert.Equal(t, "hello world.", string(Text(f.Source, n)))
	}
}

func TestWalk_error(t *testing.T) {
	t.Parallel()

	parser := goldmark.New().Parser()

	giveErr := errors.New("great sadness")
	f := Parse(parser, "foo.md", []byte("# Foo"))

	var nodes []ast.Node
	err := Walk(f.AST, func(n ast.Node) error {
		if len(nodes) > 0 {
			return giveErr
		}
		nodes = append(nodes, n)
		return nil
	})
	require.ErrorIs(t, err, giveErr)

	require.Len(t, nodes, 1)
	assert.IsType(t, new(ast.Document), nodes[0])
}
