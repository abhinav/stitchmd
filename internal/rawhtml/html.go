package rawhtml

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/stitchmd/internal/goldtext"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var _closeTagRe = regexp.MustCompile(`(?i)</[a-z][a-z0-9-]*\s*>`)

// Pair is a pair of opening and closing raw HTML nodes.
type Pair struct {
	Open, Close *ast.RawHTML
}

// ParseHTML parses the underlying HTML node for this pair.
func (p *Pair) ParseHTML(src []byte) (*html.Node, error) {
	r := io.MultiReader(
		&goldtext.Reader{Source: src, Segments: p.Open.Segments},
		&goldtext.Reader{Source: src, Segments: p.Close.Segments},
	)
	return parseHTMLSingle(r)
}

// PairFromHTML formats the given HTML node as a [Pair].
// The node must be an HTML tag with separate open and close tags,
// and must not have any children.
// Text needed to render the node is appended to src.
// Returns an error if the HTML node does not have
// separate opening and closing tags.
func PairFromHTML(n *html.Node, src []byte) (*Pair, []byte, error) {
	if n.FirstChild != nil {
		return nil, nil, fmt.Errorf("expected no children, got %d", countSiblings(n.FirstChild)+1)
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, n); err != nil {
		return nil, nil, fmt.Errorf("render html: %w", err)
	}

	loc := _closeTagRe.FindIndex(buf.Bytes())
	if len(loc) == 0 {
		return nil, nil, fmt.Errorf("expected separate open-close tags, got %q", buf.Bytes())
	}

	before := len(src)
	src = append(src, buf.Bytes()...)
	return &Pair{
		Open: &ast.RawHTML{
			Segments: segmentsOf(text.NewSegment(before, before+loc[0])),
		},
		Close: &ast.RawHTML{
			Segments: segmentsOf(text.NewSegment(before+loc[0], before+loc[1])),
		},
	}, src, nil
}

func segmentsOf(segs ...text.Segment) *text.Segments {
	segments := text.NewSegments()
	for _, s := range segs {
		segments.Append(s)
	}
	return segments
}

// ParseHTMLFragmentBodies parses fragments of HTML from the given reader.
// It returns the top-level nodes of the HTML fragments -- unwrapping
// the <html><head></head> tags, yielding the <body>s.
// That is, the direct children of each returned node hold the contents
// of the <body> tag.
func ParseHTMLFragmentBodies(r io.Reader) ([]*html.Node, error) {
	frags, err := html.ParseFragment(r, nil)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	nodes := make([]*html.Node, len(frags))
	for i, hn := range frags {
		// Each fragment begins with <html><head></head><body>.
		// Skip those.
		if hn.DataAtom != atom.Html {
			return nil, fmt.Errorf("expected <html>, got <%s>", hn.Data)
		}

		hn = hn.FirstChild // <html> -> <head>
		if hn == nil || hn.DataAtom != atom.Head {
			return nil, &unexpectedTagError{Want: "head", Have: hn}
		}

		hn = hn.NextSibling // <head> -> <body>
		if hn == nil || hn.DataAtom != atom.Body {
			return nil, &unexpectedTagError{Want: "body", Have: hn}
		}

		nodes[i] = hn
	}

	return nodes, nil
}

// parseHTMLSingle parses HTML from the given reader,
// extracting a single HTML node from it.
//
// It returns an error if the body does not contain
// exactly one HTML node.
func parseHTMLSingle(r io.Reader) (*html.Node, error) {
	frags, err := ParseHTMLFragmentBodies(r)
	if err != nil {
		return nil, err
	}

	if len(frags) != 1 {
		return nil, fmt.Errorf("expected 1 fragment, got %d", len(frags))
	}

	hn := frags[0].FirstChild
	if hn == nil || hn.NextSibling != nil {
		var got int
		if hn != nil {
			got = countSiblings(hn) + 1
		}
		return nil, fmt.Errorf("expected <body> to have 1 child, got %d", got)
	}

	return hn, nil
}

type unexpectedTagError struct {
	Want string
	Have *html.Node
}

func (err *unexpectedTagError) Error() string {
	var s strings.Builder
	fmt.Fprintf(&s, "expected <%s>, got ", err.Want)
	if err.Have == nil {
		s.WriteString("nil")
	} else {
		fmt.Fprintf(&s, "<%s>", err.Have.Data)
	}
	return s.String()
}

func countSiblings(n *html.Node) int {
	var count int
	for n = n.NextSibling; n != nil; n = n.NextSibling {
		count++
	}
	return count
}
