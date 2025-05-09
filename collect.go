package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/toc"
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

	// Paths relative to root of fs.FS,
	// representing the stack of summary file embeds.
	// Used to detect cycles.
	Stack []string

	// Directory under FS to resolve relative paths from.
	// Must use '/' as the path separator.
	Dir string

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
	if c.idGen == nil {
		c.idGen = header.NewIDGen()
	}

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
//   - markdownEmbedItem: a request to embed another summary file
type markdownItem interface {
	markdownItem()
}

func (c *collector) collectItem(cursor tree.Cursor[stitch.Item]) (markdownItem, error) {
	item := cursor.Value()
	switch item := item.(type) {
	case *stitch.LinkItem:
		return c.collectLinkItem(item, cursor)

	case *stitch.EmbedItem:
		return c.collectEmbedItem(item, cursor)

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

	// Absorb indicates that the headings in this file
	// should be included in the parent TOC.
	Absorb bool
	TOC    *toc.TOC
}

func (*markdownFileItem) markdownItem() {}

func (c *collector) collectFileItem(item *stitch.LinkItem) (*markdownFileItem, error) {
	src, err := c.readFile(item.Target)
	if err != nil {
		return nil, err
	}

	ctx := parser.NewContext()
	f := goldast.Parse(c.Parser, item.Target, src, parser.WithContext(ctx))
	fidgen := header.NewIDGen()

	var options struct {
		// Headings included in the file
		// should be absorbed into the parent TOC.
		Absorb bool `yaml:"absorb"`
	}

	if data := frontmatter.Get(ctx); data != nil {
		if err := data.Decode(&options); err != nil {
			return nil, fmt.Errorf("bad frontmatter: %v", err)
		}
	}

	var (
		links      []*ast.Link
		images     []*ast.Image
		headings   []*markdownHeading
		h1s        []*markdownHeading
		rawHTMLs   []*ast.RawHTML
		htmlBlocks []*ast.HTMLBlock
	)
	headingsByOldID := make(map[string]*markdownHeading)
	// Error ignored because walker doesn't return errors.
	_ = goldast.Walk(f.AST, func(n ast.Node) error {
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
		Absorb:          options.Absorb,
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

	// If we're being absorbed, we'll need a TOC.
	if mf.Absorb {
		fileTOC, err := toc.Inspect(mf.File.AST, mf.File.Source, toc.Compact(true))
		if err != nil {
			return nil, err
		}
		mf.TOC = fileTOC
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

type markdownEmbedItem struct {
	Item        *stitch.EmbedItem
	Section     *markdownSection
	FilesByPath map[string]*markdownFileItem
	Heading     *markdownHeading
	SummaryFile *goldast.File

	src []byte
}

var _ markdownItem = (*markdownEmbedItem)(nil)

func (c *collector) collectEmbedItem(item *stitch.EmbedItem, cursor tree.Cursor[stitch.Item]) (*markdownEmbedItem, error) {
	if cursor.ChildCount() > 0 {
		return nil, errors.New("embed cannot have children")
	}

	embedPath := path.Join(c.Dir, item.Target)
	for _, p := range c.Stack {
		if p == embedPath {
			return nil, fmt.Errorf("embed cycle: %v", strings.Join(append(c.Stack, embedPath), " -> "))
		}
	}
	summaryStack := append(c.Stack, embedPath)

	src, err := c.readFile(item.Target)
	if err != nil {
		return nil, err
	}

	summaryFile := goldast.Parse(c.Parser, embedPath, src)
	summary, err := stitch.ParseSummary(summaryFile)
	if err != nil {
		return nil, err
	}

	coll, err := (&collector{
		Dir:    path.Join(c.Dir, path.Dir(item.Target)),
		Parser: c.Parser,
		FS:     c.FS,
		idGen:  c.idGen,
		Stack:  summaryStack,
	}).Collect(summaryFile.Info, summary)
	if err != nil {
		return nil, err
	}

	switch len(coll.Sections) {
	case 0:
		// Unreachable: ParseSummary always returns at least one section.
		return nil, errors.New("no sections found")
	case 1:
		// ok
	default:
		pos := summaryFile.Position(goldast.OffsetOf(coll.Sections[1].Title))
		return nil, fmt.Errorf("%v:unexpected section; expected only one section", pos)
	}

	section := coll.Sections[0]
	var heading *markdownHeading

	if h := section.Title; h != nil {
		// Ignore the heading level in the summary file.
		// It'll get whatever the depth of the embed is.
		h.Level = 1
		id, _ := c.idGen.GenerateID(string(goldast.Text(summaryFile.Source, h)))
		heading = &markdownHeading{
			AST: h,
			ID:  id,
			Lvl: h.Level,
		}

		// Unset the section title so it doesn't transform
		// heading levels.
		section.Title = nil
	} else {
		// The included file does not have a title.
		// Generate one from the TOC link.
		h := ast.NewHeading(1) // will be transformed
		h.AppendChild(h, ast.NewString([]byte(item.Text)))
		h.SetBlankPreviousLines(true)
		id, _ := c.idGen.GenerateID(item.Text)
		heading = &markdownHeading{
			AST: h,
			ID:  id,
			Lvl: h.Level,
		}
	}

	return &markdownEmbedItem{
		Item:        item,
		Section:     section,
		FilesByPath: coll.FilesByPath,
		SummaryFile: summaryFile,
		Heading:     heading,
	}, nil
}

func (*markdownEmbedItem) markdownItem() {}

type markdownHeading struct {
	AST ast.Node
	ID  string
	Lvl int

	// ID of the heading in the original file.
	OldID string
}

func (c *collector) newHeading(f *goldast.File, fgen *header.IDGen, h *ast.Heading) *markdownHeading {
	text := string(goldast.Text(f.Source, h))
	id, _ := c.idGen.GenerateID(text)
	oldID, _ := fgen.GenerateID(text)
	h.SetAttributeString("id", []byte(id)) // needed for toc.Inspect
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

// readFile reads a file from the underlying filesystem.
func (c *collector) readFile(p string) ([]byte, error) {
	p = path.Join(c.Dir, filepath.ToSlash(p))
	src, err := fs.ReadFile(c.FS, p)
	if err != nil {
		// If the error is because the path name was not valid,
		// it likely contains "." or ".." components,
		// or has a "/" at the start or end of the path.
		// Provide a hint to the user.
		if errors.Is(err, fs.ErrInvalid) {
			return nil, fmt.Errorf("invalid path %q; did you mean to use -unsafe?", p)
		}
		return nil, err
	}
	return src, nil
}
