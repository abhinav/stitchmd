package main

import (
	"io/fs"
	"path/filepath"

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
				File:    f,
				Section: sec,
			},
			List: items,
		}
	}

	return sections, errs.Err()
}

func (c *collector) collectItem(item *summary.Item) (markdownItem, error) {
	if item.Target == "" {
		return &markdownTitle{
			Text: item.Text,
			Item: item,
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
		headings []*markdownHeading

		// Level 1 headings in the file.
		h1s []*markdownHeading
	)

	err = goldast.Walk(f.AST, func(n *goldast.Node[ast.Node], enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}

		h, ok := goldast.Cast[*ast.Heading](n)
		if !ok {
			return ast.WalkContinue, nil
		}

		title := n.Node.Text(src)
		slug, _ := c.IDGen.GenerateID(string(title))
		// if !ok {
		// 	// TODO: do we need to handle this?
		// }
		heading := &markdownHeading{
			AST:   h,
			ID:    slug,
			Level: h.Node.Level,
		}
		headings = append(headings, heading)
		if heading.Level == 1 {
			h1s = append(h1s, heading)
		}
		return ast.WalkSkipChildren, nil
	})
	if err != nil {
		return nil, err
	}

	var title *markdownHeading
	if len(h1s) == 1 {
		title = h1s[0]
	}

	mf := &markdownFile{
		Dir:   filepath.Dir(item.Target),
		Path:  item.Target,
		File:  f,
		Item:  item,
		Title: title,
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
	File    *goldast.File
	Section *summary.Section
}

func (*markdownSection) markdownItem() {}

type markdownHeading struct {
	ID    string
	AST   *goldast.Node[*ast.Heading]
	Level int
}

type markdownFile struct {
	Dir  string
	Path string
	File *goldast.File
	Item *summary.Item

	// Level 1 heading acting as the title for the document.
	// This is non-nil only if the document has exactly one such heading.
	Title *markdownHeading
}

func (*markdownFile) markdownItem() {}

type markdownTitle struct {
	Text string
	Item *summary.Item
}

func (*markdownTitle) markdownItem() {}
