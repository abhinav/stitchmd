package main

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"path"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/container/ring"
	"go.abhg.dev/goldmark/toc"
	"go.abhg.dev/stitchmd/internal/goldast"
	"go.abhg.dev/stitchmd/internal/goldtext"
	"go.abhg.dev/stitchmd/internal/must"
	"go.abhg.dev/stitchmd/internal/rawhtml"
	"go.abhg.dev/stitchmd/internal/stitch"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type transformer struct {
	Log *log.Logger

	// /-separated relative path to the input directory
	// from wherever the output is going.
	InputRelPath string

	// Flat heading offset for all headings.
	Offset int

	SummaryFile *goldast.File

	// Heading offset for the current section.
	sectionOffset int

	filesByPath map[string]*markdownFileItem
}

func (t *transformer) Transform(coll *markdownCollection) {
	t.filesByPath = coll.FilesByPath
	for _, sec := range coll.Sections {
		offset := t.Offset
		if t := sec.Title; t != nil {
			offset += t.Level

			t.Level = offset
			if t.Level < 1 {
				t.Level = 1
			}
		}
		t.sectionOffset = offset

		err := sec.Items.Walk(func(item markdownItem) error {
			t.transformItem(item)
			return nil
		})

		// The function returns nil, not an error.
		// If this fails, something went seriously wrong.
		must.NotErrorf(err, "Error transforming section")
	}
}

func (t *transformer) transformItem(item markdownItem) {
	switch item := item.(type) {
	case *markdownGroupItem:
		t.transformGroup(item)
	case *markdownFileItem:
		t.transformFile(item)
	case *markdownExternalLinkItem:
		// Nothing to do.
	case *markdownEmbedItem:
		t.transformEmbed(item)
	default:
		panic(fmt.Sprintf("unknown item type: %T", item))
	}
}

func (t *transformer) transformEmbed(embed *markdownEmbedItem) {
	(&transformer{
		Log:          t.Log,
		InputRelPath: t.InputRelPath,
		Offset:       t.sectionOffset + embed.Item.ItemDepth() + 1,
		SummaryFile:  embed.SummaryFile,
	}).Transform(&markdownCollection{
		Sections:    []*markdownSection{embed.Section},
		FilesByPath: embed.FilesByPath,
	})

	embed.src = t.transformHeading(embed.src, embed.Item, embed.Heading)

	// Replace ![foo](foo.md) with [foo](#foo).
	item := embed.Item.AST
	parent := item.Parent()

	link := ast.NewLink()
	link.Destination = []byte("#" + embed.Heading.ID)
	parent.ReplaceChild(parent, item, link)
	for c := item.FirstChild(); c != nil; c = c.NextSibling() {
		link.AppendChild(link, c)
	}

	cloneSegment := func(seg text.Segment) text.Segment {
		bs := seg.Value(embed.SummaryFile.Source)

		seg.Start = len(t.SummaryFile.Source)
		t.SummaryFile.Source = append(t.SummaryFile.Source, bs...)
		seg.Stop = len(t.SummaryFile.Source)
		return seg
	}

	cloneSegments := func(segs *text.Segments) {
		for i := 0; i < segs.Len(); i++ {
			seg := segs.At(i)
			segs.Set(i, cloneSegment(seg))
		}
	}

	// We need to nest the TOC items of the embedded section
	// under the current section's list item.
	// However, those reference source positions in the other summary file.
	// They need to be appended to this summary file's source.
	//
	// This isn't super efficient because
	// it'll copy the bytes for each level of nesting.
	// But it's good enough for now.
	_ = ast.Walk(embed.Section.TOCItems, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n := n.(type) {
		case *ast.HTMLBlock:
			n.ClosureLine = cloneSegment(n.ClosureLine)
		case *ast.RawHTML:
			cloneSegments(n.Segments)
		case *ast.Text:
			n.Segment = cloneSegment(n.Segment)
		}

		if n.Type() == ast.TypeBlock {
			cloneSegments(n.Lines())
		}

		return ast.WalkContinue, nil
	})
	// Part of a bigger whole now. The row below must not be blank.
	embed.Section.TOCItems.SetBlankPreviousLines(false)
	parent.AppendChild(parent, embed.Section.TOCItems)
}

