package main

import (
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

func (c *collector) Collect(f *goldast.File) (tree.List[markdownItem], error) {
	toc, err := summary.Parse(f)
	if err != nil {
		return nil, err
	}

	errs := pos.NewErrorList(f.Positioner)
	sections := make(tree.List[markdownItem], len(toc.Sections))
	for i, sec := range toc.Sections {
		items := tree.TransformList(sec.Items, func(item *summary.Item) markdownItem {
			i, err := c.collectItem(item)
			if err != nil {
				errs.Pushf(item.Pos, "%v", err)
				return nil
			}
			return i
		})

		sections[i] = &tree.Node[markdownItem]{
			Value: &markdownSection{
				AST:        sec.AST,
				Positioner: f.Positioner,
				Source:     f.Source,
			},
			List: items,
		}
	}

	return sections, errs.Err()
}

func (c *collector) collectItem(item *summary.Item) (markdownItem, error) {
	if item.Target == "" {
		return &markdownTitle{
			Text:  item.Text,
			Depth: item.Depth,
		}, nil
	}

	src, err := fs.ReadFile(c.FS, item.Target)
	if err != nil {
		return nil, err
	}

	// TODO: -r/--recurse flag
	// to extract summary from top of included files?

	f, err := goldast.Parse(c.Parser, item.Target, src)
	if err != nil {
		return nil, err
	}

	// TODO: title should be the first thing in the document

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

		// if !ok {
		// 	// TODO: do we need to handle this?
		// }
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
		File:  f,
		Path:  item.Target,
		Depth: item.Depth,
	}
	if len(h1s) == 1 {
		mf.ID = h1IDs[0]
	}

	c.files[item.Target] = mf
	return mf, nil
}

// markdownItem unifies nodes of the following kinds:
//
//   - markdownSection: a section in the TOC.
//     It won't have other sections as children.
//   - markdownFile: an included Markdown file
//   - markdownTitle: a lone title without any file associated with it
type markdownItem interface {
	markdownItem()
}

// TODO: Maybe this AST should be represented in terms of summary.Node somehow.

type markdownSection struct {
	// TODO: turn markdownSection AST into a single node
	AST        []*goldast.Any
	Positioner pos.Positioner
	Source     []byte
}

func (*markdownSection) markdownItem() {}

type markdownFile struct {
	*goldast.File

	Path  string
	Depth int
	ID    string
}

func (*markdownFile) markdownItem() {}

type markdownTitle struct {
	Text  string
	Depth int
}

func (*markdownTitle) markdownItem() {}
