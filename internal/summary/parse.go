package summary

import (
	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/mdreduce/internal/goldast"
	"go.abhg.dev/mdreduce/internal/pos"
	"go.abhg.dev/mdreduce/internal/tree"
)

// Parse parses a summary from a Markdown document.
// The summary is expected in a very specific format:
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
func Parse(f *goldast.File) (*TOC, error) {
	errs := pos.NewErrorList(f.Positioner)
	parser := newTOCParser(f.Source, errs)
	parser.parseSections(f.AST)
	if len(parser.sections) == 0 && errs.Len() == 0 {
		errs.Pushf(f.Pos, "no sections found")
		return nil, errs.Err()
	}

	return &TOC{
		Sections: parser.sections,
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

	depth int // current depth
	items tree.List[*Item]
}

func (p *sectionParser) child() *sectionParser {
	return &sectionParser{
		src:   p.src,
		errs:  p.errs,
		depth: p.depth + 1,
	}
}

func (p *sectionParser) parse(ls *goldast.Node[*ast.List]) tree.List[*Item] {
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
	node := tree.Node[*Item]{
		Value: &Item{
			Text:  string(link.Node.Text(p.src)),
			File:  dest,
			Depth: p.depth,
			Pos:   link.Pos(),
		},
	}
	if hasChildren {
		ls, ok := goldast.Cast[*ast.List](li.LastChild())
		if !ok {
			p.errs.Pushf(li.LastChild().Pos(), "expected a list, got %v", li.LastChild().Kind())
			return
		}

		node.List = p.child().parse(ls)
	}

	p.items = append(p.items, &node)
}
