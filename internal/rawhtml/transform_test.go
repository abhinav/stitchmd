package rawhtml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTMLTag(t *testing.T) {
	t.Parallel()

	t.Run("standard", func(t *testing.T) {
		t.Parallel()

		tag := newHTMLTag(nil, []byte("em"))
		assert.Equal(t, "em", tag.Name())
		assert.Equal(t, "<em>", tag.String())
		assert.True(t, tag.EqName(newHTMLTag(nil, []byte("em"))))
	})

	t.Run("non-standard", func(t *testing.T) {
		t.Parallel()

		tag := newHTMLTag(nil, []byte("madeuptag"))
		assert.Equal(t, "madeuptag", tag.Name())
		assert.Equal(t, "<madeuptag>", tag.String())
		assert.True(t, tag.EqName(newHTMLTag(nil, []byte("madeuptag"))))
	})
}
