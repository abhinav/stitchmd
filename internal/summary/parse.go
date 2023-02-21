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
//   - each item is either a link to a Markdown document or a plain text title
//   - items may be nested to indicate a hierarchy
//
// For example:
//
//	# User Guide
//
//	- [Getting Started](getting-started.md)
//	    - [Installation](installation.md)
//	- Options
//	    - [foo](foo.md)
//	    - [bar](bar.md)
//	- [Reference](reference.md)
//
//	# Appendix
//
//	- [FAQ](faq.md)
//
// Anything else will result in an error.
func Parse(f *goldast.File) (*TOC, error) {
	errs := pos.NewErrorList(f.Info)
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

func (p *tocParser) parseSections(n *goldast.Any) {
	for n := n.FirstChild(); n != nil; {
		sec, next := p.parseSection(n)
		if sec != nil {
			p.sections = append(p.sections, sec)
		}
		n = next
	}
}

// parseSection parses a Section from the given node.
func (p *tocParser) parseSection(n *goldast.Any) (*Section, *goldast.Any) {
	title, n := p.parseSectionTitle(n)

	ls, ok := goldast.Cast[*ast.List](n)
	if !ok {
		if title != nil {
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
		AST:   ls,
	}, n.NextSibling()
}

func (p *tocParser) parseSectionTitle(n *goldast.Any) (*SectionTitle, *goldast.Any) {
	h, ok := goldast.Cast[*ast.Heading](n)
	if !ok {
		return nil, n
	}

	return &SectionTitle{
		Text:  string(h.Node.Text(p.src)),
		Level: h.Node.Level,
		AST:   h,
	}, n.NextSibling()
}

// sectionParser is a recursive-descent parser for a hierarchy of list items.
type sectionParser struct {
	src  []byte
	errs *pos.ErrorList

	depth int // current depth
	items tree.List[Item]
}

func (p *sectionParser) child() *sectionParser {
	return &sectionParser{
		src:   p.src,
		errs:  p.errs,
		depth: p.depth + 1,
	}
}

func (p *sectionParser) parse(ls *goldast.List) tree.List[Item] {
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

func (p *sectionParser) parseItem(li *goldast.ListItem) {
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

	var n *goldast.Any
	switch ch := li.FirstChild(); ch.Kind() {
	case ast.KindTextBlock, ast.KindParagraph:
		switch count := ch.ChildCount(); count {
		case 0:
			p.errs.Pushf(ch.Pos(), "list item is empty")
			return
		case 1:
			n = ch.FirstChild()
		default:
			p.errs.Pushf(ch.Pos(), "item has too many children (%v)", count)
			return
		}
	default:
		p.errs.Pushf(ch.Pos(), "expected text or paragraph, got %v", ch.Kind())
		return
	}

	var item Item
	if link, ok := goldast.Cast[*ast.Link](n); ok {
		item = &LinkItem{
			Text:   string(link.Node.Text(p.src)),
			Target: string(link.Node.Destination),
			Depth:  p.depth,
			AST:    link,
		}
	} else if text, ok := goldast.Cast[*ast.Text](n); ok {
		item = &TextItem{
			Text:  string(text.Node.Text(p.src)),
			Depth: p.depth,
			AST:   text,
		}
	} else {
		p.errs.Pushf(n.Pos(), "expected a link or text, got %v", n.Kind())
		return
	}

	tnode := tree.Node[Item]{Value: item}
	if hasChildren {
		ls, ok := goldast.Cast[*ast.List](li.LastChild())
		if !ok {
			p.errs.Pushf(li.LastChild().Pos(), "expected a list, got %v", li.LastChild().Kind())
			return
		}

		tnode.List = p.child().parse(ls)
	}

	p.items = append(p.items, &tnode)
}
