package rawhtml

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/stitchmd/internal/goldtext"
	"golang.org/x/net/html"
)

func TestCloseTagRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give string
		want []int
	}{
		{
			desc: "simple",
			give: "</em>",
			want: []int{0, 5},
		},
		{
			desc: "pair",
			give: "<em></em>",
			want: []int{4, 9},
		},
		{
			desc: "uppercase",
			give: "<EM></EM>",
			want: []int{4, 9},
		},
		{
			desc: "space",
			give: "<em></em  >",
			want: []int{4, 11},
		},
		{
			desc: "newline",
			give: "<em></em\n>",
			want: []int{4, 10},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			got := _closeTagRe.FindStringIndex(tt.give)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPair_parseHTMLSingle_errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc    string
		give    string
		wantErr string
	}{
		{
			desc:    "too many children",
			give:    "<p></p><p></p><p></p>",
			wantErr: "expected <body> to have 1 child, got 3",
		},
		{
			desc:    "no children",
			give:    "",
			wantErr: "expected <body> to have 1 child, got 0",
		},
		{
			desc:    "no head",
			give:    "<!foo",
			wantErr: "expected <head>, got <foo>",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			_, err := parseHTMLSingle(strings.NewReader(tt.give))
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestPairFromHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give string

		wantOpen, wantClose string
	}{
		{
			desc:      "simple",
			give:      "<em></em>",
			wantOpen:  "<em>",
			wantClose: "</em>",
		},
		{
			desc:      "with attributes",
			give:      `<em class="foo" id="bar"></em>`,
			wantOpen:  `<em class="foo" id="bar">`,
			wantClose: "</em>",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			h, err := parseHTMLSingle(strings.NewReader(tt.give))
			require.NoError(t, err)

			pair, src, err := PairFromHTML(h, nil /* src */)
			require.NoError(t, err)

			gotOpen := segmentText(t, src, pair.Open.Segments)
			gotClose := segmentText(t, src, pair.Close.Segments)

			assert.Equal(t, tt.wantOpen, gotOpen)
			assert.Equal(t, tt.wantClose, gotClose)
		})
	}
}

func TestPairFromHTML_errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc    string
		give    string
		wantErr string
	}{
		{
			desc:    "too many children",
			give:    "<em>foo</em>",
			wantErr: "expected no children, got 1",
		},
		{
			desc:    "no open-close",
			give:    `<img src="foo">`,
			wantErr: `expected separate open-close tags, got "<img src=\"foo\"/>`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			h, err := parseHTMLSingle(strings.NewReader(tt.give))
			require.NoError(t, err)

			_, _, err = PairFromHTML(h, nil /* src */)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// Fuzz test over Goldmark's RawHTML parsing
// to verify that it never creates a RawHTML node
// with multiple HTML tags in it.
//
// This is just to sanity check our own parsing
// because if/when this changes, our logic will need to be updated.
func FuzzGoldmarkRawHTML(f *testing.F) {
	f.Add("foo <em>bar</em> <strong>baz</strong> qux")
	f.Add("# foo <code>bar</code> <strong>baz</strong> qux")
	f.Add("# Foo\n\n<em>bar</em> <strong>baz</strong> qux")
	f.Add("<em>hello <strong>world</strong></em>")
	f.Fuzz(func(t *testing.T, s string) {
		src := []byte(s)

		p := goldmark.DefaultParser()
		doc := p.Parse(text.NewReader(src))
		err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}

			h, ok := n.(*ast.RawHTML)
			if !ok {
				return ast.WalkContinue, nil
			}

			count := countTags(&goldtext.Reader{
				Source:   src,
				Segments: h.Segments,
			})
			if count > 1 {
				t.Errorf("RawHTML node with %d tags", count)
			}

			return ast.WalkContinue, nil
		})
		require.NoError(t, err)
	})
}

func TestUnexpectedTagError(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		err := unexpectedTagError{Want: "foo"}
		assert.Equal(t, "expected <foo>, got nil", err.Error())
	})

	t.Run("non-nil", func(t *testing.T) {
		t.Parallel()

		err := unexpectedTagError{
			Want: "foo",
			Have: &html.Node{Type: html.ElementNode, Data: "bar"},
		}
		assert.Equal(t, "expected <foo>, got <bar>", err.Error())
	})
}

func countTags(r io.Reader) (count int) {
	z := html.NewTokenizerFragment(r, "")
	for {
		switch tok := z.Next(); tok {
		case html.ErrorToken:
			return count

		case html.StartTagToken, html.EndTagToken:
			count++
		}
	}
}

// Returns the contents of a segment as a string.
func segmentText(t *testing.T, src []byte, segs *text.Segments) string {
	if segs == nil {
		return ""
	}
	var buff bytes.Buffer
	_, err := io.Copy(&buff, &goldtext.Reader{Source: src, Segments: segs})
	require.NoError(t, err)
	return buff.String()
}
