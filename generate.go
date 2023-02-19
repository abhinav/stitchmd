package main

import (
	"fmt"
	"io"
	"log"
	"path/filepath"

	mdfmt "github.com/Kunde21/markdownfmt/v3/markdown"
	"github.com/yuin/goldmark/ast"
)

type generator struct {
	idx int

	W           io.Writer
	Renderer    *mdfmt.Renderer
	Log         *log.Logger
	FilesByPath map[string]*markdownFile
}

func (g *generator) render(item markdownItem) error {
	if g.idx > 0 {
		io.WriteString(g.W, "\n")
	}
	g.idx++

	switch item := item.(type) {
	case *markdownSection:
		return g.renderSection(item)

	case *markdownTitle:
		return g.renderTitle(item)

	case *markdownFile:
		return g.renderFile(item)

	default:
		panic(fmt.Sprintf("unknown markdown item type %T", item))
	}
}

func (g *generator) renderSection(sec *markdownSection) error {
	// TODO: opt-out flag to not render the TOC
	for _, n := range sec.Section.AST {
		// TODO: need to process the links in the TOC as well.
		if err := g.Renderer.Render(g.W, sec.File.Source, n.Node); err != nil {
			return err
		}
		io.WriteString(g.W, "\n")
	}
	return nil
}

func (g *generator) renderTitle(title *markdownTitle) error {
	heading := ast.NewHeading(title.Item.Depth + 1) // offset?
	heading.AppendChild(heading, ast.NewString([]byte(title.Text)))

	if err := g.Renderer.Render(g.W, nil, heading); err != nil {
		return err
	}
	io.WriteString(g.W, "\n")

	return nil
}

func (g *generator) renderFile(file *markdownFile) error {
	for _, h := range file.Headings {
		// TODO: Flag for base level offset
		h.AST.Node.Level += file.Item.Depth
	}

	// TODO: link rewriting should happen between collect and generate.
	for _, l := range file.LocalLinks {
		dst := filepath.Join(file.Dir, l.URL.Path)

		dstf, ok := g.FilesByPath[dst]
		if !ok {
			// TODO: reduce depth of access here -- extract
			// specific information in collector
			g.Log.Printf("%v: link destination not found: %v", file.File.Positioner.Position(l.AST.Pos()), dst)
			continue
		}
		l.URL.Path = ""
		if l.URL.Fragment == "" && dstf.Title != nil {
			// TODO: resolve section ID if fragment is non-empty
			l.AST.Node.Destination = []byte("#" + dstf.Title.ID)
		}
	}
	//
	// TODO: image links relative to output file.
	// TODO: if Title is nil, render

	return g.Renderer.Render(g.W, file.File.Source, file.File.AST.Node)
}
