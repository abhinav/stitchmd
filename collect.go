package main

import (
	"fmt"
	"io/fs"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/mdreduce/internal/goldast"
	"go.abhg.dev/mdreduce/internal/header"
	"go.abhg.dev/mdreduce/internal/pos"
	"go.abhg.dev/mdreduce/internal/summary"
	"go.abhg.dev/mdreduce/internal/tree"
)

// collector loads all Markdown files in a TOC
// and builds an alternative, parsed representation of the files.
type collector struct {
	Parser parser.Parser
	FS     fs.FS
	IDGen  *header.IDGen

	files map[string]*markdownFileItem // path => file
}

func (c *collector) Collect(f *goldast.File) ([]*markdownSection, error) {
	toc, err := summary.Parse(f)
	if err != nil {
		return nil, err
	}

	errs := pos.NewErrorList(f.Positioner)
	sections := make([]*markdownSection, len(toc.Sections))
	for i, sec := range toc.Sections {
		items := tree.TransformList(sec.Items, func(item summary.Item) markdownItem {
			i, err := c.collectItem(item)
			if err != nil {
				errs.Pushf(item.ASTNode().Pos(), "%v", err)
				return nil
			}
			return i
		})

		sections[i] = &markdownSection{
			Source:  f.Source,
			Section: sec,
			Items:   items,
		}
	}

	return sections, errs.Err()
}

func (c *collector) collectItem(item summary.Item) (markdownItem, error) {
	switch item := item.(type) {
	case *summary.LinkItem:
		return c.collectFileItem(item)

	case *summary.TextItem:
		return c.collectGroupItem(item), nil

	default:
		panic(fmt.Sprintf("unhandled item type %T", item))
	}
}

type markdownSection struct {
	*summary.Section

	Source []byte
	Items  tree.List[markdownItem]
}

// markdownItem unifies nodes of the following kinds:
//
//   - markdownFileItem: an included Markdown file
//   - markdownGroupItem: a title without any files, grouping other items
type markdownItem interface {
	markdownItem()
}

type markdownFileItem struct {
	File  *goldast.File
	Item  *summary.LinkItem
	Title *markdownHeading
	Path  string

	Links    []*goldast.Link
	Headings []*markdownHeading
}

func (*markdownFileItem) markdownItem() {}

func (c *collector) collectFileItem(item *summary.LinkItem) (*markdownFileItem, error) {
	src, err := fs.ReadFile(c.FS, item.Target)
	if err != nil {
		return nil, err
	}

	f, err := goldast.Parse(c.Parser, item.Target, src)
	if err != nil {
		return nil, err
	}

	var (
		links    []*goldast.Link
		headings []*markdownHeading
		h1s      []*markdownHeading
	)
	err = goldast.Walk(f.AST, func(n *goldast.Any, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}

		if l, ok := goldast.Cast[*ast.Link](n); ok {
			links = append(links, l)
		} else if h, ok := goldast.Cast[*ast.Heading](n); ok {
			mh := c.newHeading(f, h)
			headings = append(headings, mh)
			if mh.Level() == 1 {
				h1s = append(h1s, mh)
			}
		} else {
			return ast.WalkContinue, nil
		}

		return ast.WalkSkipChildren, nil
	})
	if err != nil {
		return nil, err
	}

	mf := &markdownFileItem{
		File:     f,
		Item:     item,
		Path:     item.Target,
		Links:    links,
		Headings: headings,
	}
	if len(h1s) == 1 {
		mf.Title = h1s[0]
	}

	c.files[item.Target] = mf
	return mf, nil
}

type markdownGroupItem struct {
	*summary.TextItem

	ID string
}

func (*markdownGroupItem) markdownItem() {}

func (c *collector) collectGroupItem(item *summary.TextItem) *markdownGroupItem {
	id, _ := c.IDGen.GenerateID(item.Text)
	return &markdownGroupItem{
		TextItem: item,
		ID:       id,
	}
}

type markdownHeading struct {
	AST *goldast.Heading
	ID  string
}

func (c *collector) newHeading(f *goldast.File, h *goldast.Heading) *markdownHeading {
	id, _ := c.IDGen.GenerateID(string(h.Node.Text(f.Source)))
	return &markdownHeading{
		AST: h,
		ID:  id,
	}
}

func (h *markdownHeading) Level() int {
	return h.AST.Node.Level
}
