package goldast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func TestCombineAdjacentTexts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		text string
		segs []text.Segment
		want []string
	}{
		{
			desc: "empty",
			text: "foobar",
		},
		{
			desc: "single",
			text: "foobar",
			segs: []text.Segment{
				{Start: 0, Stop: 3},
			},
			want: []string{"foo"},
		},
		{
			desc: "adjacent span",
			text: "foobar",
			segs: []text.Segment{
				{Start: 0, Stop: 3},
				{Start: 3, Stop: 6},
			},
			want: []string{"foobar"},
		},
		{
			desc: "separate adjacent spans",
			//     0123456
			text: "foobar",
			segs: []text.Segment{
				{Start: 0, Stop: 1},
				{Start: 1, Stop: 2},
				{Start: 2, Stop: 3},
				{Start: 4, Stop: 5},
				{Start: 5, Stop: 6},
			},
			want: []string{"foo", "ar"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			src := []byte(tt.text)
			n := ast.NewTextBlock()
			for _, seg := range tt.segs {
				_ = seg.Value(src) // panics if out of bounds
				n.AppendChild(n, ast.NewTextSegment(seg))
			}

			CombineAdjacentTexts(n, src)

			var got []string
			for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
				got = append(got, string(ch.(*ast.Text).Segment.Value(src)))
			}
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("non-text", func(t *testing.T) {
		t.Parallel()

		//             012345678901
		src := []byte("foo bar baz")
		n := ast.NewParagraph()
		n.AppendChild(n, ast.NewTextSegment(text.Segment{Start: 0, Stop: 3}))
		n.AppendChild(n, ast.NewEmphasis(1))
		n.AppendChild(n, ast.NewTextSegment(text.Segment{Start: 4, Stop: 7}))

		CombineAdjacentTexts(n, src)

		var got []string
		for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
			if t, ok := ch.(*ast.Text); ok {
				got = append(got, string(t.Segment.Value(src)))
			}
		}
		assert.Equal(t, []string{"foo", "bar"}, got)
	})
}
