package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/stitchmd/internal/goldast"
	"go.abhg.dev/stitchmd/internal/header"
	"go.abhg.dev/stitchmd/internal/rawhtml"
	"go.abhg.dev/stitchmd/internal/stitch"
	"go.abhg.dev/stitchmd/internal/tree"
)

// collector loads all Markdown files in a TOC
// and builds an alternative, parsed representation of the files.
type collector struct {
	Parser parser.Parser // required
	FS     fs.FS         // required

	idGen *header.IDGen
	files map[string]*markdownFileItem
}

type markdownCollection struct {
	Sections []*markdownSection

	// FilesByPath maps a Markdown file path to its parsed representation.
	// The path is /-separated, regardless of the OS.
	FilesByPath map[string]*markdownFileItem
}

func (c *collector) Collect(info goldast.Positioner, toc *stitch.Summary) (*markdownCollection, error) {
	c.files = make(map[string]*markdownFileItem)
	c.idGen = header.NewIDGen()

	errs := goldast.NewErrorList(info)
	sections := make([]*markdownSection, len(toc.Sections))
	for i, sec := range toc.Sections {
		sections[i] = c.collectSection(errs, sec)
	}

	return &markdownCollection{
		Sections:    sections,
		FilesByPath: c.files,
	}, errs.Err()
}

type markdownSection struct {
	Title    *ast.Heading
	TOCItems *ast.List
	Items    tree.List[markdownItem]
}

