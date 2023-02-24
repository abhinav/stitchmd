package pos

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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

	info := FromContent("foo", nil)
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

	info := FromContent("a.txt", []byte("foo\nbar\nbaz\n"))
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
