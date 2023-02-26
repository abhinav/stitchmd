package main

import (
	"fmt"
	"io"
	"log"

	mdfmt "github.com/Kunde21/markdownfmt/v3/markdown"
)

type generator struct {
	idx int

	W        io.Writer
	Renderer *mdfmt.Renderer
	Log      *log.Logger
	NoTOC    bool
}

func (g *generator) Generate(src []byte, coll *markdownCollection) error {
	for _, sec := range coll.Sections {
		if err := g.renderSection(src, sec); err != nil {
			return err
		}
		if err := sec.Items.Walk(g.renderItem); err != nil {
			return err
		}
	}
	return nil
}

func (g *generator) renderSection(src []byte, sec *markdownSection) error {
	empty := true
	if t := sec.Title; t != nil {
		empty = false
		if err := g.Renderer.Render(g.W, src, t); err != nil {
			return err
		}
	}

	if !g.NoTOC {
		empty = false
		if err := g.Renderer.Render(g.W, src, sec.TOCItems); err != nil {
			return err
		}
	}

	if !empty {
		io.WriteString(g.W, "\n\n")
	}
	return nil
}

func (g *generator) renderItem(item markdownItem) error {
	if g.idx > 0 {
		io.WriteString(g.W, "\n")
	}
	g.idx++

	switch item := item.(type) {
	case *markdownGroupItem:
		return g.renderGroupItem(item)

	case *markdownFileItem:
		return g.renderFileItem(item)

	default:
		panic(fmt.Sprintf("unknown markdown item type %T", item))
	}
}

func (g *generator) renderGroupItem(group *markdownGroupItem) error {
	if err := g.Renderer.Render(g.W, nil, group.Heading.AST); err != nil {
		return err
	}
	io.WriteString(g.W, "\n")
	return nil
}

func (g *generator) renderFileItem(file *markdownFileItem) error {
	return g.Renderer.Render(g.W, file.File.Source, file.File.AST)
}
