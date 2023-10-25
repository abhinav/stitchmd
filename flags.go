package main

import (
	_ "embed" // for go:embed
	"errors"
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
	Preface string
	Input   string // defaults to stdin
	Output  string // defaults to stdout
	Dir     string
	Offset  int
	NoTOC   bool
	Unsafe  bool

	Diff        bool
	ColorOutput colorOutput
}

// cliParser parses command line arguments.
type cliParser struct {
	Stdout io.Writer // required
	Stderr io.Writer // required

	version bool
	help    bool
}

func (p *cliParser) newFlagSet() (*params, *flag.FlagSet) {
	flag := flag.NewFlagSet("stitchmd", flag.ContinueOnError)
	flag.SetOutput(p.Stderr)
	flag.Usage = func() {
		fmt.Fprint(p.Stderr, _shortHelp)
	}

	var opts params
	flag.StringVar(&opts.Preface, "preface", "", "")
	flag.StringVar(&opts.Output, "o", "", "")
	flag.StringVar(&opts.Dir, "C", "", "")
	flag.IntVar(&opts.Offset, "offset", 0, "")
	flag.BoolVar(&opts.NoTOC, "no-toc", false, "")
	flag.Var(&opts.ColorOutput, "color", "")
	flag.BoolVar(&opts.Diff, "d", false, "")
	flag.BoolVar(&opts.Diff, "diff", false, "")
	flag.BoolVar(&opts.Unsafe, "unsafe", false, "")

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
		fmt.Fprintln(p.Stdout, "Copyright (C) 2023 Abhinav Gupta")
		fmt.Fprintln(p.Stdout, "  <https://github.com/abhinav/stitchmd>")
		fmt.Fprintln(p.Stdout, "stitchmd comes with ABSOLUTELY NO WARRANTY.")
		fmt.Fprintln(p.Stdout, "This is free software, and you are welcome to redistribute it")
		fmt.Fprintln(p.Stdout, "under certain conditions. See source for details.")
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

	// Reject -d if -o is not set.
	if opts.Diff && opts.Output == "" {
		fmt.Fprintln(p.Stderr, "cannot use -d without -o")
		fset.Usage()
		return nil, cliParseError
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

type colorOutput int

const (
	colorOutputAuto colorOutput = iota
	colorOutputAlways
	colorOutputNever
)

var _ flag.Getter = (*colorOutput)(nil)

func (c colorOutput) String() string {
	switch c {
	case colorOutputAuto:
		return "auto"
	case colorOutputAlways:
		return "always"
	case colorOutputNever:
		return "never"
	default:
		return fmt.Sprintf("unknown (%d)", int(c))
	}
}

func (c colorOutput) Get() interface{} {
	return c
}

func (c *colorOutput) Set(s string) error {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "auto":
		*c = colorOutputAuto
	case "always", "true":
		*c = colorOutputAlways
	case "never", "false":
		*c = colorOutputNever
	default:
		return errors.New("must be one of 'always', 'never', 'auto'")
	}
	return nil
}

// Tells "flag" that the flag argument is optional.
// If not provided, Set will be called with "true".
func (c colorOutput) IsBoolFlag() bool {
	return true
}
