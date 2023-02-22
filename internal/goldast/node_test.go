package goldast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/stitchmd/internal/pos"
)

func TestWrap(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		_, err := Wrap[ast.Node](nil)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		segs := text.NewSegments()
		segs.Append(text.NewSegment(5, 10))

		n := ast.NewParagraph()
		n.SetLines(segs)

		w, err := Wrap(n)
		require.NoError(t, err)
		assert.Equal(t, n, w.Node)
		assert.Equal(t, pos.Pos(5), w.Pos())
	})

	// Blocks with no lines can't be wrapped.
	t.Run("failure/empty block", func(t *testing.T) {
		t.Parallel()

		n := ast.NewParagraph()
		n.SetLines(text.NewSegments())

		_, err := Wrap(n)
		require.Error(t, err)
	})

	// Inline blocks don't have lines associated with them.
	t.Run("failure/inline", func(t *testing.T) {
		t.Parallel()

		n := ast.NewString([]byte("hello"))
		_, err := Wrap(n)
		require.Error(t, err)
	})
}

func TestCast(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		n := &Any{Node: ast.NewString([]byte("hello"))}
		c, ok := Cast[*ast.String](n.AsAny())
		require.True(t, ok)
		assert.Equal(t, n.Node, c.Node)
	})

	t.Run("failure", func(t *testing.T) {
		t.Parallel()

		n := &Any{Node: ast.NewString([]byte("hello"))}
		_, ok := Cast[*ast.Heading](n.AsAny())
		require.False(t, ok)
	})
}

func TestMustCast(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		n := &Any{Node: ast.NewString([]byte("hello"))}
		c := MustCast[*ast.String](n.AsAny())
		assert.Equal(t, n.Node, c.Node)
	})

	t.Run("failure", func(t *testing.T) {
		t.Parallel()

		n := &Any{Node: ast.NewString([]byte("hello"))}
		require.Panics(t, func() {
			MustCast[*ast.Heading](n.AsAny())
		})
	})
}

func TestNode_AsAny(t *testing.T) {
	t.Parallel()

	strn := Node[*ast.String]{
		Node:   ast.NewString([]byte("hello")),
		pos:    42,
		parent: &Any{Node: ast.NewParagraph()},
	}

	n := strn.AsAny()
	assert.Equal(t, strn.Node, n.Node)
	assert.Equal(t, strn.Pos(), n.Pos())
	assert.Equal(t, strn.parent, n.parent)

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		var n *Node[*ast.String]
		assert.Nil(t, n.AsAny())
	})
}

func TestNode_Pos_nil(t *testing.T) {
	t.Parallel()

	var n *Any
	assert.Equal(t, pos.Pos(0), n.Pos())
}

func TestNode_ChildCount_nil(t *testing.T) {
	t.Parallel()

	var n *Any
	assert.Equal(t, 0, n.ChildCount())
}

func TestNode_Relations(t *testing.T) {
	t.Parallel()

	//               0           1
	//               012345 6 789012 3 45678
	doc := parse(t, "# Foo\n\nhello\n\nworld")

	assert.Equal(t, 3, doc.ChildCount())

	h1 := doc.FirstChild()
	if assert.Equal(t, ast.KindHeading, h1.Kind(), "heading") {
		assert.Equal(t, doc, h1.Parent())
		assert.Equal(t, pos.Pos(2), h1.Pos())
	}

	p1 := h1.NextSibling()
	if assert.Equal(t, ast.KindParagraph, p1.Kind(), "paragraph") {
		assert.Equal(t, doc, p1.Parent())
		assert.Equal(t, h1, p1.PreviousSibling())
		assert.Equal(t, pos.Pos(7), p1.Pos())
	}

	p2 := p1.NextSibling()
	if assert.Equal(t, ast.KindParagraph, p2.Kind(), "paragraph") {
		assert.Equal(t, doc, p2.Parent())
		assert.Equal(t, p1, p2.PreviousSibling())
		assert.Equal(t, pos.Pos(14), p2.Pos())
	}

	assert.Equal(t, p2, doc.LastChild())

	t.Run("out of bounds", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, doc.Parent())
		assert.Nil(t, doc.PreviousSibling())
		assert.Nil(t, p2.NextSibling())
	})
}

func TestNode_relation_nil(t *testing.T) {
	t.Parallel()

	var n *Any
	assert.Nil(t, n.Parent())
	assert.Nil(t, n.PreviousSibling())
	assert.Nil(t, n.NextSibling())
}

func TestNode_Parent_fallback(t *testing.T) {
	t.Parallel()

	doc := parse(t, "# Foo\n\nhello\n\nworld").Node
	para, err := Wrap(doc.LastChild())
	require.NoError(t, err)

	p := para.Parent()
	require.NotNil(t, p)
	assert.Equal(t, doc, p.Node)
	assert.Equal(t, pos.Pos(0), p.Pos())
}

func parse(t *testing.T, src string) *Any {
	t.Helper()

	p := goldmark.New().Parser()
	n, err := Parse(p, "", []byte(src))
	require.NoError(t, err)
	return n.AST
}
