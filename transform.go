package main

import (
	"fmt"
	"log"
	"net/url"
	"path"
	"path/filepath"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/stitchmd/internal/goldast"
)

type transformer struct {
	Log *log.Logger

	// Relative path to the input directory
	// from wherever the output is going.
	InputRelPath string

	// Level of the current section's header, if any.
	sectionLevel int

	filesByPath map[string]*markdownFileItem
	tocFile     *goldast.File
}

func (t *transformer) Transform(coll *markdownCollection) {
	t.tocFile = coll.TOCFile
	t.filesByPath = coll.FilesByPath
	for _, sec := range coll.Sections {
		t.sectionLevel = sec.TitleLevel()
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
	default:
		panic(fmt.Sprintf("unknown item type: %T", item))
	}
}

func (t *transformer) transformGroup(group *markdownGroupItem) {
	group.Heading.AST.Level += group.Item.Depth + t.sectionLevel

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
		h.AST.Level += f.Item.Depth + t.sectionLevel
	}

	t.transformLink(".", f, f.Item.AST)

	dir := filepath.Dir(f.Path)
	for _, l := range f.Links {
		t.transformLink(dir, f, l)
	}

	for _, i := range f.Images {
		t.transformImage(dir, f, i)
	}

	doc := f.File.AST
	if doc.ChildCount() > 0 {
		doc.InsertBefore(doc, doc.FirstChild(), f.Title.AST)
	} else {
		doc.AppendChild(doc, f.Title.AST)
	}
}

func (t *transformer) transformLink(fromDir string, f *markdownFileItem, link *ast.Link) {
	link.Destination = []byte(t.transformURL(fromDir, f, string(link.Destination)))
}

func (t *transformer) transformImage(fromDir string, f *markdownFileItem, image *ast.Image) {
	image.Destination = []byte(t.transformURL(fromDir, f, string(image.Destination)))
}

func (t *transformer) transformURL(fromDir string, f *markdownFileItem, toURL string) string {
	u, err := url.Parse(toURL)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return toURL
	}

	// Resolve the Path component of the URL to the destination file.
	to := f
	if u.Path != "" {
		dst := filepath.Join(fromDir, u.Path)
		var ok bool
		to, ok = t.filesByPath[dst]
		if !ok {
			// This is a relative path that does not point to a Markdown
			// file in the collection.
			// It may be a link to a file in the input directory.
			// Update the path and leave everything else as-is.
			u.Path = path.Join(filepath.ToSlash(t.InputRelPath), dst)
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
