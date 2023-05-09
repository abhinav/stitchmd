package rawhtml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func TestPairs_getAndSet(t *testing.T) {
	t.Parallel()

	t.Run("unset", func(t *testing.T) {
		t.Parallel()

		ctx := parser.NewContext()
		assert.Empty(t, GetPairs(ctx))
	})

	t.Run("set/empty", func(t *testing.T) {
		t.Parallel()

		ctx := parser.NewContext()
		var p Pairs
		p.set(ctx)

		assert.Empty(t, GetPairs(ctx))
	})

	t.Run("set/non-empty", func(t *testing.T) {
		t.Parallel()

		ctx := parser.NewContext()
		p := Pairs{
			{
				Open: &ast.RawHTML{
					Segments: segmentsOf(
						text.NewSegment(1, 2),
					),
				},
				Close: &ast.RawHTML{
					Segments: segmentsOf(
						text.NewSegment(3, 4),
					),
				},
			},
		}
		p.set(ctx)

		assert.Equal(t, p, GetPairs(ctx))
	})
}
