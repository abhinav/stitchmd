// Package stitch defines the core data types for a stitch document.
//
// [Summary] is the full summary file, comprised of one or more [Section]s,
// each of which in turn has one or more [Item]s.
package stitch

import (
	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/stitchmd/internal/goldast"
	"go.abhg.dev/stitchmd/internal/pos"
	"go.abhg.dev/stitchmd/internal/tree"
)

// Summary is the complete summary document.
// It's comprised of one or more sections.
type Summary struct {
	// Sections is a list of sections in the summary.
	Sections []*Section
}

// ParseSummary parses a summary from a Markdown document.
// The summary is expected in a very specific format:
//
//   - it's comprised of one or more sections
//   - each section has an optional title header and a list of items
//   - each item is either a link to a Markdown document or a plain text title
//   - items may be nested to indicate a hierarchy
//
// For example:
//
//	# User Guide
//
//	- [Getting Started](getting-started.md)
//	    - [Installation](installation.md)
//	- Options
//	    - [foo](foo.md)
//	    - [bar](bar.md)
//	- [Reference](reference.md)
//
//	# Appendix
//
//	- [FAQ](faq.md)
//
// Anything else will result in an error.
func ParseSummary(f *goldast.File) (*Summary, error) {
	errs := pos.NewErrorList(f.Info)
	summary := (&summaryParser{
		src:  f.Source,
		errs: errs,
	}).Parse(f.AST)
	return summary, errs.Err()
}

type summaryParser struct {
	src      []byte
	errs     *pos.ErrorList
	sections []*Section
}

func (p *summaryParser) Parse(n *goldast.Any) *Summary {
	for n := n.FirstChild(); n != nil; {
		sec, next := p.parseSection(n)
		if sec != nil {
			p.sections = append(p.sections, sec)
		}
		n = next
	}

	if len(p.sections) == 0 && p.errs.Len() == 0 {
		p.errs.Pushf(n.Pos(), "no sections found")
		return nil
	}

	return &Summary{Sections: p.sections}
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

// parseSection parses a Section from the given node.
func (p *summaryParser) parseSection(n *goldast.Any) (*Section, *goldast.Any) {
	title, n := p.parseSectionTitle(n)

	ls, ok := goldast.Cast[*ast.List](n)
	if !ok {
		if title != nil {
			p.errs.Pushf(n.Pos(), "expected a list, got %v", n.Kind())
		} else {
			p.errs.Pushf(n.Pos(), "expected a list or heading, got %v", n.Kind())
		}
		return nil, n.NextSibling()
	}

	items := (&itemTreeParser{
		src:  p.src,
		errs: p.errs,
	}).Parse(ls)

	return &Section{
		Title: title,
		Items: items,
		AST:   ls,
	}, n.NextSibling()
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

func (p *summaryParser) parseSectionTitle(n *goldast.Any) (*SectionTitle, *goldast.Any) {
	h, ok := goldast.Cast[*ast.Heading](n)
	if !ok {
		return nil, n
	}

	return &SectionTitle{
		Text:  string(h.Node.Text(p.src)),
		Level: h.Node.Level,
		AST:   h,
	}, n.NextSibling()
}
