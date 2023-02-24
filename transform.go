package main

import (
	"fmt"
	"log"
	"net/url"
	"path/filepath"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/stitchmd/internal/goldast"
)

type transformer struct {
	Log *log.Logger

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

	if err := t.transformLink(".", f.Item.AST); err != nil {
		offset := goldast.OffsetOf(f.Item.Node())
		t.Log.Printf("%v:%v", t.tocFile.Position(offset), err)
	}

	dir := filepath.Dir(f.Path)
	for _, l := range f.Links {
		if err := t.transformLink(dir, l); err != nil {
			t.Log.Printf("%v:%v", f.File.Position(goldast.OffsetOf(l)), err)
		}
	}

	doc := f.File.AST
	if doc.ChildCount() > 0 {
		doc.InsertBefore(doc, doc.FirstChild(), f.Title.AST)
	} else {
		doc.AppendChild(doc, f.Title.AST)
	}
}

func (t *transformer) transformLink(from string, link *ast.Link) error {
	u, err := url.Parse(string(link.Destination))
	if err != nil || u.Scheme != "" || u.Host != "" {
		return nil // skip external and invalid links
	}

	if u.Path == "" {
		return nil
	}

	dst := filepath.Join(from, u.Path)
	to, ok := t.filesByPath[dst]
	if !ok {
		return fmt.Errorf("link to unknown file: %v", dst)
	}

	u.Path = ""
	if u.Fragment != "" {
		// If the fragment of a link to another Markdown file
		// is a known heading in that Markdown file,
		// use the new ID of that header.
		if h, ok := to.HeadingsByOldID[u.Fragment]; ok {
			u.Fragment = h.ID
		}
	} else {
		u.Fragment = to.Title.ID
	}
	link.Destination = []byte(u.String())

	return nil
}
