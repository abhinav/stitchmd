// mdreduce reads a Markdown file defining a table of contents
// with links to other Markdown files,
// and reduces it all to a single Markdown file.
//
// See README for more details.
package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path/filepath"

	mdfmt "github.com/Kunde21/markdownfmt/v3/markdown"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/mdreduce/internal/goldast"
	"go.abhg.dev/mdreduce/internal/pos"
	"go.abhg.dev/mdreduce/internal/summary"
)

var _version string = "dev"

func main() {
	cmd := mainCmd{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Getwd:  os.Getwd,
	}
	os.Exit(cmd.Run(os.Args[1:]))
}

type mainCmd struct {
	Stdin  io.Reader // == os.Stdin
	Stdout io.Writer // == os.Stdout
	Stderr io.Writer // == os.Stderr

	Getwd func() (string, error) // == os.Getwd

	log *log.Logger
}

func (cmd *mainCmd) Run(args []string) (exitCode int) {
	cmd.log = log.New(cmd.Stderr, "", 0)

	opts, res := (&cliParser{
		Stdout: cmd.Stdout,
		Stderr: cmd.Stderr,
	}).Parse(args)
	switch res {
	case cliParseSuccess:
		// continue
	case cliParseHelp:
		return 0
	case cliParseError:
		return 1
	}

	if err := cmd.run(opts); err != nil {
		fmt.Fprintln(cmd.Stderr, "mdreduce:", err)
		return 1
	}

	return 0
}

func (cmd *mainCmd) run(opts *params) error {
	input := cmd.Stdin
	filename := "<stdin>"
	if len(opts.Input) > 0 {
		filename = opts.Input
		f, err := os.Open(opts.Input)
		if err != nil {
			return fmt.Errorf("open input: %w", err)
		}
		defer f.Close()
		input = f
	}

	// Default to file's directory if -C is not specified,
	// and current directory if reading from stdin.
	if opts.Dir == "" {
		if len(opts.Input) > 0 {
			opts.Dir = filepath.Dir(opts.Input)
		} else {
			var err error
			opts.Dir, err = cmd.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
		}
	}

	output := cmd.Stdout
	if len(opts.Output) > 0 {
		f, err := os.Create(opts.Output)
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		defer f.Close()
		output = f
	}

	src, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	mdParser := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	).Parser()

	f, err := goldast.Parse(mdParser, filename, src)
	if err != nil {
		return err
	}

	toc, err := summary.Parse(f)
	if err != nil {
		cmd.log.Println(err)
		return errors.New("error extracting TOC")
	}

	rdr := Reader{
		FS:     os.DirFS(opts.Dir),
		Parser: mdParser,
		IDGen:  newIDGenerator(),
	}
	var files []*markdownFile
	errs := pos.NewErrorList(f.Positioner)
	for _, sec := range toc.Sections {
		sec.Items.Walk(func(item *summary.Item) error {
			f, err := rdr.ReadFile(item.File)
			if err != nil {
				errs.Pushf(item.Pos, "%v", err)
				return nil
			}
			files = append(files, f)
			return nil
		})
	}
	if err := errs.Err(); err != nil {
		cmd.log.Println(err)
		return errors.New("error reading files")
	}

	render := mdfmt.NewRenderer()
	render.AddMarkdownOptions(
		mdfmt.WithSoftWraps(),
	)

	filesByPath := make(map[string]*markdownFile)
	for _, f := range files {
		filesByPath[f.Path] = f
	}

	g := &generator{
		W:           output,
		FilesByPath: filesByPath,
		Renderer:    render,
		Log:         cmd.log,
	}

	if err := g.Render(f, toc); err != nil {
		cmd.log.Println(err)
		return errors.New("error rendering TOC")
	}

	return nil
}

type generator struct {
	W           io.Writer
	Renderer    *mdfmt.Renderer
	Log         *log.Logger
	FilesByPath map[string]*markdownFile
}

func (g *generator) Render(f *goldast.File, toc *summary.TOC) error {
	errs := pos.NewErrorList(f.Positioner)
	for _, sec := range toc.Sections {
		// TODO: opt-out flag to not render the TOC
		for _, n := range sec.AST {
			// TODO: need to process the links in the TOC as well.
			if err := g.Renderer.Render(g.W, f.Source, n.Node); err != nil {
				return err
			}
			io.WriteString(g.W, "\n")
		}

		sec.Items.Walk(func(i *summary.Item) error {
			io.WriteString(g.W, "\n")
			if err := g.RenderItem(i); err != nil {
				errs.Pushf(i.Pos, "render: %w", err)
			}
			return nil
			// TODO: Prevent auto-heading ID from being rendered
		})
	}

	return errs.Err()
}