func (t *transformer) transformGroup(group *markdownGroupItem) {
	group.src = t.transformHeading(group.src, group.Item, group.Heading)

	// Replace "Foo" in the list with "[Foo](#foo)".
	item := group.Item.AST
	parent := item.Parent()

	link := ast.NewLink()
	link.Destination = []byte("#" + group.Heading.ID)
	parent.ReplaceChild(parent, item, link)

	link.AppendChild(link, item)
}

func (t *transformer) transformFile(f *markdownFileItem) {
	src := f.File.Source
	for _, h := range f.Headings {
		src = t.transformHeading(src, f.Item, h)
	}
	f.File.Source = src

	t.transformLink(".", f, f.Item.AST)

	fromPath := path.Dir(f.Path)
	for _, l := range f.Links {
		t.transformLink(fromPath, f, l)
	}

	for _, i := range f.Images {
		t.transformImage(fromPath, f, i)
	}

	src = f.File.Source
	for _, pair := range f.HTMLPairs {
		src = t.transformHTMLPair(src, fromPath, f, pair)
	}
	for _, h := range f.RawHTMLs {
		src = t.transformHTML(src, fromPath, f, h.Segments)
	}
	for _, h := range f.HTMLBlocks {
		src = t.transformHTML(src, fromPath, f, h.Lines())
	}
	f.File.Source = src

	// If the file requested absorbtion of headings, add them to the summary
	// as TOC items.
	if f.Absorb && len(f.Headings) > 0 {
		var parentListItem *ast.ListItem
		for item := f.Item.AST.Parent(); item != nil; item = item.Parent() {
			if li, ok := item.(*ast.ListItem); ok {
				parentListItem = li
				break
			}
		}
		if parentListItem == nil {
			t.Log.Panicf("could not find parent list item for %q", f.Path)
		}
		parentList := parentListItem.Parent().(*ast.List)

		marker := parentList.Marker
		var (
			renderItems func([]*toc.Item) ast.Node
			renderItem  func(*toc.Item) ast.Node
		)

		renderItems = func(items []*toc.Item) ast.Node {
			if len(items) == 0 {
				return nil
			}

			list := ast.NewList(marker)
			for _, item := range items {
				if listItem := renderItem(item); listItem != nil {
					list.AppendChild(list, listItem)
				}
			}
			return list
		}

		renderItem = func(item *toc.Item) ast.Node {
			title := ast.NewString(item.Title)
			title.SetRaw(true)

			link := ast.NewLink()
			link.Destination = append([]byte("#"), item.ID...)
			link.AppendChild(link, title)

			listItem := ast.NewListItem(0)
			listItem.AppendChild(listItem, link)

			if items := renderItems(item.Items); items != nil {
				listItem.AppendChild(listItem, items)
			}

			return listItem
		}

		if tocItems := renderItems(f.TOC.Items); tocItems != nil {
			parentListItem.AppendChild(parentListItem, tocItems)
		}

	}
	doc := f.File.AST
	if doc.ChildCount() > 0 {
		doc.InsertBefore(doc, doc.FirstChild(), f.Title.AST)
	} else {
		doc.AppendChild(doc, f.Title.AST)
	}
}

func (t *transformer) transformHTMLPair(src []byte, fromPath string, f *markdownFileItem, pair rawhtml.Pair) []byte {
	hn, err := pair.ParseHTML(src)
	if err != nil {
		return src // leave broken HTML alone
	}

	if changed := t.transformHTMLNode(fromPath, f, hn); !changed {
		return src // no changes
	}

	newPair, newSrc, err := rawhtml.PairFromHTML(hn, src)
	if err != nil {
		return src // leave broken HTML alone
	}

	src = newSrc
	pair.Open.Parent().ReplaceChild(pair.Open.Parent(), pair.Open, newPair.Open)
	pair.Close.Parent().ReplaceChild(pair.Close.Parent(), pair.Close, newPair.Close)
	return src
}

