package goldast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func TestOffsetOf(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		assert.Zero(t, OffsetOf(nil))
	})

	t.Run("document", func(t *testing.T) {
		t.Parallel()

		n := ast.NewDocument()
		assert.Zero(t, OffsetOf(n))
	})

	t.Run("empty block", func(t *testing.T) {
		t.Parallel()

		n := ast.NewParagraph()
		n.SetLines(text.NewSegments())
		assert.Zero(t, OffsetOf(n))
	})

	t.Run("block", func(t *testing.T) {
		t.Parallel()

		segs := text.NewSegments()
		segs.Append(text.NewSegment(5, 10))

		n := ast.NewParagraph()
		n.SetLines(segs)

		assert.Equal(t, 5, OffsetOf(n))
	})

	t.Run("inline", func(t *testing.T) {
		t.Parallel()

		segs := text.NewSegments()
		segs.Append(text.NewSegment(5, 10))

		para := ast.NewParagraph()
		para.SetLines(segs)

		n := ast.NewString([]byte("hello"))
		para.AppendChild(para, n)

		assert.Equal(t, 5, OffsetOf(n))
	})
}
