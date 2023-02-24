package goldast

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
)

func TestParse(t *testing.T) {
	t.Parallel()

	parser := goldmark.New().Parser()
	f := Parse(parser, "foo.md", []byte("hello world"))

	require.Equal(t, "hello world", string(f.Source))
	require.Equal(t, "foo.md:1:1", f.Info.Position(0).String())
}
