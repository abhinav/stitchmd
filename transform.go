package main

import (
	"fmt"
	"log"
	"net/url"
	"path/filepath"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/mdreduce/internal/goldast"
)

type transformer struct {
	Files map[string]*markdownFileItem // path => file
	Log   *log.Logger

	// Level of the current section's header, if any.
	sectionLevel int

	tocFile *goldast.File
}

func (t *transformer) Transform(coll *markdownCollection) {
	t.tocFile = coll.TOCFile
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
		t.transformTitle(item)
	case *markdownFileItem:
		t.transformFile(item)
	default:
		panic(fmt.Sprintf("unknown item type: %T", item))
	}
}

func (t *transformer) transformTitle(group *markdownGroupItem) {
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
	if err := t.transformLink(".", f.TOCLink); err != nil {
		t.Log.Printf("%v:%v", t.tocFile.Position(f.TOCLink.Pos()), err)
	}

	dir := filepath.Dir(f.Path)
	for _, l := range f.Links {
		if err := t.transformLink(dir, l); err != nil {
			t.Log.Printf("%v:%v", f.File.Position(l.Pos()), err)
		}
	}

	for _, h := range f.Headings {
		h.AST.Node.Level += f.TOCDepth + t.sectionLevel
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
	to, ok := t.Files[dst]
	if !ok {
		return fmt.Errorf("link to unknown file: %v", dst)
	}

	if u.Fragment == "" && to.Title != nil {
		link.Node.Destination = []byte("#" + to.Title.ID)
	}

	return nil
}
