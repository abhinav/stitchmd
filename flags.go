package main

import (
	_ "embed" // for go:embed
	"flag"
	"fmt"
	"io"
	"strings"
)

var (
	//go:embed usage.txt
	_usage string

	_shortHelp = firstLineOf(_usage)
)

// params defines the parameters for the command line program.
type params struct {
	Input  string // defaults to stdin
	Output string // defaults to stdout
	Dir    string
}

// cliParser parses command line arguments.
type cliParser struct {
	Stdout io.Writer
	Stderr io.Writer

	version bool
	help    bool
}

func (p *cliParser) newFlagSet() (*params, *flag.FlagSet) {
	flag := flag.NewFlagSet("mdreduce", flag.ContinueOnError)
	flag.SetOutput(p.Stderr)
	flag.Usage = func() {
		fmt.Fprint(p.Stderr, _shortHelp)
	}

	var opts params
	flag.StringVar(&opts.Output, "o", "", "")
	flag.StringVar(&opts.Dir, "C", "", "")

	flag.BoolVar(&p.version, "version", false, "")
	flag.BoolVar(&p.help, "help", false, "")
	flag.BoolVar(&p.help, "h", false, "")

	return &opts, flag
}

type cliParseResult int

const (
	cliParseSuccess cliParseResult = iota
	cliParseHelp
	cliParseError
)

// Parses and returns command line parameters.
// This function does not return an error to ensure
// that error messages are not double-printed.
func (p *cliParser) Parse(args []string) (*params, cliParseResult) {
	opts, fset := p.newFlagSet()
	if err := fset.Parse(args); err != nil {
		return nil, cliParseError
	}
	args = fset.Args()

	if p.version {
		fmt.Fprintln(p.Stdout, fset.Name(), strings.TrimSpace(_version))
		return nil, cliParseHelp
	}
	if p.help {
		fmt.Fprint(p.Stdout, _usage)
		return nil, cliParseHelp
	}

	switch len(args) {
	case 0:
		fmt.Fprintln(p.Stderr, "please specify a file name")
		fset.Usage()
		return nil, cliParseError
	case 1:
		opts.Input = args[0]
	default:
		fmt.Fprintf(p.Stderr, "unexpected arguments: %q\n", args[1:])
		fset.Usage()
		return nil, cliParseError
	}

	// "-" means stdin/stdout
	if opts.Input == "-" {
		opts.Input = ""
	}
	if opts.Output == "-" {
		opts.Output = ""
	}

	return opts, cliParseSuccess
}

// Returns the first line of the given string.
func firstLineOf(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i+1]
	}
	return s
}
