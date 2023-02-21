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

	files map[string]*markdownFile // path => file
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
		return c.collectLinkItem(item)

	case *summary.TextItem:
		id, _ := c.IDGen.GenerateID(item.Text)
		return &markdownTitle{
			TextItem: item,
			TitleID:  id,
		}, nil

	default:
		panic(fmt.Sprintf("unhandled item type %T", item))
	}
}

func (c *collector) collectLinkItem(item *summary.LinkItem) (*markdownFile, error) {
	src, err := fs.ReadFile(c.FS, item.Target)
	if err != nil {
		return nil, err
	}

	f, err := goldast.Parse(c.Parser, item.Target, src)
	if err != nil {
		return nil, err
	}

	var (
		h1s   []*goldast.Heading
		h1IDs []string
		// inv: len(h1s) == len(h1IDs)
	)
	err = goldast.Walk(f.AST, func(n *goldast.Any, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}

		h, ok := goldast.Cast[*ast.Heading](n)
		if !ok {
			return ast.WalkContinue, nil
		}

		// We generate IDs even though we don't use them
		// to ensure that ID collisions are handled correctly.
		title := n.Node.Text(src)
		id, _ := c.IDGen.GenerateID(string(title))
		if h.Node.Level == 1 {
			h1s = append(h1s, h)
			h1IDs = append(h1IDs, id)
		}
		return ast.WalkSkipChildren, nil
	})
	if err != nil {
		return nil, err
	}

	mf := &markdownFile{
		File: f,
		Item: item,
		Path: item.Target,
	}
	if len(h1s) == 1 {
		mf.TitleID = h1IDs[0]
	}

	c.files[item.Target] = mf
	return mf, nil
}

type markdownSection struct {
	*summary.Section

	Source []byte
	Items  tree.List[markdownItem]
}

// markdownItem unifies nodes of the following kinds:
//
//   - markdownFile: an included Markdown file
//   - markdownTitle: a lone title without any file associated with it
type markdownItem interface {
	markdownItem()
}

type markdownFile struct {
	File    *goldast.File
	Item    *summary.LinkItem
	Path    string
	TitleID string
}

func (*markdownFile) markdownItem() {}

type markdownTitle struct {
	*summary.TextItem

	TitleID string
}

func (*markdownTitle) markdownItem() {}
