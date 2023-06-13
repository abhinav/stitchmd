package rawhtml

import (
	"fmt"
	"sort"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/stitchmd/internal/goldtext"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Transformer searches for raw HTML nodes in the Goldmark document
// and adds information about pairs to the parser context.
//
// It uses a fairly simple algorithm:
// For every open tag, scan until the matching close tag.
type Transformer struct{}

var _ parser.ASTTransformer = (*Transformer)(nil)

// Transform transforms a Markdown document.
func (t *Transformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	// Ignore errors; do a best effort transformation.
	_ = (&transform{
		Context: pc,
		Reader:  reader,
	}).Transform(doc)
}

// transform holds the state for a single Transform operation.
type transform struct {
	Context parser.Context
	Reader  text.Reader

	pairs Pairs     // pairs found in the document
	stack []htmlTag // stack of open tags
}

// Transform transforms the given document.
func (t *transform) Transform(doc *ast.Document) error {
	if err := ast.Walk(doc, t.visit); err != nil {
		return err
	}

	// Sort by start positions to get more deterministic output.
	sort.Slice(t.pairs, func(i, j int) bool {
		l, r := t.pairs[i].Open, t.pairs[j].Open
		return l.Segments.At(0).Start < r.Segments.At(0).Start
	})

	t.pairs.set(t.Context)
	return nil
}

func (t *transform) visit(node ast.Node, enter bool) (ast.WalkStatus, error) {
	n, ok := node.(*ast.RawHTML)
	if !ok {
		return ast.WalkContinue, nil
	}

	if enter {
		t.rawHTML(n)
	}

	return ast.WalkContinue, nil
}

func (t *transform) rawHTML(n *ast.RawHTML) {
	rdr := goldtext.Reader{
		Source:   t.Reader.Source(),
		Segments: n.Segments,
	}
	z := html.NewTokenizerFragment(&rdr, "")
	for {
		switch tok := z.Next(); tok {
		case html.ErrorToken:
			return // ignore bad HTML

		case html.StartTagToken, html.EndTagToken:
			name, _ := z.TagName()
			tag := newHTMLTag(n, name)
			t.tag(tag, tok == html.StartTagToken)
		}
	}
}

func (t *transform) tag(tag htmlTag, startTag bool) {
	if startTag {
		t.stack = append(t.stack, tag)
		return
	}

	// Find the matching open tag.
	openIdx := -1
	for i := len(t.stack) - 1; i >= 0; i-- {
		if t.stack[i].EqName(tag) {
			openIdx = i
			break
		}
	}

	// No matching open tag, just ignore this close tag.
	if openIdx < 0 {
		return
	}

	t.pairs = append(t.pairs, Pair{
		Open:  t.stack[openIdx].parent,
		Close: tag.parent,
	})
	t.stack = t.stack[:openIdx]
}

// htmlTag is an efficient representation of any HTML tag.
// For a majority of tags, it'll use the atom.Atom representation
// which is just an integer.
// For non-standard tags, it'll use the string representation.
type htmlTag struct {
	parent *ast.RawHTML // parent node

	// Exactly one of the following is set.
	atom atom.Atom
	name string
}

func newHTMLTag(parent *ast.RawHTML, bs []byte) (t htmlTag) {
	t.parent = parent
	if a := atom.Lookup(bs); a != 0 {
		t.atom = a
	} else {
		t.name = string(bs)
	}
	return t
}

// EqName reports whether the two tag names are equal.
func (t htmlTag) EqName(other htmlTag) bool {
	// If they're both atoms, int comparison is enough.
	if t.atom != 0 && other.atom != 0 {
		return t.atom == other.atom
	}
	return t.Name() == other.Name()
}

// Name reports the name of the tag.
func (t htmlTag) Name() string {
	if t.atom != 0 {
		return t.atom.String()
	}
	return t.name
}

func (t htmlTag) String() string {
	return fmt.Sprintf("<%s>", t.Name())
}
