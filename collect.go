package main

import (
	"fmt"
	"io/fs"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/stitchmd/internal/goldast"
	"go.abhg.dev/stitchmd/internal/header"
	"go.abhg.dev/stitchmd/internal/pos"
	"go.abhg.dev/stitchmd/internal/stitch"
	"go.abhg.dev/stitchmd/internal/tree"
)

// collector loads all Markdown files in a TOC
// and builds an alternative, parsed representation of the files.
type collector struct {
	Parser parser.Parser
	FS     fs.FS
	IDGen  *header.IDGen

	files map[string]*markdownFileItem
}

type markdownCollection struct {
	TOCFile  *goldast.File
	Sections []*markdownSection

	// FilesByPath maps a Markdown file path to its parsed representation.
	FilesByPath map[string]*markdownFileItem
}

func (c *collector) Collect(f *goldast.File) (*markdownCollection, error) {
	c.files = make(map[string]*markdownFileItem)

	toc, err := stitch.ParseSummary(f)
	if err != nil {
		return nil, err
	}

	errs := pos.NewErrorList(f.Info)
	sections := make([]*markdownSection, len(toc.Sections))
	for i, sec := range toc.Sections {
		sections[i] = c.collectSection(errs, sec)
	}

	return &markdownCollection{
		TOCFile:     f,
		Sections:    sections,
		FilesByPath: c.files,
	}, errs.Err()
}

type markdownSection struct {
	Title    *goldast.Heading
	TOCItems *goldast.List
	Items    tree.List[markdownItem]
}

func (s *markdownSection) TitleLevel() int {
	if s.Title != nil {
		return s.Title.Node.Level
	}
	return 0
}

func (c *collector) collectSection(errs *pos.ErrorList, sec *stitch.Section) *markdownSection {
	items := tree.TransformList(sec.Items, func(item stitch.Item) markdownItem {
		i, err := c.collectItem(item)
		if err != nil {
			errs.Pushf(item.Pos(), "%v", err)
			return nil
		}
		return i
	})

	var title *goldast.Heading
	if sec.Title != nil {
		title = sec.Title.AST
	}

	return &markdownSection{
		Title:    title,
		TOCItems: sec.AST,
		Items:    items,
	}
}

// markdownItem unifies nodes of the following kinds:
//
//   - markdownFileItem: an included Markdown file
//   - markdownGroupItem: a title without any files, grouping other items
type markdownItem interface {
	markdownItem()
}

func (c *collector) collectItem(item stitch.Item) (markdownItem, error) {
	switch item := item.(type) {
	case *stitch.LinkItem:
		return c.collectFileItem(item)

	case *stitch.TextItem:
		return c.collectGroupItem(item), nil

	default:
		panic(fmt.Sprintf("unhandled item type %T", item))
	}
}

type markdownFileItem struct {
	// TOCLink is the original link in the TOC
	// that referenced the Markdown file.
	TOCLink *goldast.Link

	// Depth is the depth at which the link was found in the TOC.
	TOCDepth int

	// File is the parsed Markdown file
	// that the link points to.
	File *goldast.File

	// Title is the title of the Markdown file, if any.
	Title *markdownHeading

	// TOCTitle is the title for this file generated from the TOC text.
	// Only one of Title and TOCTitle is set.
	TOCTitle *markdownHeading

	// Path is the path to the Markdown file.
	Path string

	// Links holds all links that were found in the Markdown file.
	Links []*goldast.Link

	// Headings holds all headings that were found in the Markdown file.
	Headings []*markdownHeading

	HeadingsByOldID map[string]*markdownHeading
}

func (*markdownFileItem) markdownItem() {}

func (c *collector) collectFileItem(item *stitch.LinkItem) (*markdownFileItem, error) {
	src, err := fs.ReadFile(c.FS, item.Target)
	if err != nil {
		return nil, err
	}

	f, err := goldast.Parse(c.Parser, item.Target, src)
	if err != nil {
		return nil, err
	}

	fidgen := header.NewIDGen()

	var (
		links    []*goldast.Link
		headings []*markdownHeading
		h1s      []*markdownHeading
	)
	headingsByOldID := make(map[string]*markdownHeading)
	err = goldast.Walk(f.AST, func(n *goldast.Any) error {
		if l, ok := goldast.Cast[*ast.Link](n); ok {
			links = append(links, l)
		} else if h, ok := goldast.Cast[*ast.Heading](n); ok {
			mh := c.newHeading(f, fidgen, h)
			headings = append(headings, mh)
			if mh.Level() == 1 {
				h1s = append(h1s, mh)
			}
			headingsByOldID[mh.OldID] = mh
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	mf := &markdownFileItem{
		TOCLink:         item.AST,
		TOCDepth:        item.Depth,
		File:            f,
		Path:            item.Target,
		Links:           links,
		Headings:        headings,
		HeadingsByOldID: headingsByOldID,
	}

	// If the page has only one level 1 heading,
	// and it's the first element in the page,
	// then use it as the title.
	if len(h1s) == 1 && h1s[0].AST.PreviousSibling() == nil {
		mf.Title = h1s[0]
	} else {
		heading := ast.NewHeading(1)
		heading.AppendChild(
			heading,
			ast.NewString([]byte(item.Text)),
		)
		mf.TOCTitle = c.newHeading(f, fidgen, goldast.WithPos(heading, f.Pos))
	}

	c.files[item.Target] = mf
	return mf, nil
}

type markdownGroupItem struct {
	TOCText  *goldast.Text
	TOCDepth int
	Heading  *markdownHeading
}

func (*markdownGroupItem) markdownItem() {}

func (c *collector) collectGroupItem(item *stitch.TextItem) *markdownGroupItem {
	h := ast.NewHeading(1) // will be transformed
	h.AppendChild(h, ast.NewString([]byte(item.Text)))

	id, _ := c.IDGen.GenerateID(item.Text)
	return &markdownGroupItem{
		TOCText:  item.AST,
		TOCDepth: item.Depth,
		Heading: &markdownHeading{
			AST: goldast.WithPos(h, 0 /* never used */),
			ID:  id,
		},
	}
}

type markdownHeading struct {
	AST *goldast.Heading
	ID  string

	// ID of the heading in the original file.
	OldID string
}

func (c *collector) newHeading(f *goldast.File, fgen *header.IDGen, h *goldast.Heading) *markdownHeading {
	text := string(h.Node.Text(f.Source))
	id, _ := c.IDGen.GenerateID(text)
	oldID, _ := fgen.GenerateID(text)
	return &markdownHeading{
		AST:   h,
		ID:    id,
		OldID: oldID,
	}
}

func (h *markdownHeading) Level() int {
	return h.AST.Node.Level
}
