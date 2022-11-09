package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"go.abhg.dev/mdreduce/internal/goldast"
)

func TestParseTOC(t *testing.T) {
	t.Parallel()

	item := func(depth int, title, file string, children ...*Item) *Item {
		return &Item{
			Title: title,
			File:  file,
			Depth: depth,
			Items: children,
		}
	}

	section := func(title string, items ...*Item) *Section {
		return &Section{
			Title: title,
			Items: items,
		}
	}

	toc := func(secs ...*Section) *TOC {
		return &TOC{
			Sections: secs,
		}
	}

	tests := []struct {
		desc string
		give string
		want *TOC
	}{
		{
			desc: "one",
			give: "- [foo](bar.md)",
			want: toc(
				section("", item(0, "foo", "bar.md")),
			),
		},
		{
			desc: "siblings",
			give: unlines(
				"- [foo](foo.md)",
				"- [bar](bar.md)",
				"- [baz](baz.md)",
			),
			want: toc(
				section("",
					item(0, "foo", "foo.md"),
					item(0, "bar", "bar.md"),
					item(0, "baz", "baz.md")),
			),
		},
		{
			desc: "children",
			give: unlines(
				"- [foo](foo.md)",
				"    - [bar](bar.md)",
				"    - [baz](baz.md)",
				"- [qux](qux.md)",
				"    - [quux](quux.md)",
			),
			want: toc(
				section("",
					item(0, "foo", "foo.md",
						item(1, "bar", "bar.md"),
						item(1, "baz", "baz.md")),
					item(0, "qux", "qux.md",
						item(1, "quux", "quux.md"))),
			),
		},
		{
			desc: "section headings",
			give: unlines(
				"# User Guide",
				"- [foo](foo.md)",
				"- [bar](bar.md)",
				"",
				"# Appendix",
				"- [baz](baz.md)",
			),
			want: toc(
				section("User Guide",
					item(0, "foo", "foo.md"),
					item(0, "bar", "bar.md")),
				section("Appendix",
					item(0, "baz", "baz.md")),
			),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			src := []byte(tt.give)
			doc, err := goldast.Parse(goldmark.DefaultParser(), src)
			require.NoError(t, err)

			got, err := parseTOC("", src, doc)
			require.NoError(t, err)

			if assert.Equal(t, src, got.Source) {
				got.Source = nil
			}
			if assert.NotNil(t, got.Positioner) {
				got.Positioner = nil
			}

			for _, s := range got.Sections {
				if assert.NotNil(t, s.AST) {
					s.AST = nil
				}

				s.visitAllItems(func(i *Item) {
					i.Pos = 0
				})
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseTOCErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc     string
		give     string
		filename string
		want     []string
	}{
		{
			desc:     "no sections",
			filename: "stdin",
			want: []string{
				"stdin:1:1:no sections found",
			},
		},
		{
			desc: "no list or heading",
			give: unlines("foo"),
			want: []string{
				"1:1:expected a list or heading, got Paragraph",
			},
		},
		{
			desc: "no list after heading",
			give: unlines(
				"# Foo",
				"",
				"bar",
			),
			want: []string{
				"3:1:expected a list, got Paragraph",
			},
		},
		{
			desc: "no link",
			give: unlines(
				"- [foo](foo.md)",
				"    - bar",
				"- [baz](baz.md)",
			),
			want: []string{
				"2:7:expected a link, got Text",
			},
		},
		{
			desc: "too many children",
			give: unlines(
				"- [foo](foo.md)",
				"    - [bar](bar.md)",
				"    - [baz](baz.md)",
				"    - qux",
				"",
				"      bar",
				"",
				"      baz",
			),
			want: []string{
				"4:7:item has too many children (3)",
			},
		},
		{
			desc: "not a sublist",
			give: unlines(
				"- [foo](foo.md)",
				"",
				"    not a list item",
				"- [bar](bar.md)",
			),
			want: []string{
				"3:5:expected a list, got Paragraph",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			src := []byte(tt.give)
			doc, err := goldast.Parse(goldmark.DefaultParser(), src)
			require.NoError(t, err)
			defer func() {
				if t.Failed() {
					doc.Dump(src, 0)
				}
			}()

			_, err = parseTOC(tt.filename, src, doc)
			require.Error(t, err)

			var gotErrors []error
			if uerr, ok := err.(interface{ Unwrap() []error }); ok {
				gotErrors = uerr.Unwrap()
			} else {
				gotErrors = []error{err}
			}

			require.Len(t, gotErrors, len(tt.want), "unexpected number of errors:\n%+v", err)
			for i, wantErr := range tt.want {
				assert.EqualError(t, gotErrors[i], wantErr)
			}
		})
	}
}

func unlines(lines ...string) string {
	return strings.Join(append(lines, ""), "\n")
}
