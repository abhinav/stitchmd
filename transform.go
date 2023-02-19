package main

import (
	"fmt"
	"log"
	"net/url"
	"path/filepath"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/mdreduce/internal/goldast"
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
	for _, n := range section.AST {
		// TODO: turn non-Link nodes into Links to their respective
		// sections.
		goldast.Walk(n, func(n *goldast.Any, enter bool) (ast.WalkStatus, error) {
			if !enter {
				return ast.WalkContinue, nil
			}
			if l, ok := goldast.Cast[*ast.Link](n); ok {
				// TODO: put Positioner on markdownFile
				if err := t.transformLink(".", l); err != nil {
					t.Log.Printf("%v:%v",
						section.Positioner.Position(l.Pos()), err)
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
	dir := filepath.Dir(file.Path)
	goldast.Walk(file.AST, func(n *goldast.Any, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		if l, ok := goldast.Cast[*ast.Link](n); ok {
			// TODO: put Positioner on markdownFile
			if err := t.transformLink(dir, l); err != nil {
				t.Log.Printf("%v:%v", file.Position(l.Pos()), err)
			}
		} else if h, ok := goldast.Cast[*ast.Heading](n); ok {
			h.Node.Level += file.Depth
		} else {
			return ast.WalkContinue, nil
		}

		// TODO: rewrite image links

		return ast.WalkSkipChildren, nil
	})
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
	if u.Fragment == "" && to.ID != "" {
		link.Node.Destination = []byte("#" + to.ID)
	}

	// TODO: if u.Fragment was not empty,
	// map it to a header in the original file,
	// and check if it has a new header ID.
	return nil
}
