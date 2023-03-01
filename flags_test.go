package main

import (
	"bytes"
	"flag"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Verifies that all flags registered against the flag set
// have documentation in the usage text.
func TestUsage_allFlagsDocumented(t *testing.T) {
	t.Parallel()

	_, fset := (&cliParser{
		Stdout: io.Discard,
		Stderr: io.Discard,
	}).newFlagSet()
	fset.VisitAll(func(f *flag.Flag) {
		if !strings.Contains(_usage, "-"+f.Name) {
			t.Errorf("flag -%s is not documented", f.Name)
		}
	})
}

func TestCLIParser_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc    string
		args    []string
		want    params
		wantRes cliParseResult
		wantErr string
	}{
		{
			desc:    "no args",
			wantRes: cliParseError,
			wantErr: "please specify a file name",
		},
		{
			desc:    "help",
			args:    []string{"-h"},
			want:    params{},
			wantRes: cliParseHelp,
		},
		{
			desc:    "version",
			args:    []string{"-version"},
			want:    params{},
			wantRes: cliParseHelp,
		},
		{
			desc: "output",
			args: []string{"-o", "foo", "bar"},
			want: params{Output: "foo", Input: "bar"},
		},
		{
			desc: "stdin",
			args: []string{"-o", "foo", "-"},
			want: params{Output: "foo", Input: ""},
		},
		{
			desc: "output stdout",
			args: []string{"-o", "-", "bar"},
			want: params{Output: "", Input: "bar"},
		},
		{
			desc: "offset",
			args: []string{"-offset", "2", "bar"},
			want: params{Offset: 2, Input: "bar"},
		},
		{
			desc: "offset/negative",
			args: []string{"-offset", "-2", "bar"},
			want: params{Offset: -2, Input: "bar"},
		},
		{
			desc: "no-toc",
			args: []string{"-no-toc", "bar"},
			want: params{NoTOC: true, Input: "bar"},
		},
		{
			desc: "no-toc/explicit true",
			args: []string{"-no-toc=true", "bar"},
			want: params{NoTOC: true, Input: "bar"},
		},
		{
			desc: "no-toc/explicit false",
			args: []string{"-no-toc=false", "bar"},
			want: params{NoTOC: false, Input: "bar"},
		},
		{
			desc: "diff",
			args: []string{"-d", "-o", "foo", "bar"},
			want: params{
				Diff:   true,
				Output: "foo",
				Input:  "bar",
			},
		},
		{
			desc:    "diff/missing o",
			args:    []string{"-d", "bar"},
			wantRes: cliParseError,
			wantErr: "cannot use -d without -o",
		},
		{
			desc:    "too many args",
			args:    []string{"-o", "foo", "bar", "baz"},
			wantErr: "unexpected arguments:",
			wantRes: cliParseError,
		},
		{
			desc:    "unknown flag",
			args:    []string{"-o", "foo", "-x"},
			wantErr: "flag provided but not defined: -x",
			wantRes: cliParseError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			p := &cliParser{
				Stdout: io.Discard,
				Stderr: &stderr,
			}
			got, res := p.Parse(tt.args)
			assert.Equal(t, tt.wantRes, res)
			switch res {
			case cliParseSuccess:
				assert.Equal(t, &tt.want, got)

			case cliParseError:
				assert.Contains(t, stderr.String(), tt.wantErr)
			}
		})
	}
}

func TestFirstLineOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give string
		want string
	}{
		{desc: "empty"},
		{
			desc: "no newline",
			give: "foo",
			want: "foo",
		},
		{
			desc: "single newline",
			give: "foo\n",
			want: "foo\n",
		},
		{
			desc: "multiple newlines",
			give: "foo\nbar\nbaz",
			want: "foo\n",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, firstLineOf(tt.give))
		})
	}
}
