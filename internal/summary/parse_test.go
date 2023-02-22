package summary

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.abhg.dev/stitchmd/internal/goldast"
	"go.abhg.dev/stitchmd/internal/tree"
	"gopkg.in/yaml.v3"
)

func TestParseSummary(t *testing.T) {
	t.Parallel()

	linkItem := func(depth int, text, dest string, children ...*tree.Node[Item]) *tree.Node[Item] {
		return &tree.Node[Item]{
			Value: &LinkItem{
				Text:   text,
				Target: dest,
				Depth:  depth,
			},
			List: tree.List[Item](children),
		}
	}

	textItem := func(depth int, text string, children ...*tree.Node[Item]) *tree.Node[Item] {
		return &tree.Node[Item]{
			Value: &TextItem{
				Text:  text,
				Depth: depth,
			},
			List: tree.List[Item](children),
		}
	}

	section := func(lvl int, title string, items ...*tree.Node[Item]) *Section {
		var stitle *SectionTitle
		if len(title) > 0 {
			stitle = &SectionTitle{
				Text:  title,
				Level: lvl,
			}
		}

		return &Section{
			Title: stitle,
			Items: tree.List[Item](items),
		}
	}

	toc := func(secs ...*Section) *TOC {
		return &TOC{Sections: secs}
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
				section(0, "", linkItem(0, "foo", "bar.md")),
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
					linkItem(0, "foo", "foo.md"),
					linkItem(0, "bar", "bar.md"),
					linkItem(0, "baz", "baz.md")),
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
					linkItem(0, "foo", "foo.md",
						linkItem(1, "bar", "bar.md"),
						linkItem(1, "baz", "baz.md")),
					linkItem(0, "qux", "qux.md",
						linkItem(1, "quux", "quux.md"))),
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
					linkItem(0, "foo", "foo.md"),
					linkItem(0, "bar", "bar.md")),
				section(1, "Appendix",
					linkItem(0, "baz", "baz.md")),
			),
		},
		{
			desc: "item groups",
			give: unlines(
				"- foo",
				"    - [bar](bar.md)",
				"    - [baz](baz.md)",
			),
			want: toc(
				section(0, "",
					textItem(0, "foo",
						linkItem(1, "bar", "bar.md"),
						linkItem(1, "baz", "baz.md"))),
			),
		},
		{
			desc: "longer group names",
			give: unlines(
				"- foo bar baz qux quux.",
				"    - [bar](bar.md)",
				"    - [baz](baz.md)",
			),
			want: toc(
				section(0, "",
					textItem(0, "foo bar baz qux quux.",
						linkItem(1, "bar", "bar.md"),
						linkItem(1, "baz", "baz.md"))),
			),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			f, err := goldast.Parse(goldast.DefaultParser(), "", []byte(tt.give))
			require.NoError(t, err)

			got, err := Parse(f)
			require.NoError(t, err)

			// Strip ASTs from the result to make it easier to compare.
			for _, sec := range got.Sections {
				sec.AST = nil
				if sec.Title != nil {
					sec.Title.AST = nil
				}

				sec.Items.Walk(func(n Item) error {
					switch i := n.(type) {
					case *LinkItem:
						i.AST = nil
					case *TextItem:
						i.AST = nil
					default:
						t.Fatalf("unexpected item type %T", i)
					}
					return nil
				})
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSummaryErrors(t *testing.T) {
	t.Parallel()

	testdata, err := os.ReadFile("testdata/parse_errors.yaml")
	require.NoError(t, err)

	var tests []struct {
		Name     string   `yaml:"name"`
		Give     string   `yaml:"give"`
		Filename string   `yaml:"filename"`
		Want     []string `yaml:"want"`
	}
	require.NoError(t, yaml.Unmarshal(testdata, &tests))

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			f, err := goldast.Parse(goldast.DefaultParser(), tt.Filename, []byte(tt.Give))
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

			require.Len(t, gotErrors, len(tt.Want), "unexpected number of errors:\n%+v", err)
			for i, wantErr := range tt.Want {
				assert.EqualError(t, gotErrors[i], wantErr)
			}
		})
	}
}

func unlines(lines ...string) string {
	return strings.Join(append(lines, ""), "\n")
}
