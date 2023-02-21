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
	"log"
	"os"
	"path/filepath"

	mdfmt "github.com/Kunde21/markdownfmt/v3/markdown"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"go.abhg.dev/mdreduce/internal/goldast"
	"go.abhg.dev/mdreduce/internal/header"
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

	filesByPath := make(map[string]*markdownFile)
	sections, err := (&collector{
		FS:     os.DirFS(opts.Dir),
		Parser: mdParser,
		IDGen:  header.NewIDGen(),
		files:  filesByPath,
	}).Collect(f)
	if err != nil {
		cmd.log.Println(err)
		return errors.New("error reading markdown")
	}

	(&transformer{
		Files: filesByPath,
		Log:   cmd.log,
	}).transformList(sections)

	render := mdfmt.NewRenderer()
	render.AddMarkdownOptions(
		mdfmt.WithSoftWraps(),
	)

	g := &generator{
		W:        output,
		Renderer: render,
		Log:      cmd.log,
	}
	return g.Generate(sections)
}
