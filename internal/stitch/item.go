package stitch

import (
	"path/filepath"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/stitchmd/internal/goldast"
	"go.abhg.dev/stitchmd/internal/tree"
)

// Item is a single item in a section.
// It can be a [LinkItem] or a [TextItem].
type Item interface {
	item() // seals the interface

	ItemDepth() int
	Node() ast.Node
}

// itemTreeParser is a recursive-descent parser for a hierarchy of list items.
type itemTreeParser struct {
	src  []byte
	errs *goldast.ErrorList

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
			p.errs.Pushf(ch, "expected a list item, got %v", ch.Kind())
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
		p.errs.Pushf(li, "list item is empty")
		return

	case 2:
		ch, ok := li.LastChild().(*ast.List)
		if !ok {
			p.errs.Pushf(li.LastChild(), "expected a list, got %v", li.LastChild().Kind())
			return
		}
		children = ch
		fallthrough
	case 1:
		switch ch := li.FirstChild(); ch.Kind() {
		case ast.KindTextBlock, ast.KindParagraph:
			n = ch
		default:
			p.errs.Pushf(ch, "expected text or paragraph, got %v", ch.Kind())
			return
		}

	default:
		childKinds := make([]string, 0, count)
		for ch := li.FirstChild(); ch != nil; ch = ch.NextSibling() {
			childKinds = append(childKinds, ch.Kind().String())
		}

		p.errs.Pushf(li.FirstChild(), "list item has too many children (%v): %v", count, childKinds)
		return
	}

	goldast.CombineAdjacentTexts(n, p.src)
	switch count := n.ChildCount(); count {
	case 0:
		p.errs.Pushf(n, "list item is empty")
		return
	case 1:
		n = n.FirstChild()
	default:
		childKinds := make([]string, 0, count)
		for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
			childKinds = append(childKinds, ch.Kind().String())
		}
		p.errs.Pushf(n, "text has too many children (%v): %v", count, childKinds)
		return
	}

	var item Item
	switch n := n.(type) {
	case *ast.Link:
		item = p.parseLinkItem(n)
		// TODO: separate link and external link?
		// TODO: external link can't have children validation should be
		// here
	case *ast.Image:
		item = p.parseEmbedItem(n)
		// TODO: embed can't have children
	case *ast.Text:
		item = p.parseTextItem(n, children != nil)
	default:
		p.errs.Pushf(n, "expected a link or text, got %v", n.Kind())
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
	// It's /-separated, even on Windows.
	Target string

	// Depth is the depth of the item in the table of contents.
	// Depth starts at zero for top-level items.
	Depth int

	// AST holds the original link node.
	AST *ast.Link
}

func (p *itemTreeParser) parseLinkItem(link *ast.Link) *LinkItem {
	return &LinkItem{
		Text:   string(goldast.Text(p.src, link)),
		Target: filepath.ToSlash(string(link.Destination)),
		Depth:  p.depth,
		AST:    link,
	}
}

func (*LinkItem) item() {}

// ItemDepth reports the depth of the item in the table of contents.
func (i *LinkItem) ItemDepth() int {
	return i.Depth
}

// Node reports the underlying AST node
// that this item was parsed from.
func (i *LinkItem) Node() ast.Node {
	return i.AST
}

// EmbedItem is a reference to another summary file
// intended to be nested in the table of contents.
//
//	![Foo](foo.md)
type EmbedItem struct {
	// Title of the link.
	// This is the text inside the "[..]" section.
	Text string

	// Target is the destination of this item.
	// This is the text inside the "(..)" section of the link.
	// It's /-separated, even on Windows.
	Target string

	// Depth is the depth of this item in the table of contents.
	Depth int

	// AST holds the original node.
	AST ast.Node
}

var _ Item = (*EmbedItem)(nil)

func (p *itemTreeParser) parseEmbedItem(embed *ast.Image) *EmbedItem {
	return &EmbedItem{
		Text:   string(goldast.Text(p.src, embed)),
		Target: filepath.ToSlash(string(embed.Destination)),
		Depth:  p.depth,
		AST:    embed,
	}
}

func (*EmbedItem) item() {}

// ItemDepth reports the depth of this embed request in the TOC.
func (i *EmbedItem) ItemDepth() int {
	return i.Depth
}

// Node returns the original AST node in the summary file
// that this item is built from.
func (i *EmbedItem) Node() ast.Node {
	return i.AST
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
		p.errs.Pushf(text, "text item must have children")
		return nil
	}

	return &TextItem{
		Text:  string(goldast.Text(p.src, text)),
		Depth: p.depth,
		AST:   text,
	}
}

func (*TextItem) item() {}

// ItemDepth reports the depth of the item in the table of contents.
func (i *TextItem) ItemDepth() int {
	return i.Depth
}

// Node reports the underlying AST node
// that this item was parsed from.
func (i *TextItem) Node() ast.Node {
	return i.AST
}
