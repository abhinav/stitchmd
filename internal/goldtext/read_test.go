package goldtext

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark/text"
)

//nolint:paralleltest // shared state in subtests
func TestReader_empty(t *testing.T) {
	t.Parallel()

	r := newReaderFrom(t, "non empty input")

	t.Run("Read", func(t *testing.T) {
		r.Reset()

		_, err := r.Read(make([]byte, 42))
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("ReadAll", func(t *testing.T) {
		r.Reset()

		got, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestReader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc    string
		src     string
		offsets []int // pairs

		reads []string
	}{
		{
			desc: "contiguous",
			//              1
			//    012345678901
			src: "foobarbazqux",
			offsets: []int{
				0, 3,
				3, 7,
				7, 10,
				10, 12,
			},
			//              012    3456    789    01
			reads: []string{"foo", "barb", "azq", "ux"},
		},
		{
			desc: "broken",
			//              1
			//    012345678901
			src: "foobarbazqux",
			offsets: []int{
				0, 3,
				6, 9,
			},
			reads: []string{"f", "o", "o", "baz"},
		},
		{
			desc: "overlapping", // unlikely but possible
			//              1
			//    012345678901
			src: "foobarbazqux",
			offsets: []int{
				0, 4, // foob
				3, 7, // barb
			},
			reads: []string{"fo", "obb", "arb"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			t.Run("ReadAll", func(t *testing.T) {
				t.Parallel()

				var want strings.Builder
				for _, r := range tt.reads {
					want.WriteString(r)
				}

				r := newReaderFrom(t, tt.src, tt.offsets...)
				got, err := io.ReadAll(r)
				require.NoError(t, err)
				assert.Equal(t, want.String(), string(got))
			})

			t.Run("Read", func(t *testing.T) {
				t.Parallel()

				r := newReaderFrom(t, tt.src, tt.offsets...)
				for _, want := range tt.reads {
					got := make([]byte, len(want))
					n, err := r.Read(got)
					require.NoError(t, err, "read(%q)", want)
					assert.Equal(t, len(want), n, "read(%q): bytes read", want)
					assert.Equal(t, want, string(got), "read(%q): contents", want)
				}
			})
		})
	}
}

// Builds a segment from a source string and pairs of segment offsets.
func newReaderFrom(t testing.TB, src string, offsets ...int) *Reader {
	t.Helper()

	require.Zero(t, len(offsets)%2,
		"expected pairs of offsets, got %d offsets", len(offsets))

	segments := text.NewSegments()
	for i := 0; i < len(offsets); i += 2 {
		start, stop := offsets[i], offsets[i+1]
		require.GreaterOrEqual(t, start, 0, "offset must be positive")
		require.LessOrEqual(t, start, len(src), "offset must be within bounds")

		require.GreaterOrEqual(t, stop, 0, "offset must be positive")
		require.LessOrEqual(t, stop, len(src), "offset must be within bounds")

		segment := text.NewSegment(start, stop)
		segments.Append(segment)
	}

	return &Reader{Source: []byte(src), Segments: segments}
}
