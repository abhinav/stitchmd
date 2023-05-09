// stitchmd reads a Markdown file defining a table of contents
// with links to other Markdown files,
// and reduces it all to a single Markdown file.
//
// See README for more details.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	mdfmt "github.com/Kunde21/markdownfmt/v3/markdown"
	"github.com/mattn/go-colorable"
	isatty "github.com/mattn/go-isatty"
	"github.com/pkg/diff"
	"github.com/pkg/diff/write"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/stitchmd/internal/goldast"
	"go.abhg.dev/stitchmd/internal/rawhtml"
	"go.abhg.dev/stitchmd/internal/stitch"
)

var _version = "dev"

func main() {
	cmd := mainCmd{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Getwd:  os.Getwd,
		Getenv: os.Getenv,
	}
	os.Exit(cmd.Run(os.Args[1:]))
}

type mainCmd struct {
	Stdin  io.Reader // required (os.Stdin)
	Stdout io.Writer // required (os.Stdout)
	Stderr io.Writer // required (os.Stderr)

	Getwd  func() (string, error) // required (os.Getwd)
	Getenv func(string) string    // required (os.Getenv)
}

func (cmd *mainCmd) Run(args []string) (exitCode int) {
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
		fmt.Fprintln(cmd.Stderr, "stitchmd:", err)
		return 1
	}

	return 0
}

func (cmd *mainCmd) shouldColor(opts *params) bool {
	switch opts.ColorOutput {
	case colorOutputAuto:
		return cmd.Getenv("NO_COLOR") == "" &&
			cmd.Getenv("TERM") != "dumb" &&
			supportsColor(cmd.Stdout)
	case colorOutputAlways:
		return true
	default:
		return false
	}
}

func (cmd *mainCmd) run(opts *params) error {
	shouldColor := cmd.shouldColor(opts)
	if shouldColor {
		cmd.Stdout = makeColorable(cmd.Stdout)
	}

	log := log.New(cmd.Stderr, "", 0)

	input := cmd.Stdin
	filename := "<stdin>"
	if len(opts.Input) > 0 {
		filename = opts.Input
		f, err := os.Open(opts.Input)
		if err != nil {
			return err
		}
		defer f.Close()
		input = f
	}

	var preface []byte
	if len(opts.Preface) > 0 {
		var err error
		preface, err = os.ReadFile(opts.Preface)
		if err != nil {
			return fmt.Errorf("-preface: %w", err)
		}

		// Ensure trailing newline.
		if len(preface) > 0 && preface[len(preface)-1] != '\n' {
			preface = append(preface, '\n')
		}
	}

	cwd, err := cmd.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}
	// Input and output directories are determined in the following order:
	//
	//  - -C flag takes precedence over everything
	//  - If a file path is specified for input/output, use that directory
	//  - Use current directory otherwise
	determineDir := func(fpath string) string {
		if opts.Dir != "" {
			return opts.Dir
		}
		if fpath != "" {
			return filepath.Dir(fpath)
		}
		return cwd
	}

	inputDir := determineDir(opts.Input)
	outputDir := determineDir(opts.Output)

	output := cmd.Stdout
	if len(opts.Output) > 0 {
		if opts.Diff {
			dw, err := newDiffWriter(opts.Output, shouldColor)
			if err != nil {
				return fmt.Errorf("-diff: %w", err)
			}
			defer dw.Diff(cmd.Stdout)
			output = dw
		} else {
			outDir := filepath.Dir(opts.Output)
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return fmt.Errorf("create output directory: %w", err)
			}

			f, err := os.Create(opts.Output)
			if err != nil {
				return fmt.Errorf("create output: %w", err)
			}
			defer f.Close()
			output = f
		}
	}

	// Relative path from the output directory back to the input directory.
	// This is used to generate relative links to images and other files
	// that aren't part of the collection.
	var inputRel string
	{
		outAbs, err := filepath.Abs(outputDir)
		if err != nil {
			return err
		}
		inAbs, err := filepath.Abs(inputDir)
		if err != nil {
			return err
		}

		inputRel, err = filepath.Rel(outAbs, inAbs)
		if err != nil {
			return err
		}
	}

	src, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("input: %w", err)
	}

	mdParser := goldast.DefaultParser()
	mdParser.AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&rawhtml.Transformer{}, 100),
		),
	)

	f := goldast.Parse(mdParser, filename, src)
	summary, err := stitch.ParseSummary(f)
	if err != nil {
		log.Println(err)
		return errors.New("error parsing summary")
	}

	coll, err := (&collector{
		FS:     os.DirFS(inputDir),
		Parser: mdParser,
	}).Collect(f.Info, summary)
	if err != nil {
		log.Println(err)
		return errors.New("error reading markdown")
	}

	(&transformer{
		Log:          log,
		Offset:       opts.Offset,
		InputRelPath: filepath.ToSlash(inputRel),
	}).Transform(coll)

	render := mdfmt.NewRenderer()
	render.AddMarkdownOptions(
		mdfmt.WithSoftWraps(),
	)

	g := &generator{
		Preface:  preface,
		W:        output,
		Renderer: render,
		Log:      log,
		NoTOC:    opts.NoTOC,
	}
	return g.Generate(f.Source, coll)
}

// diffWriter is an io.Writer that buffers the input
// and compares it against a reference.
// If the input doesn't match the reference,
// a diff is printed to stdout when the writer closes.
type diffWriter struct {
	fname string
	old   []byte
	new   bytes.Buffer
	color bool
}

func newDiffWriter(fname string, color bool) (*diffWriter, error) {
	old, err := os.ReadFile(fname)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		old = nil
	}

	return &diffWriter{
		fname: fname,
		old:   old,
		color: color,
	}, nil
}

func (dw *diffWriter) Write(p []byte) (int, error) {
	return dw.new.Write(p)
}

func (dw *diffWriter) Diff(w io.Writer) error {
	if bytes.Equal(dw.old, dw.new.Bytes()) {
		return nil
	}

	var opts []write.Option
	if dw.color {
		opts = append(opts, write.TerminalColor())
	}

	return diff.Text(
		filepath.Join("a", dw.fname),
		filepath.Join("b", dw.fname),
		dw.old,
		dw.new.Bytes(),
		w,
		opts...,
	)
}

func supportsColor(w io.Writer) bool {
	// TODO: Use Is*Writer variants once this lands:
	// https://github.com/mattn/go-isatty/pull/81
	if f, ok := w.(interface{ Fd() uintptr }); ok {
		return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
	return false
}

func makeColorable(w io.Writer) io.Writer {
	if f, ok := w.(*os.File); ok {
		// TODO: Drop upcast once this lands:
		// https://github.com/mattn/go-colorable/pull/66
		return colorable.NewColorable(f)
	}
	return w
}
