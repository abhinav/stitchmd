// Package summary defines the representation of the summary document.
//
// The summary document defines the table of contents
// for the combined Markdown document.
package summary

import (
	"go.abhg.dev/mdreduce/internal/goldast"
	"go.abhg.dev/mdreduce/internal/tree"
)

// TOC is the complete summary document.
// It's comprised of one or more sections.
type TOC struct {
	// Sections is a list of sections in the summary.
	Sections []*Section
}

// Section is a single section of a summary document.
// It's comprised of an optional title and a tree of items.
type Section struct {
	Title *SectionTitle

	// Items lists the items in the section
	// and their nested items.
	Items tree.List[Item]

	// AST holds the original list
	// from which this Section was built.
	AST *goldast.List
}

func (s *Section) TitleLevel() int {
	if s.Title == nil {
		return 0
	}
	return s.Title.Level
}

type SectionTitle struct {
	Text  string
	Level int
	AST   *goldast.Heading
}

// Item is a single item in a section.
// It can be a [LinkItem] or a [TextItem].
type Item interface {
	item() // seals the interface

	// Reports the depth of the item in the tree,
	// with zero being the top-level items.
	ItemDepth() int

	// ASTNode returns the AST node that this item was built from.
	ASTNode() *goldast.Any
}

// LinkItem is a single link item in a table of contents.
//
//	[Foo](foo.md)
type LinkItem struct {
	// Text of the item.
	// This is the text inside the "[..]" section of the link.
	Text string

	// Target is the destination of this item.
	// This is the text inside the "(..)" section of the link.
	Target string

	// Depth is the depth of the item in the table of contents.
	// Depth starts at zero for top-level items.
	Depth int

	// AST holds the original link node.
	AST *goldast.Link
}

func (*LinkItem) item() {}

// ItemDepth reports the depth of the LinkItem in the tree.
func (i *LinkItem) ItemDepth() int {
	return i.Depth
}

// ASTNode returns the Link node that this item was built from.
func (i *LinkItem) ASTNode() *goldast.Any {
	return i.AST.AsAny()
}

// TextItem is a single text entry in the table of contents.
//
//	Foo
type TextItem struct {
	// Text of the item.
	Text string

	// Depth is the depth of the item in the table of contents.
	// Depth starts at zero for top-level items.
	Depth int

	// AST holds the original text node.
	AST *goldast.Text
}

func (*TextItem) item() {}

// ItemDepth reports the depth of the TextItem in the tree.
func (i *TextItem) ItemDepth() int {
	return i.Depth
}

// ASTNode returns the Text node that this item was built from.
func (i *TextItem) ASTNode() *goldast.Any {
	return i.AST.AsAny()
}
