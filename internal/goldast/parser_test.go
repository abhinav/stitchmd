package goldast

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	f := Parse(DefaultParser(), "foo.md", []byte("hello world"))

	require.Equal(t, "hello world", string(f.Source))
	require.Equal(t, "foo.md:1:1", f.Position(0).String())
}
