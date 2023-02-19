package main

import (
	"io/fs"
	"net/url"
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
			if item.Target == "" {
				return &markdownTitle{
					Text: item.Text,
					Item: item,
				}
			}

			mdf, err := c.loadFile(item)
			if err != nil {
				errs.Pushf(item.Pos, "%v", err)
				return nil
			}

			return mdf
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

type localReference[N ast.Node] struct {
	AST *goldast.Node[N]
	URL *url.URL
}
type markdownFile struct {
	Dir  string
	Path string
	File *goldast.File
	Item *summary.Item

	// Level 1 heading acting as the title for the document.
	// This is non-nil only if the document has exactly one such heading.
	Title *markdownHeading

	// Local links and images in the file.
	LocalLinks  []*localReference[*ast.Link]
	LocalImages []*localReference[*ast.Image]
	Headings    []*markdownHeading
}

func (*markdownFile) markdownItem() {}

func (c *collector) loadFile(item *summary.Item) (*markdownFile, error) {
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

	var (
		links    []*localReference[*ast.Link]
		images   []*localReference[*ast.Image]
		headings []*markdownHeading

		// Level 1 headings in the file.
		h1s []*markdownHeading
	)

	err = goldast.Walk(f.AST, func(n *goldast.Node[ast.Node], enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}

		if l, ok := goldast.Cast[*ast.Link](n); ok {
			u, err := url.Parse(string(l.Node.Destination))
			if err != nil || u.Scheme != "" || u.Host != "" {
				return ast.WalkContinue, nil // skip external and invalid links
			}
			links = append(links, &localReference[*ast.Link]{
				AST: l,
				URL: u,
			})
		} else if i, ok := goldast.Cast[*ast.Image](n); ok {
			u, err := url.Parse(string(i.Node.Destination))
			if err != nil || u.Scheme != "" || u.Host != "" {
				return ast.WalkContinue, nil // skip external and invalid links
			}
			images = append(images, &localReference[*ast.Image]{
				AST: i,
				URL: u,
			})
		} else if h, ok := goldast.Cast[*ast.Heading](n); ok {
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
		} else {
			return ast.WalkContinue, nil
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
		Dir:         filepath.Dir(item.Target),
		Path:        item.Target,
		File:        f,
		Item:        item,
		Title:       title,
		LocalLinks:  links,
		LocalImages: images,
		Headings:    headings,
	}
	c.files[item.Target] = mf
	return mf, nil
}

type markdownTitle struct {
	Text string
	Item *summary.Item
}

func (*markdownTitle) markdownItem() {}
