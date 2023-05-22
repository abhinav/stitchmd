package rawhtml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

func TestExtender(t *testing.T) {
	t.Parallel()

	type wantPair struct {
		Open   string `yaml:"open"`
		Close  string `yaml:"close"`
		Middle string `yaml:"middle"`
	}

	var tests []struct {
		Desc string     `yaml:"desc"`
		Give string     `yaml:"give"`
		Want []wantPair `yaml:"want"`
	}

	testdata, err := os.ReadFile(filepath.Join("testdata", "tests.yml"))
	require.NoError(t, err, "read testdata")
	require.NoError(t, yaml.Unmarshal(testdata, &tests), "unmarshal testdata")

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Desc, func(t *testing.T) {
			t.Parallel()

			src := []byte(tt.Give)
			pairs := parsePairs(src)

			var got []wantPair
			for _, pair := range pairs {
				mid := text.NewSegments()
				mid.Append(text.NewSegment(
					pair.Open.Segments.At(pair.Open.Segments.Len()-1).Stop,
					pair.Close.Segments.At(0).Start,
				))

				got = append(got, wantPair{
					Open:   segmentText(t, src, pair.Open.Segments),
					Close:  segmentText(t, src, pair.Close.Segments),
					Middle: segmentText(t, src, mid),
				})
			}

			assert.Equal(t, tt.Want, got)
		})
	}
}

// parsePairs parses the given Markdown and returns the HTML pairs
// found in the document.
func parsePairs(src []byte) Pairs {
	md := goldmark.New(
		goldmark.WithExtensions(
			&Extender{},
		),
	)

	ctx := parser.NewContext()
	_ = md.Parser().Parse(text.NewReader(src), parser.WithContext(ctx))

	return GetPairs(ctx)
}
