// Package summary defines the representation of the summary document.
//
// The summary document defines the table of contents
// for the combined Markdown document.
package summary

import (
	"go.abhg.dev/stitchmd/internal/goldast"
	"go.abhg.dev/stitchmd/internal/pos"
	"go.abhg.dev/stitchmd/internal/tree"
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

// SectionTitle holds information about a section title.
type SectionTitle struct {
	// Text of the title.
	Text string

	// Level of the title.
	Level int

	// AST node that this title was built from.
	AST *goldast.Heading
}

// Item is a single item in a section.
// It can be a [LinkItem] or a [TextItem].
type Item interface {
	item() // seals the interface

	Pos() pos.Pos
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

// Pos reports the position in the original TOC
// where this item was found.
func (i *LinkItem) Pos() pos.Pos {
	return i.AST.Pos()
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

// Pos reports the position in the original TOC
// where this item was found.
func (i *TextItem) Pos() pos.Pos {
	return i.AST.Pos()
}