func (t *transformer) transformHTML(src []byte, fromPath string, f *markdownFileItem, segs *text.Segments) []byte {
	htmlBodies, err := rawhtml.ParseHTMLFragmentBodies(&goldtext.Reader{
		Source:   src,
		Segments: segs,
	})
	if err != nil {
		return src // Don't mess with broken HTML.
	}

	// Replace all links and images in the HTML.
	var (
		roots   []*html.Node
		changed bool
	)
	for _, body := range htmlBodies {
		for c := body.FirstChild; c != nil; c = c.NextSibling {
			roots = append(roots, c)
		}
	}

	var q ring.Q[*html.Node]
	for _, n := range roots {
		q.Push(n)
	}

	for !q.Empty() {
		n := q.Pop()
		changed = t.transformHTMLNode(fromPath, f, n) || changed
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			q.Push(c)
		}
	}

	if !changed {
		return src
	}

	var buff bytes.Buffer
	for _, n := range roots {
		if err := html.Render(&buff, n); err != nil {
			return src
		}
	}

	start := len(src)
	src = append(src, buff.Bytes()...)
	end := len(src)

	segs.Clear()
	segs.Append(text.NewSegment(start, end))

	return src
}

func (t *transformer) transformHTMLNode(fromPath string, f *markdownFileItem, n *html.Node) (changed bool) {
	switch n.Type {
	case html.ElementNode:
		switch n.DataAtom {
		case atom.A:
			for i, attr := range n.Attr {
				if attr.Key != "href" {
					continue
				}

				newURL := t.transformURL(fromPath, f, attr.Val)
				if newURL != attr.Val {
					n.Attr[i].Val = newURL
					changed = true
				}
			}

		case atom.Img:
			for i, attr := range n.Attr {
				if attr.Key != "src" {
					continue
				}

				newURL := t.transformURL(fromPath, f, attr.Val)
				if newURL != attr.Val {
					n.Attr[i].Val = newURL
					changed = true
				}
			}
		}
	}
	return changed
}

func (t *transformer) transformHeading(src []byte, item stitch.Item, h *markdownHeading) []byte {
	// GitHub doesn't support Heading attribute syntax.
	h.AST.RemoveAttributes()

	h.Lvl += item.ItemDepth() + t.sectionOffset
	if h.Lvl < 1 {
		h.Lvl = 1
	}

	if h.Lvl <= 6 {
		if hn, ok := h.AST.(*ast.Heading); ok {
			hn.Level = h.Lvl
			return src
		}
	}

	// This heading is too deep to represent in Markdown.
	// Replace it with a manual anchor and bold text.
	para := ast.NewParagraph()

	start := len(src)
	src = fmt.Appendf(src, "<a id=%q></a> ", h.ID)
	end := len(src)

	link := ast.NewRawHTML()
	link.Segments.Append(text.NewSegment(start, end))
	para.AppendChild(para, link)

	bold := ast.NewEmphasis(2)
	bold.AppendChild(bold, ast.NewString(h.AST.Text(src)))
	para.AppendChild(para, bold)

	if parent := h.AST.Parent(); parent != nil {
		parent.ReplaceChild(parent, h.AST, para)
	}
	h.AST = para

	return src
}

func (t *transformer) transformLink(fromPath string, f *markdownFileItem, link *ast.Link) {
	link.Destination = []byte(t.transformURL(fromPath, f, string(link.Destination)))
}

func (t *transformer) transformImage(fromPath string, f *markdownFileItem, image *ast.Image) {
	image.Destination = []byte(t.transformURL(fromPath, f, string(image.Destination)))
}

func (t *transformer) transformURL(fromPath string, f *markdownFileItem, toURL string) string {
	u, err := url.Parse(toURL)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return toURL
	}

	// Resolve the Path component of the URL to the destination file.
	to := f
	if u.Path != "" {
		dst := path.Join(fromPath, u.Path)
		var ok bool
		to, ok = t.filesByPath[dst]
		if !ok {
			// This is a relative path that does not point to a Markdown
			// file in the collection.
			// It may be a link to a file in the input directory.
			// Update the path and leave everything else as-is.
			u.Path = path.Join(t.InputRelPath, dst)
			return u.String()
		}
		u.Path = ""
	}

	if u.Fragment != "" {
		// If the fragment of a link to a Markdown file
		// is a known heading in that Markdown file,
		// use the new ID of that header.
		if h, ok := to.HeadingsByOldID[u.Fragment]; ok {
			u.Fragment = h.ID
		}
	} else {
		u.Fragment = to.Title.ID
	}
	return u.String()
}
