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
	group.Heading.AST.Node.Level += group.TOCDepth + t.sectionLevel

	// Replace "Foo" in the list with "[Foo](#foo)".
	item := group.TOCText
	parent := item.Node.Parent()

	link := ast.NewLink()
	link.Destination = []byte("#" + group.Heading.ID)
	parent.ReplaceChild(parent, item.Node, link)

	link.AppendChild(link, item.Node)
}

func (t *transformer) transformFile(f *markdownFileItem) {
	for _, h := range f.Headings {
		h.AST.Node.Level += f.TOCDepth + t.sectionLevel
	}

	if err := t.transformLink(".", f.TOCLink); err != nil {
		t.Log.Printf("%v:%v", t.tocFile.Position(f.TOCLink.Pos()), err)
	}

	dir := filepath.Dir(f.Path)
	for _, l := range f.Links {
		if err := t.transformLink(dir, l); err != nil {
			t.Log.Printf("%v:%v", f.File.Position(l.Pos()), err)
		}
	}

	doc := f.File.AST.Node
	if doc.ChildCount() > 0 {
		doc.InsertBefore(doc, doc.FirstChild(), f.Title.AST.Node)
	} else {
		doc.AppendChild(doc, f.Title.AST.Node)
	}
}

func (t *transformer) transformLink(from string, link *goldast.Node[*ast.Link]) error {
	u, err := url.Parse(string(link.Node.Destination))
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
	link.Node.Destination = []byte(u.String())

	return nil
}
