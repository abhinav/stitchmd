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

func (g *generator) Generate(sections []*markdownSection) error {
	for _, sec := range sections {
		if err := g.renderSection(sec); err != nil {
			return err
		}
		if err := sec.Items.Walk(g.renderItem); err != nil {
			return err
		}
	}
	return nil
}

func (g *generator) renderItem(item markdownItem) error {
	if g.idx > 0 {
		io.WriteString(g.W, "\n")
	}
	g.idx++

	switch item := item.(type) {
	case *markdownTitle:
		return g.renderTitle(item)

	case *markdownFile:
		return g.renderFile(item)

	default:
		panic(fmt.Sprintf("unknown markdown item type %T", item))
	}
}

func (g *generator) renderSection(sec *markdownSection) error {
	if t := sec.Title; t != nil {
		if err := g.Renderer.Render(g.W, sec.Source, t.AST.Node); err != nil {
			return err
		}
	}

	if err := g.Renderer.Render(g.W, sec.Source, sec.AST.Node); err != nil {
		return err
	}
	io.WriteString(g.W, "\n\n")
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
	return g.Renderer.Render(g.W, file.File.Source, file.File.AST.Node)
}