func (g *generator) RenderItem(item *summary.Item) error {
	f := g.FilesByPath[item.File]
	if f == nil {
		// TODO: this shouldn't need a map lookup.
		// We should track the files and their parsed contents
		// together.
		return fmt.Errorf("file not found: %q", item.File)
	}

	// if f.Title == nil {
	// TODO: render a <a id="..."></a> tag for the file name.
	// }

	for _, h := range f.Headings {
		h.AST.Node.Level += item.Depth
	}

	for _, l := range f.LocalLinks {
		dst := filepath.Join(f.Dir, l.URL.Path)
		dstf, ok := g.FilesByPath[dst]
		if !ok {
			g.Log.Printf("%v: link destination not found: %v", f.Positioner.Position(l.AST.Pos()), dst)
			continue
		}
		l.URL.Path = ""
		if l.URL.Fragment == "" && dstf.Title != nil {
			// TODO: resolve section ID if fragment is non-empty
			l.AST.Node.Destination = []byte("#" + dstf.Title.ID)
		}
	}

	// TODO: image links relative to output file.
	// TODO: if Title is nil, render

	return g.Renderer.Render(g.W, f.Source, f.AST.Node)
}

type markdownFile struct {
	*goldast.File

	// Path to the file relative to the FS root.
	Path string

	// Directory containing the file.
	// This is the same as filepath.Dir(Path).
	Dir string

	// Level 1 heading acting as the title for the document.
	// This is non-nil only if the document has exactly one such heading.
	Title *markdownHeading

	// Local links and images in the file.
	LocalLinks  []*localReference[*ast.Link]
	LocalImages []*localReference[*ast.Image]
	Headings    []*markdownHeading
}

type markdownHeading struct {
	ID    string
	AST   *goldast.Node[*ast.Heading]
	Level int
}

type localReference[N ast.Node] struct {
	AST *goldast.Node[N]
	URL *url.URL
}

type Reader struct {
	FS     fs.FS
	Parser parser.Parser

	IDGen *idGenerator
}

// ReadFile reads a Markdown file from the given path.
// The path must be relative to the root of the Reader's FS.
func (r *Reader) ReadFile(path string) (*markdownFile, error) {
	bs, err := fs.ReadFile(r.FS, path)
	if err != nil {
		return nil, err
	}

	f, err := goldast.Parse(r.Parser, path, bs)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	var (
		links    []*localReference[*ast.Link]
		images   []*localReference[*ast.Image]
		headings []*markdownHeading

		// Level 1 headings in the file.
		h1s []*markdownHeading
	)

	err = goldast.Walk(f.AST, func(n *goldast.Node[ast.Node], enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}

		if l, ok := goldast.Cast[*ast.Link](n); ok {
			u, err := url.Parse(string(l.Node.Destination))
			if err != nil || u.Scheme != "" || u.Host != "" {
				return ast.WalkContinue, nil // skip external and invalid links
			}
			links = append(links, &localReference[*ast.Link]{
				AST: l,
				URL: u,
			})
		} else if i, ok := goldast.Cast[*ast.Image](n); ok {
			u, err := url.Parse(string(i.Node.Destination))
			if err != nil || u.Scheme != "" || u.Host != "" {
				return ast.WalkContinue, nil // skip external and invalid links
			}
			images = append(images, &localReference[*ast.Image]{
				AST: i,
				URL: u,
			})
		} else if h, ok := goldast.Cast[*ast.Heading](n); ok {
			title := n.Node.Text(bs)
			slug, _ := r.IDGen.GenerateID(string(title))
			// if !ok {
			// 	// TODO: do we need to handle this?
			// }
			heading := &markdownHeading{
				AST:   h,
				ID:    slug,
				Level: h.Node.Level,
			}
			headings = append(headings, heading)
			if heading.Level == 1 {
				h1s = append(h1s, heading)
			}
		} else {
			return ast.WalkContinue, nil
		}
		return ast.WalkSkipChildren, nil
	})

	var title *markdownHeading
	if len(h1s) == 1 {
		title = h1s[0]
	}

	return &markdownFile{
		Path:        path,
		Dir:         filepath.Dir(path),
		File:        f,
		Title:       title,
		LocalLinks:  links,
		LocalImages: images,
		Headings:    headings,
	}, err
}
