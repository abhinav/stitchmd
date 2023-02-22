package stitch

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/stitchmd/internal/goldast"
	"go.abhg.dev/stitchmd/internal/pos"
	"go.abhg.dev/stitchmd/internal/tree"
)

// Item is a single item in a section.
// It can be a [LinkItem] or a [TextItem].
type Item interface {
	item() // seals the interface

	Pos() pos.Pos
}

// itemTreeParser is a recursive-descent parser for a hierarchy of list items.
type itemTreeParser struct {
	src  []byte
	errs *pos.ErrorList

	depth int // current depth
	items tree.List[Item]
}

func (p *itemTreeParser) child() *itemTreeParser {
	return &itemTreeParser{
		src:   p.src,
		errs:  p.errs,
		depth: p.depth + 1,
	}
}

func (p *itemTreeParser) Parse(ls *goldast.List) tree.List[Item] {
	for ch := ls.FirstChild(); ch != nil; ch = ch.NextSibling() {
		li, ok := goldast.Cast[*ast.ListItem](ch)
		if !ok {
			// Impossible for parsed ASTs.
			// Only hand-crafted ASTs could trigger this.
			p.errs.Pushf(ch.Pos(), "expected a list item, got %v", ch.Kind())
			continue
		}
		p.parseItem(li)
	}
	return p.items
}

func (p *itemTreeParser) parseItem(li *goldast.ListItem) {
	var (
		// Node holding the item's link or text.
		n *goldast.Any

		// Children of this node, if any.
		children *goldast.List
	)
	switch count := li.ChildCount(); count {
	case 0:
		p.errs.Pushf(li.Pos(), "list item is empty")
		return

	case 2:
		ch, ok := goldast.Cast[*ast.List](li.LastChild())
		if !ok {
			p.errs.Pushf(li.LastChild().Pos(), "expected a list, got %v", li.LastChild().Kind())
			return
		}
		children = ch
		fallthrough
	case 1:
		switch ch := li.FirstChild(); ch.Kind() {
		case ast.KindTextBlock, ast.KindParagraph:
			n = ch
		default:
			p.errs.Pushf(ch.Pos(), "expected text or paragraph, got %v", ch.Kind())
			return
		}

	default:
		childKinds := make([]string, 0, count)
		for ch := li.FirstChild(); ch != nil; ch = ch.NextSibling() {
			childKinds = append(childKinds, ch.Kind().String())
		}

		p.errs.Pushf(li.FirstChild().Pos(), "list item has too many children (%v): %v", count, childKinds)
		return
	}

	combineTextNodes(n.Node)
	switch count := n.ChildCount(); count {
	case 0:
		p.errs.Pushf(n.Pos(), "list item is empty")
		return
	case 1:
		n = n.FirstChild()
	default:
		childKinds := make([]string, 0, count)
		for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
			childKinds = append(childKinds, ch.Kind().String())
		}
		p.errs.Pushf(n.Pos(), "text has too many children (%v): %v", count, childKinds)
		return
	}

	var item Item
	if link, ok := goldast.Cast[*ast.Link](n); ok {
		item = p.parseLinkItem(link)
	} else if text, ok := goldast.Cast[*ast.Text](n); ok {
		item = p.parseTextItem(text, children != nil)
	} else {
		p.errs.Pushf(n.Pos(), "expected a link or text, got %v", n.Kind())
		return
	}

	tnode := tree.Node[Item]{Value: item}
	if children != nil {
		tnode.List = p.child().Parse(children)
	}

	p.items = append(p.items, &tnode)
}

// LinkItem is a single link item in a table of contents.
//
//	[Foo](foo.md)
type LinkItem struct {
	// Text of the item.
	// This is the text inside the "[..]" section of the link.
	Text string

	// Target is the destination of this item.
	// This is the text inside the "(..)" section of the link.
	Target string

	// Depth is the depth of the item in the table of contents.
	// Depth starts at zero for top-level items.
	Depth int

	// AST holds the original link node.
	AST *goldast.Link
}

func (p *itemTreeParser) parseLinkItem(link *goldast.Link) *LinkItem {
	return &LinkItem{
		Text:   string(link.Node.Text(p.src)),
		Target: string(link.Node.Destination),
		Depth:  p.depth,
		AST:    link,
	}
}

func (*LinkItem) item() {}

// Pos reports the position in the original TOC
// where this item was found.
func (i *LinkItem) Pos() pos.Pos {
	return i.AST.Pos()
}

// TextItem is a single text entry in the table of contents.
//
//	Foo
type TextItem struct {
	// Text of the item.
	Text string

	// Depth is the depth of the item in the table of contents.
	// Depth starts at zero for top-level items.
	Depth int

	// AST holds the original text node.
	AST *goldast.Text
}

func (p *itemTreeParser) parseTextItem(text *goldast.Text, hasChildren bool) *TextItem {
	if !hasChildren {
		p.errs.Pushf(text.Pos(), "text item must have children")
		return nil
	}

	return &TextItem{
		Text:  string(text.Node.Text(p.src)),
		Depth: p.depth,
		AST:   text,
	}
}

func (*TextItem) item() {}

// Pos reports the position in the original TOC
// where this item was found.
func (i *TextItem) Pos() pos.Pos {
	return i.AST.Pos()
}

// combineTextNodes combines adjacent text child nodes of the given node
// into a single text node.
//
// This is necessary because sometimes,
// the parser will split text in a text block into multiple nodes.
func combineTextNodes(n ast.Node) {
	// TODO: Maybe this should be a transformer on the Goldmark parser.
	if n.ChildCount() <= 1 {
		return
	}

	var (
		idx int
		seg text.Segment
	)
	for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
		if ch.Kind() != ast.KindText {
			return
		}
		if idx == 0 {
			seg.Start = ch.(*ast.Text).Segment.Start
		}
		seg.Stop = ch.(*ast.Text).Segment.Stop
		idx++
	}

	newch := ast.NewTextSegment(seg)
	for n.ChildCount() > 0 {
		n.RemoveChild(n, n.FirstChild())
	}
	n.AppendChild(n, newch)
}
