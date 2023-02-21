package goldast

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
)

func TestParse(t *testing.T) {
	t.Parallel()

	parser := goldmark.New().Parser()
	f, err := Parse(parser, "foo.md", []byte("hello world"))
	require.NoError(t, err)

	require.Equal(t, "foo.md", f.Name)
	require.Equal(t, "hello world", string(f.Source))
	require.Equal(t, "foo.md:1:1", f.Positioner.Position(f.Pos).String())
}
