package main

import (
	"fmt"
	"log"
	"net/url"
	"path/filepath"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/mdreduce/internal/goldast"
	"go.abhg.dev/mdreduce/internal/summary"
	"go.abhg.dev/mdreduce/internal/tree"
)

type transformer struct {
	Files map[string]*markdownFile // path => file
	Log   *log.Logger
}

func (t *transformer) transformList(items tree.List[markdownItem]) {
	items.Walk(func(item markdownItem) error {
		t.transformItem(item)
		return nil
	})
}

func (t *transformer) transformItem(item markdownItem) {
	switch item := item.(type) {
	case *markdownFile:
		t.transformFile(item)
	case *markdownSection:
		t.transformSection(item)
	case *markdownTitle:
		t.transformTitle(item)
	}
}

func (t *transformer) transformSection(section *markdownSection) {
	for _, n := range section.Section.AST {
		goldast.Walk(n, func(n *goldast.Node[ast.Node], enter bool) (ast.WalkStatus, error) {
			if !enter {
				return ast.WalkContinue, nil
			}
			if l, ok := goldast.Cast[*ast.Link](n); ok {
				// TODO: put Positioner on markdownFile
				if err := t.transformLink(".", l); err != nil {
					t.Log.Printf("%v:%v",
						section.File.Positioner.Position(l.Pos()), err)
				}
				return ast.WalkSkipChildren, nil
			}
			return ast.WalkContinue, nil
		})
	}
}

func (t *transformer) transformTitle(title *markdownTitle) {
}

func (t *transformer) transformFile(file *markdownFile) {
	goldast.Walk(file.File.AST, func(n *goldast.Node[ast.Node], enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		if l, ok := goldast.Cast[*ast.Link](n); ok {
			// TODO: put Positioner on markdownFile
			if err := t.transformLink(file.Dir, l); err != nil {
				t.Log.Printf("%v:%v", file.File.Positioner.Position(l.Pos()), err)
			}
		} else if h, ok := goldast.Cast[*ast.Heading](n); ok {
			t.transformHeading(file.Item, h)
		} else {
			return ast.WalkContinue, nil
		}

		// TODO: rewrite image links

		return ast.WalkSkipChildren, nil
	})
}

func (t *transformer) transformHeading(item *summary.Item, h *goldast.Node[*ast.Heading]) {
	h.Node.Level += item.Depth
}

func (t *transformer) transformLink(from string, link *goldast.Node[*ast.Link]) error {
	u, err := url.Parse(string(link.Node.Destination))
	if err != nil || u.Scheme != "" || u.Host != "" {
		return nil // skip external and invalid links
	}

	if u.Path == "" {
		// TODO: rewrite relative header links
		return nil
	}

	dst := filepath.Join(from, u.Path)
	to, ok := t.Files[dst]
	if !ok {
		return fmt.Errorf("link to unknown file: %v", dst)
	}

	// TODO: handle no file title
	if u.Fragment == "" && to.Title != nil {
		link.Node.Destination = []byte("#" + to.Title.ID)
	}

	// TODO: if u.Fragment was not empty,
	// map it to a header in the original file,
	// and check if it has a new header ID.
	return nil
}
