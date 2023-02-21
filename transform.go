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
	Files map[string]*markdownFile // path => file
	Log   *log.Logger

	// Level of the current section's header, if any.
	sectionLevel int
}

func (t *transformer) transformList(sections []*markdownSection) {
	for _, sec := range sections {
		t.sectionLevel = sec.TitleLevel()
		sec.Items.Walk(func(item markdownItem) error {
			t.transformItem(item)
			return nil
		})
	}
}

func (t *transformer) transformItem(item markdownItem) {
	switch item := item.(type) {
	case *markdownTitle:
		t.transformTitle(item)
	case *markdownFile:
		t.transformFile(item)
	default:
		panic(fmt.Sprintf("unknown item type: %T", item))
	}
}

func (t *transformer) transformTitle(title *markdownTitle) {
	title.Depth += t.sectionLevel

	// Replace "Foo" in the list with "[Foo](#foo)".
	item := title.AST
	parent := item.Node.Parent()

	link := ast.NewLink()
	link.Destination = []byte("#" + title.TitleID)
	parent.ReplaceChild(parent, item.Node, link)

	link.AppendChild(link, item.Node)
}

func (t *transformer) transformFile(f *markdownFile) {
	_ = t.transformLink(".", f.Item.AST) // TODO: handle error
	dir := filepath.Dir(f.Path)
	file := f.File
	goldast.Walk(file.AST, func(n *goldast.Any, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		if l, ok := goldast.Cast[*ast.Link](n); ok {
			if err := t.transformLink(dir, l); err != nil {
				t.Log.Printf("%v:%v", file.Position(l.Pos()), err)
			}
		} else if h, ok := goldast.Cast[*ast.Heading](n); ok {
			h.Node.Level += f.Item.ItemDepth() + t.sectionLevel
		} else {
			return ast.WalkContinue, nil
		}

		return ast.WalkSkipChildren, nil
	})
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

	if u.Fragment == "" && to.TitleID != "" {
		link.Node.Destination = []byte("#" + to.TitleID)
	}

	return nil
}
