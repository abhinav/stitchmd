package main

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.abhg.dev/stitchmd/internal/goldast"
	"go.abhg.dev/stitchmd/internal/stitch"
	"go.abhg.dev/stitchmd/internal/tree"
)

func TestCollector_fileDoesNotExist(t *testing.T) {
	t.Parallel()

	file := goldast.Parse(
		goldast.DefaultParser(),
		"stdin",
		[]byte("- [foo](foo.md)"),
	)
	summary, err := stitch.ParseSummary(file)
	require.NoError(t, err)

	_, err = (&collector{
		Parser: goldast.DefaultParser(),
		FS:     make(fstest.MapFS), // empty filesystem
	}).Collect(file.Info, summary)
	require.Error(t, err)

	assert.ErrorContains(t, err, "foo.md: file does not exist")
}

func TestCollector_unknownItemType(t *testing.T) {
	t.Parallel()

	type badItem struct {
		stitch.Item
	}

	summary := &stitch.Summary{
		Sections: []*stitch.Section{
			{Items: tree.List[stitch.Item]{
				{Value: badItem{}},
			}},
		},
	}

	assert.Panics(t, func() {
		(&collector{
			Parser: goldast.DefaultParser(),
			FS:     make(fstest.MapFS),
		}).Collect(fixedPositioner{Line: 1, Column: 1}, summary)
	})
}

type fixedPositioner goldast.Position

var _ goldast.Positioner = fixedPositioner{}

func (p fixedPositioner) Position(offset int) goldast.Position {
	return goldast.Position(p)
}
