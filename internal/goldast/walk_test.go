package goldast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
)

func TestWalk(t *testing.T) {
	t.Parallel()

	parser := goldmark.New().Parser()

	f, err := Parse(parser, "foo.md", []byte("\n# Foo\nhello world."))
	require.NoError(t, err)

	var nodes []*Any
	err = Walk(f.AST, func(n *Any, enter bool) (ast.WalkStatus, error) {
		if enter {
			nodes = append(nodes, n)
		}
		return ast.WalkContinue, nil
	})
	require.NoError(t, err)

	posc := f.Positioner
	require.Len(t, nodes, 5)
	if n := nodes[0]; assert.IsType(t, new(ast.Document), n.Node) {
		assert.Equal(t, "foo.md:1:1", posc.Position(n.Pos()).String(), "document")
	}
	if n := nodes[1]; assert.IsType(t, new(ast.Heading), n.Node) {
		assert.Equal(t, "foo.md:2:3", posc.Position(n.Pos()).String(), "heading")
		assert.Equal(t, "Foo", string(n.Node.Text(f.Source)))
	}
	if n := nodes[2]; assert.IsType(t, new(ast.Text), n.Node) {
		assert.Equal(t, "foo.md:2:3", posc.Position(n.Pos()).String(), "heading text")
		assert.Equal(t, "Foo", string(n.Node.Text(f.Source)))
	}
	if n := nodes[3]; assert.IsType(t, new(ast.Paragraph), n.Node) {
		assert.Equal(t, "foo.md:3:1", posc.Position(n.Pos()).String(), "paragraph")
	}
	if n := nodes[4]; assert.IsType(t, new(ast.Text), n.Node) {
		assert.Equal(t, "foo.md:3:1", posc.Position(n.Pos()).String(), "paragraph text")
		assert.Equal(t, "hello world.", string(n.Node.Text(f.Source)))
	}
}
