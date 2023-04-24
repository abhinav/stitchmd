package main

import (
	"fmt"
	"log"
	"net/url"
	"path"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/stitchmd/internal/stitch"
)

type transformer struct {
	Log *log.Logger

	// /-separated relative path to the input directory
	// from wherever the output is going.
	InputRelPath string

	// Flat heading offset for all headings.
	Offset int

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

		sec.Items.Walk(func(item markdownItem) error {
			t.transformItem(item)
			return nil
		})
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
	default:
		panic(fmt.Sprintf("unknown item type: %T", item))
	}
}

func (t *transformer) transformGroup(group *markdownGroupItem) {
	t.transformHeading(group.Item, group.Heading.AST)

	// Replace "Foo" in the list with "[Foo](#foo)".
	item := group.Item.AST
	parent := item.Parent()

	link := ast.NewLink()
	link.Destination = []byte("#" + group.Heading.ID)
	parent.ReplaceChild(parent, item, link)

	link.AppendChild(link, item)
}

func (t *transformer) transformFile(f *markdownFileItem) {
	for _, h := range f.Headings {
		t.transformHeading(f.Item, h.AST)
	}

	t.transformLink(".", f, f.Item.AST)

	fromPath := path.Dir(f.Path)
	for _, l := range f.Links {
		t.transformLink(fromPath, f, l)
	}

	for _, i := range f.Images {
		t.transformImage(fromPath, f, i)
	}

	doc := f.File.AST
	if doc.ChildCount() > 0 {
		doc.InsertBefore(doc, doc.FirstChild(), f.Title.AST)
	} else {
		doc.AppendChild(doc, f.Title.AST)
	}
}

func (t *transformer) transformHeading(item stitch.Item, h *ast.Heading) {
	h.Level += item.ItemDepth() + t.sectionOffset
	if h.Level < 1 {
		h.Level = 1
	}
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
