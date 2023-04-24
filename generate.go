package main

import (
	"fmt"
	"io"
	"log"

	mdfmt "github.com/Kunde21/markdownfmt/v3/markdown"
)

type generator struct {
	headingIdx int

	Preface  []byte
	W        io.Writer       // required
	Renderer *mdfmt.Renderer // required
	Log      *log.Logger
	NoTOC    bool
}

func (g *generator) Generate(src []byte, coll *markdownCollection) error {
	if _, err := g.W.Write(g.Preface); err != nil {
		return err
	}

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

func (g *generator) addHeadingSep() {
	if g.headingIdx > 0 {
		io.WriteString(g.W, "\n")
	}
	g.headingIdx++
}

func (g *generator) renderItem(item markdownItem) error {
	switch item := item.(type) {
	case *markdownGroupItem:
		return g.renderGroupItem(item)

	case *markdownFileItem:
		return g.renderFileItem(item)

	case *markdownExternalLinkItem:
		// Nothing to do.
		// The item was already rendered in the TOC.
		return nil

	default:
		panic(fmt.Sprintf("unknown markdown item type %T", item))
	}
}

func (g *generator) renderGroupItem(group *markdownGroupItem) error {
	g.addHeadingSep()
	if err := g.Renderer.Render(g.W, nil, group.Heading.AST); err != nil {
		return err
	}
	io.WriteString(g.W, "\n")
	return nil
}

func (g *generator) renderFileItem(file *markdownFileItem) error {
	g.addHeadingSep()
	return g.Renderer.Render(g.W, file.File.Source, file.File.AST)
}
