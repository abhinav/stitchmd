package goldast

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func TestPosition_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give Position
		want string
	}{
		{
			desc: "all",
			give: Position{File: "foo", Line: 1, Column: 2},
			want: "foo:1:2",
		},
		{
			desc: "no name",
			give: Position{Line: 2, Column: 1},
			want: "2:1",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.give.String())
		})
	}
}

func TestInfo_Position_empty(t *testing.T) {
	t.Parallel()

	info := infoFromContent("foo", nil)
	assert.Equal(t, Position{
		File:   "foo",
		Line:   1,
		Column: 1,
	}, info.Position(0))

	assert.Panics(t, func() {
		info.Position(1)
	}, "out of range lookup should panic")
}

func TestInfo_Position(t *testing.T) {
	t.Parallel()

	info := infoFromContent("a.txt", []byte("foo\nbar\nbaz\n"))
	assert.Equal(t, "a.txt", info.Filename())

	tests := []struct {
		give int
		want Position
	}{
		{0, Position{File: "a.txt", Line: 1, Column: 1}},
		{1, Position{File: "a.txt", Line: 1, Column: 2}},
		{2, Position{File: "a.txt", Line: 1, Column: 3}},

		{4, Position{File: "a.txt", Line: 2, Column: 1}},
		{5, Position{File: "a.txt", Line: 2, Column: 2}},
		{6, Position{File: "a.txt", Line: 2, Column: 3}},

		{8, Position{File: "a.txt", Line: 3, Column: 1}},
		{9, Position{File: "a.txt", Line: 3, Column: 2}},
		{10, Position{File: "a.txt", Line: 3, Column: 3}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprint(tt.give), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, info.Position(tt.give))
		})
	}
}

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
