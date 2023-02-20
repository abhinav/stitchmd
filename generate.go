package main

import (
	"fmt"
	"io"
	"log"

	mdfmt "github.com/Kunde21/markdownfmt/v3/markdown"
	"github.com/yuin/goldmark/ast"
)

type generator struct {
	idx int

	W        io.Writer
	Renderer *mdfmt.Renderer
	Log      *log.Logger
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
	for _, n := range sec.AST {
		if err := g.Renderer.Render(g.W, sec.Source, n.Node); err != nil {
			return err
		}
		io.WriteString(g.W, "\n")
	}
	return nil
}

func (g *generator) renderTitle(title *markdownTitle) error {
	heading := ast.NewHeading(title.Depth + 1) // depth => level
	heading.AppendChild(heading, ast.NewString([]byte(title.Text)))

	if err := g.Renderer.Render(g.W, nil, heading); err != nil {
		return err
	}
	io.WriteString(g.W, "\n")

	return nil
}

func (g *generator) renderFile(file *markdownFile) error {
	return g.Renderer.Render(g.W, file.Source, file.AST.Node)
}
