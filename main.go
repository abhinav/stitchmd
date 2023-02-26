// stitchmd reads a Markdown file defining a table of contents
// with links to other Markdown files,
// and reduces it all to a single Markdown file.
//
// See README for more details.
package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	mdfmt "github.com/Kunde21/markdownfmt/v3/markdown"
	"go.abhg.dev/stitchmd/internal/goldast"
	"go.abhg.dev/stitchmd/internal/stitch"
)

var _version = "dev"

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

func (cmd *mainCmd) run(opts *params) error {
	log := log.New(cmd.Stderr, "", 0)

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
		return fmt.Errorf("read input: %w", err)
	}

	mdParser := goldast.DefaultParser()

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
		W:        output,
		Renderer: render,
		Log:      log,
	}
	return g.Generate(f.Source, coll)
}
