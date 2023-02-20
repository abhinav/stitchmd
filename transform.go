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

	// Level of the current section's header, if any.
	sectionLevel int
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
	t.sectionLevel = section.Level
	for _, n := range section.AST {
		goldast.Walk(n, func(n *goldast.Any, enter bool) (ast.WalkStatus, error) {
			if !enter {
				return ast.WalkContinue, nil
			}
			if l, ok := goldast.Cast[*ast.Link](n); ok {
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
	title.Depth += t.sectionLevel
}

func (t *transformer) transformFile(file *markdownFile) {
	dir := filepath.Dir(file.Path)
	goldast.Walk(file.AST, func(n *goldast.Any, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		if l, ok := goldast.Cast[*ast.Link](n); ok {
			if err := t.transformLink(dir, l); err != nil {
				t.Log.Printf("%v:%v", file.Position(l.Pos()), err)
			}
		} else if h, ok := goldast.Cast[*ast.Heading](n); ok {
			h.Node.Level += file.Depth + t.sectionLevel
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

	if u.Fragment == "" && to.ID != "" {
		link.Node.Destination = []byte("#" + to.ID)
	}

	return nil
}
