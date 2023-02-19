// Package summary defines the representation of the summary document.
//
// The summary document defines the table of contents
// for the combined Markdown document.
package summary

import (
	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/mdreduce/internal/goldast"
	"go.abhg.dev/mdreduce/internal/pos"
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
	// Title of the section, if any.
	Title string

	// Items lists the items in the section
	// and their nested items.
	Items tree.List[*Item]

	// AST nodes that make up this section.
	AST []*goldast.Node[ast.Node]
}

// Item is a single item in a table of contents.
// It's built from links or single line titles in the list.
type Item struct {
	// Text of the item.
	//
	// If the item was built from a link,
	// this is the text inside the "[..]" section of the link.
	Text string

	// Target is the destination of this item, if any.
	// This is blank if the item was built from a single line title.
	Target string

	// Depth is the depth of the item in the table of contents.
	// Depth starts at zero for top-level items.
	Depth int

	// Position at which this item was found.
	Pos pos.Pos
}