func (c *collector) collectSection(errs *goldast.ErrorList, sec *stitch.Section) *markdownSection {
	items := tree.TransformList(sec.Items, func(cursor tree.Cursor[stitch.Item]) markdownItem {
		i, err := c.collectItem(cursor)
		if err != nil {
			errs.Pushf(cursor.Value().Node(), "%v", err)
			return nil
		}
		return i
	})

	var title *ast.Heading
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
//   - markdownExternalLinkItem: an external link
type markdownItem interface {
	markdownItem()
}

func (c *collector) collectItem(cursor tree.Cursor[stitch.Item]) (markdownItem, error) {
	item := cursor.Value()
	switch item := item.(type) {
	case *stitch.LinkItem:
		return c.collectLinkItem(item, cursor)

	case *stitch.TextItem:
		return c.collectGroupItem(item), nil

	default:
		panic(fmt.Sprintf("unhandled item type %T", item))
	}
}

func (c *collector) collectLinkItem(item *stitch.LinkItem, cursor tree.Cursor[stitch.Item]) (markdownItem, error) {
	u, err := url.Parse(item.Target)
	if err == nil && u.Host != "" {
		if cursor.ChildCount() > 0 {
			return nil, errors.New("external link cannot have children")
		}
		return &markdownExternalLinkItem{
			Item: item,
		}, nil
	}

	return c.collectFileItem(item)
}

// markdownExternalLinkItem is a marker for external links
// in the summary.
type markdownExternalLinkItem struct {
	Item *stitch.LinkItem
}

func (*markdownExternalLinkItem) markdownItem() {}

type markdownFileItem struct {
	// Path is the /-separated path to the Markdown file.
	Path string

	// Item is the original link in the TOC
	// that referenced the Markdown file.
	Item *stitch.LinkItem

	// Title is the title of the Markdown file, if any.
	Title *markdownHeading

	// File is the parsed Markdown file
	// that the link points to.
	File *goldast.File

	// Links holds all links that were found in the Markdown file.
	Links []*ast.Link

	// Images holds all images that were found in the Markdown file.
	Images []*ast.Image

	// Headings holds all headings that were found in the Markdown file.
	Headings []*markdownHeading

	// HeadingsByOldID maps IDs of headings, as interpreted in isolation.
	// The IDs will change once interpreted as part of the combined document.
	HeadingsByOldID map[string]*markdownHeading

	HTMLPairs  rawhtml.Pairs
	RawHTMLs   []*ast.RawHTML
	HTMLBlocks []*ast.HTMLBlock
}

func (*markdownFileItem) markdownItem() {}

func (c *collector) collectFileItem(item *stitch.LinkItem) (*markdownFileItem, error) {
	src, err := fs.ReadFile(c.FS, item.Target)
	if err != nil {
		// If the error is because the path name was not valid,
		// it likely contains "." or ".." components,
		// or has a "/" at the start or end of the path.
		// Provide a hint to the user.
		if errors.Is(err, fs.ErrInvalid) {
			return nil, fmt.Errorf("invalid path name %q; did you mean to use -unsafe?", item.Target)
		}

		return nil, err
	}

	ctx := parser.NewContext()
	f := goldast.Parse(c.Parser, item.Target, src, parser.WithContext(ctx))
	fidgen := header.NewIDGen()

	var (
		links      []*ast.Link
		images     []*ast.Image
		headings   []*markdownHeading
		h1s        []*markdownHeading
		rawHTMLs   []*ast.RawHTML
		htmlBlocks []*ast.HTMLBlock
	)
	headingsByOldID := make(map[string]*markdownHeading)
	err = goldast.Walk(f.AST, func(n ast.Node) error {
		switch n := n.(type) {
		case *ast.Link:
			links = append(links, n)
		case *ast.Image:
			images = append(images, n)
		case *ast.Heading:
			mh := c.newHeading(f, fidgen, n)
			headings = append(headings, mh)
			if mh.Level() == 1 {
				h1s = append(h1s, mh)
			}
			headingsByOldID[mh.OldID] = mh
		case *ast.RawHTML:
			rawHTMLs = append(rawHTMLs, n)
		case *ast.HTMLBlock:
			htmlBlocks = append(htmlBlocks, n)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	mf := &markdownFileItem{
		Path:            item.Target,
		Item:            item,
		File:            f,
		Links:           links,
		Images:          images,
		Headings:        headings,
		HeadingsByOldID: headingsByOldID,
		HTMLPairs:       rawhtml.GetPairs(ctx),
		RawHTMLs:        rawHTMLs,
		HTMLBlocks:      htmlBlocks,
	}

	// If the page has only one level 1 heading,
	// and it's the first element in the page,
	// then use it as the title.
	if len(h1s) == 1 && h1s[0].AST.PreviousSibling() == nil {
		mf.Title = h1s[0]
		f.AST.RemoveChild(f.AST, h1s[0].AST)
	} else {
		// The included file does not have a title.
		// Generate one from the TOC link.
		heading := ast.NewHeading(1)
		heading.AppendChild(
			heading,
			ast.NewString([]byte(item.Text)),
		)
		heading.SetBlankPreviousLines(true)
		mf.Title = c.newHeading(f, fidgen, heading)

		// Push all existing headers down one level
		// to make room for the new title
		// if any of them is a level 1 header.
		if len(h1s) > 0 {
			for _, h := range mf.Headings {
				h.Lvl++
			}
		}
		mf.Headings = append([]*markdownHeading{mf.Title}, mf.Headings...)
	}

	c.files[item.Target] = mf
	return mf, nil
}

type markdownGroupItem struct {
	Item    *stitch.TextItem
	Heading *markdownHeading

	src []byte
}

func (*markdownGroupItem) markdownItem() {}

func (c *collector) collectGroupItem(item *stitch.TextItem) *markdownGroupItem {
	h := ast.NewHeading(1) // will be transformed
	h.AppendChild(h, ast.NewString([]byte(item.Text)))
	h.SetBlankPreviousLines(true)

	id, _ := c.idGen.GenerateID(item.Text)
	return &markdownGroupItem{
		Item: item,
		Heading: &markdownHeading{
			AST: h,
			ID:  id,
			Lvl: h.Level,
		},
	}
}

type markdownHeading struct {
	AST ast.Node
	ID  string
	Lvl int

	// ID of the heading in the original file.
	OldID string
}

func (c *collector) newHeading(f *goldast.File, fgen *header.IDGen, h *ast.Heading) *markdownHeading {
	text := string(h.Text(f.Source))
	id, _ := c.idGen.GenerateID(text)
	oldID, _ := fgen.GenerateID(text)
	return &markdownHeading{
		AST:   h,
		ID:    id,
		OldID: oldID,
		Lvl:   h.Level,
	}
}

func (h *markdownHeading) Level() int {
	return h.Lvl
}
