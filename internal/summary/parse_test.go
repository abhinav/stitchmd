package summary

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"go.abhg.dev/mdreduce/internal/goldast"
	"go.abhg.dev/mdreduce/internal/tree"
)

func TestParseSummary(t *testing.T) {
	t.Parallel()

	item := func(depth int, text, dest string, children ...*tree.Node[*Item]) *tree.Node[*Item] {
		return &tree.Node[*Item]{
			Value: &Item{
				Text:   text,
				Target: dest,
				Depth:  depth,
			},
			List: tree.List[*Item](children),
		}
	}

	section := func(lvl int, title string, items ...*tree.Node[*Item]) *Section {
		return &Section{
			Title: title,
			Items: tree.List[*Item](items),
			Level: lvl,
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
				section(0, "", item(0, "foo", "bar.md")),
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
				section(0, "",
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
				section(0, "",
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
				section(1, "User Guide",
					item(0, "foo", "foo.md"),
					item(0, "bar", "bar.md")),
				section(1, "Appendix",
					item(0, "baz", "baz.md")),
			),
		},
		{
			desc: "items withouth links",
			give: unlines(
				"- foo",
				"- bar",
				"    - [baz](baz.md)",
				"- baz",
			),
			want: toc(
				section(0, "",
					item(0, "foo", ""),
					item(0, "bar", "",
						item(1, "baz", "baz.md")),
					item(0, "baz", "")),
			),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			f, err := goldast.Parse(goldmark.DefaultParser(), "", []byte(tt.give))
			require.NoError(t, err)

			got, err := Parse(f)
			require.NoError(t, err)

			// Zero-out the AST and position to make the test cases
			// easier to write.
			for _, s := range got.Sections {
				if assert.NotNil(t, s.AST) {
					s.AST = nil
				}

				s.Items.Walk(func(i *Item) error {
					i.Pos = 0
					return nil
				})
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSummaryErrors(t *testing.T) {
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
			desc: "styled title",
			give: unlines(
				"- [foo](foo.md)",
				"    - foo *bar* baz",
				"- [baz](baz.md)",
			),
			want: []string{
				"2:7:item has too many children (3)",
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
		{
			desc: "html block",
			give: unlines(
				"- [foo](foo.md)",
				"    - [bar](bar.md)",
				"    - <br/>",
				"- [baz](baz.md)",
			),
			want: []string{
				"3:7:expected text or paragraph, got HTMLBlock",
			},
		},
		{
			desc: "not link or text",
			give: unlines(
				"- [foo](foo.md)",
				"    - [bar](bar.md)",
				"    - `baz`",
				"- [qux](qux.md)",
			),
			want: []string{
				"3:7:expected a link or text, got CodeSpan",
			},
		},
		{
			desc: "non list item",
			give: unlines(
				"# Foo",
				"- [foo](foo.md)",
				"    - [bar](bar.md)",
				"- [baz](baz.md)",
				"",
				"Random paragraph",
			),
			want: []string{
				"6:1:expected a list or heading, got Paragraph",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			f, err := goldast.Parse(goldmark.DefaultParser(), tt.filename, []byte(tt.give))
			require.NoError(t, err)
			defer func() {
				if t.Failed() {
					f.AST.Dump(f.Source, 0)
				}
			}()

			_, err = Parse(f)
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
