package main

import (
	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/mdreduce/internal/goldast"
	"go.abhg.dev/mdreduce/internal/pos"
)

// TOC is a list of sections in a TOC file.
type TOC struct {
	Sections []*Section
	Source   []byte

	Positioner pos.Positioner
}

// Section is a section of a table of contents.
// It's comprised of an optional title and a list of items.
type Section struct {
	// Title of the section, if any.
	Title string

	// Tree of items in the section.
	Items []*Item

	// AST nodes that make up this section.
	AST []*goldast.Node[ast.Node]
}

// visitAllItems recursively calls the given function
// for every item in this section and its descendants.
func (s *Section) visitAllItems(f func(*Item)) {
	for _, item := range s.Items {
		item.visitAll(f)
	}
}

// Item is a single iten in a table of contents.
type Item struct {
	// Title is the title of the page.
	// This is the text inside the "[..]" section of the link.
	Title string

	// File is the path to the Markdown file.
	// This is the text inside the "(..)" section of the link.
	File string

	// Depth is the depth of the item in the table of contents.
	// Depth starts at zero for top-level items.
	Depth int

	// Items is a list of items nested under this item, if any.
	Items []*Item

	// Position at which this item was found.
	Pos pos.Pos
}

func (i *Item) visitAll(f func(*Item)) {
	f(i)
	for _, item := range i.Items {
		item.visitAll(f)
	}
}

// parseTOC parses a table of contents from a Markdown List node
// representing the table of contents.
//
// The table of contents is expected in a very specific format:
//
//   - it's comprised of one or more sections
//   - each section has an optional title header and a list of items
//   - each item is a link to another Markdown document
//   - items may be nested to indicate a hierarchy
//
// For example:
//
//	# User Guide
//
//	- [foo](foo.md)
//	- [bar](bar.md)
//	    - [baz](baz.md)
//	- [qux](qux.md)
//
//	# Appendix
//
//	- [Appendix A](appendix-a.md)
//
// Anything else will result in an error.
func parseTOC(filename string, src []byte, doc *goldast.Node[ast.Node]) (*TOC, error) {
	posc := pos.FromContent(filename, src)
	errs := pos.NewErrorList(posc)
	parser := newTOCParser(src, errs)
	parser.parseSections(doc)
	if len(parser.sections) == 0 && errs.Len() == 0 {
		errs.Pushf(doc.Pos(), "no sections found")
		return nil, errs.Err()
	}

	// TODO: parseTOC should take a file
	return &TOC{
		Sections:   parser.sections,
		Source:     src,
		Positioner: posc,
	}, errs.Err()
}

type tocParser struct {
	src      []byte
	errs     *pos.ErrorList
	sections []*Section
}

func newTOCParser(src []byte, errs *pos.ErrorList) *tocParser {
	return &tocParser{
		src:  src,
		errs: errs,
	}
}

func (p *tocParser) parseSections(n *goldast.Node[ast.Node]) {
	for n := n.FirstChild(); n != nil; {
		sec, next := p.parseSection(n)
		if sec != nil {
			p.sections = append(p.sections, sec)
		}
		n = next
	}
}

func (p *tocParser) parseSection(n *goldast.Node[ast.Node]) (*Section, *goldast.Node[ast.Node]) {
	var astNodes []*goldast.Node[ast.Node]

	var title string
	if h, ok := goldast.Cast[*ast.Heading](n); ok {
		astNodes = append(astNodes, n)
		title = string(h.Node.Text(p.src))
		n = n.NextSibling()
	}

	astNodes = append(astNodes, n)
	ls, ok := goldast.Cast[*ast.List](n)
	if !ok {
		if len(title) > 0 {
			p.errs.Pushf(n.Pos(), "expected a list, got %v", n.Kind())
		} else {
			p.errs.Pushf(n.Pos(), "expected a list or heading, got %v", n.Kind())
		}
		return nil, n.NextSibling()
	}

	items := (&sectionParser{
		src:  p.src,
		errs: p.errs,
	}).parse(ls)
	return &Section{
		Title: title,
		Items: items,
		AST:   astNodes,
	}, n.NextSibling()
}

// sectionParser is a recursive-descent parser for a hierarchy of list items.
type sectionParser struct {
	src  []byte
	errs *pos.ErrorList

	depth int     // current depth
	items []*Item // items parsed so far
}

func (p *sectionParser) child() *sectionParser {
	return &sectionParser{
		src:   p.src,
		errs:  p.errs,
		depth: p.depth + 1,
	}
}

func (p *sectionParser) parse(ls *goldast.Node[*ast.List]) []*Item {
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

func (p *sectionParser) parseItem(li *goldast.Node[*ast.ListItem]) {
	var hasChildren bool
	switch count := li.ChildCount(); count {
	case 0:
		p.errs.Pushf(li.Pos(), "list item is empty")
		return
	case 1:
		// do nothing
	case 2:
		hasChildren = true
	default:
		p.errs.Pushf(li.FirstChild().Pos(), "item has too many children (%v)", count)
		return
	}

	var link *goldast.Node[*ast.Link]
	switch ch := li.FirstChild(); ch.Kind() {
	case ast.KindTextBlock, ast.KindParagraph:
		switch count := ch.ChildCount(); count {
		case 0:
			p.errs.Pushf(ch.Pos(), "list item is empty")
			return
		case 1:
			// do nothing
		default:
			p.errs.Pushf(ch.Pos(), "item has too many children (%v)", count)
			return
		}

		var ok bool
		link, ok = goldast.Cast[*ast.Link](ch.FirstChild())
		if !ok {
			p.errs.Pushf(ch.Pos(), "expected a link, got %v", ch.FirstChild().Kind())
			return
		}
	default:
		p.errs.Pushf(ch.Pos(), "expected text or paragraph, got %v", ch.Kind())
		return
	}

	dest := string(link.Node.Destination)
	// TODO: validate dest
	item := Item{
		Title: string(link.Node.Text(p.src)),
		File:  dest,
		Depth: p.depth,
		Pos:   link.Pos(),
	}
	if hasChildren {
		ls, ok := goldast.Cast[*ast.List](li.LastChild())
		if !ok {
			p.errs.Pushf(li.LastChild().Pos(), "expected a list, got %v", li.LastChild().Kind())
			return
		}

		item.Items = p.child().parse(ls)
	}

	p.items = append(p.items, &item)
}
