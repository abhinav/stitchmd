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

	Offset() int
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

func (p *itemTreeParser) Parse(ls *ast.List) tree.List[Item] {
	for ch := ls.FirstChild(); ch != nil; ch = ch.NextSibling() {
		li, ok := ch.(*ast.ListItem)
		if !ok {
			// Impossible for parsed ASTs.
			// Only hand-crafted ASTs could trigger this.
			p.errs.Pushf(goldast.OffsetOf(ch), "expected a list item, got %v", ch.Kind())
			continue
		}
		p.parseItem(li)
	}
	return p.items
}

func (p *itemTreeParser) parseItem(li *ast.ListItem) {
	var (
		// Node holding the item's link or text.
		n ast.Node

		// Children of this node, if any.
		children *ast.List
	)
	switch count := li.ChildCount(); count {
	case 0:
		p.errs.Pushf(goldast.OffsetOf(li), "list item is empty")
		return

	case 2:
		ch, ok := li.LastChild().(*ast.List)
		if !ok {
			p.errs.Pushf(goldast.OffsetOf(li.LastChild()), "expected a list, got %v", li.LastChild().Kind())
			return
		}
		children = ch
		fallthrough
	case 1:
		switch ch := li.FirstChild(); ch.Kind() {
		case ast.KindTextBlock, ast.KindParagraph:
			n = ch
		default:
			p.errs.Pushf(goldast.OffsetOf(ch), "expected text or paragraph, got %v", ch.Kind())
			return
		}

	default:
		childKinds := make([]string, 0, count)
		for ch := li.FirstChild(); ch != nil; ch = ch.NextSibling() {
			childKinds = append(childKinds, ch.Kind().String())
		}

		p.errs.Pushf(goldast.OffsetOf(li.FirstChild()), "list item has too many children (%v): %v", count, childKinds)
		return
	}

	combineTextNodes(n)
	switch count := n.ChildCount(); count {
	case 0:
		p.errs.Pushf(goldast.OffsetOf(n), "list item is empty")
		return
	case 1:
		n = n.FirstChild()
	default:
		childKinds := make([]string, 0, count)
		for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
			childKinds = append(childKinds, ch.Kind().String())
		}
		p.errs.Pushf(goldast.OffsetOf(n), "text has too many children (%v): %v", count, childKinds)
		return
	}

	var item Item
	switch n := n.(type) {
	case *ast.Link:
		item = p.parseLinkItem(n)
	case *ast.Text:
		item = p.parseTextItem(n, children != nil)
	default:
		p.errs.Pushf(goldast.OffsetOf(n), "expected a link or text, got %v", n.Kind())
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
	AST *ast.Link
}

func (p *itemTreeParser) parseLinkItem(link *ast.Link) *LinkItem {
	return &LinkItem{
		Text:   string(link.Text(p.src)),
		Target: string(link.Destination),
		Depth:  p.depth,
		AST:    link,
	}
}

func (*LinkItem) item() {}

// Offset returns the offset in the summary document
// at which the link item appears.
func (i *LinkItem) Offset() int {
	return goldast.OffsetOf(i.AST)
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
	AST *ast.Text
}

func (p *itemTreeParser) parseTextItem(text *ast.Text, hasChildren bool) *TextItem {
	if !hasChildren {
		p.errs.Pushf(goldast.OffsetOf(text), "text item must have children")
		return nil
	}

	return &TextItem{
		Text:  string(text.Text(p.src)),
		Depth: p.depth,
		AST:   text,
	}
}

func (*TextItem) item() {}

// Offset returns the offset in the summary document
// at which the text item appears.
func (i *TextItem) Offset() int {
	return goldast.OffsetOf(i.AST)
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
