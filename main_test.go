package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain_badFlag(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	exitCode := (&mainCmd{
		Stdin:  bytes.NewReader(nil),
		Stdout: io.Discard,
		Stderr: &stderr,
		Getwd:  os.Getwd,
		Getenv: nopGetenv,
	}).Run([]string{"--flag-does-not-exist"})

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr.String(),
		"flag provided but not defined: -flag-does-not-exist")
}

func TestMain_helpFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	exitCode := (&mainCmd{
		Stdin:  bytes.NewReader(nil),
		Stdout: &stdout,
		Stderr: io.Discard,
		Getwd:  os.Getwd,
		Getenv: nopGetenv,
	}).Run([]string{"--help"})

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout.String(), _usage)
}

func TestMain_versionFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	exitCode := (&mainCmd{
		Stdin:  bytes.NewReader(nil),
		Stdout: &stdout,
		Stderr: io.Discard,
		Getwd:  os.Getwd,
		Getenv: nopGetenv,
	}).Run([]string{"--version"})

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout.String(), _version)
}

func TestMain_summaryDoesNotExist(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	exitCode := (&mainCmd{
		Stdin:  bytes.NewReader(nil),
		Stdout: io.Discard,
		Stderr: &stderr,
		Getwd:  os.Getwd,
		Getenv: nopGetenv,
	}).Run([]string{"does-not-exist.md"})

	assert.Equal(t, 1, exitCode)
	assertNoSuchFileError(t, stderr.String())
}

func TestMain_cwdError(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	exitCode := (&mainCmd{
		Stdin:  bytes.NewReader(nil),
		Stdout: io.Discard,
		Stderr: &stderr,
		Getwd: func() (string, error) {
			return "", os.ErrPermission
		},
		Getenv: nopGetenv,
	}).Run([]string{"-"})

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr.String(), "permission denied")
}

func TestMain_badSummary(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	exitCode := (&mainCmd{
		Stdin:  bytes.NewReader([]byte("great sadness")),
		Stdout: io.Discard,
		Stderr: &stderr,
		Getwd:  os.Getwd,
		Getenv: nopGetenv,
	}).Run([]string{"-"})

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr.String(), "1:1:expected a list or heading")
	assert.Contains(t, stderr.String(), "error parsing summary")
}

func TestMain_summaryItemDoesNotExist(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	exitCode := (&mainCmd{
		Stdin:  bytes.NewReader([]byte("- [does-not-exist](does-not-exist.md)")),
		Stdout: io.Discard,
		Stderr: &stderr,
		Getwd:  os.Getwd,
		Getenv: nopGetenv,
	}).Run([]string{"-"})
	assert.Equal(t, 1, exitCode)
	assertNoSuchFileError(t, stderr.String())
	assert.Contains(t, stderr.String(), "error reading markdown")
}

func TestMain_prefaceDoesNotExist(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	exitCode := (&mainCmd{
		Stdin:  bytes.NewReader([]byte("- [foo](foo.md)")),
		Stdout: io.Discard,
		Stderr: &stderr,
		Getwd:  os.Getwd,
		Getenv: nopGetenv,
	}).Run([]string{"-preface", "does-not-exist.md", "-"})
	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr.String(), "-preface: open does-not-exist.md:")
	assertNoSuchFileError(t, stderr.String())
}

func TestMain_shouldColor(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("pty is unsupported on Windows")
	}

	pty, tty, err := pty.Open()
	require.NoError(t, err)
	defer pty.Close()
	defer tty.Close()

	tests := []struct {
		desc   string
		color  colorOutput
		env    map[string]string
		stdout io.Writer

		want bool
	}{
		{
			desc: "auto/dumb terminal",
			env:  map[string]string{"TERM": "dumb"},
			want: false,
		},
		{
			desc: "auto/no color",
			env:  map[string]string{"NO_COLOR": "1"},
			want: false,
		},
		{
			desc:   "auto/unsupported writer",
			stdout: new(bytes.Buffer),
			want:   false,
		},
		{
			desc:   "auto/tty",
			stdout: tty,
			want:   true,
		},
		{
			// Always means always, even if everything else says no.
			desc: "always",
			env: map[string]string{
				"TERM":     "dumb",
				"NO_COLOR": "1",
			},
			stdout: new(bytes.Buffer),
			color:  colorOutputAlways,
			want:   true,
		},
		{
			desc:  "never",
			color: colorOutputNever,
			want:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			if tt.stdout == nil {
				tt.stdout = io.Discard
			}

			cmd := &mainCmd{
				Stdin:  bytes.NewReader(nil),
				Stdout: tt.stdout,
				Stderr: io.Discard,
				Getwd:  os.Getwd,
				Getenv: func(key string) string {
					return tt.env[key]
				},
			}
			assert.Equal(t, tt.want, cmd.shouldColor(&params{
				ColorOutput: tt.color,
			}))
		})
	}
}

func TestMain_diffColor(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	summary := filepath.Join(dir, "summary.md")
	require.NoError(t,
		os.WriteFile(summary, []byte("- [foo](foo.md)"), 0o644))
	require.NoError(t,
		os.WriteFile(filepath.Join(dir, "foo.md"), []byte("stuff\n"), 0o644))

	output := filepath.Join(dir, "out.md")
	require.NoError(t,
		os.WriteFile(output, []byte("old"), 0o644))

	var stdout, stderr bytes.Buffer
	defer func() {
		assert.Empty(t, stderr.String(), "stdout")
	}()
	err := (&mainCmd{
		Stdin:  bytes.NewReader(nil),
		Stdout: &stdout,
		Stderr: &stderr,
		Getwd: func() (string, error) {
			return dir, nil
		},
		Getenv: nopGetenv,
	}).run(&params{
		Input:       summary,
		Output:      output,
		Diff:        true,
		ColorOutput: colorOutputAlways,
	})
	require.NoError(t, err)

	assert.Contains(t, stdout.String(), "\x1b[32m") // green
	assert.Contains(t, stdout.String(), "\x1b[31m") // red
}

func TestDiffWriter(t *testing.T) {
	t.Parallel()

	// If the file does not exist,
	// we should consider it to be empty.
	t.Run("does not exist", func(t *testing.T) {
		t.Parallel()

		w, err := newDiffWriter("does-not-exist.md", false)
		require.NoError(t, err)

		io.WriteString(w, "hello world")

		var buf bytes.Buffer
		assert.NoError(t, w.Diff(&buf))
		assert.Contains(t, buf.String(), "+hello world")
	})

	// Actually diff the file if it exists.
	t.Run("exists", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "test")
		require.NoError(t,
			os.WriteFile(path, []byte("hello\nfoo"), 0o644))

		w, err := newDiffWriter(path, false)
		require.NoError(t, err)

		io.WriteString(w, "hello\nbar")

		var buf bytes.Buffer
		assert.NoError(t, w.Diff(&buf))
		assert.Contains(t, buf.String(), "-foo")
		assert.Contains(t, buf.String(), "+bar")
	})

	// There should be no output if the file is unchanged.
	t.Run("unchanged", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "test")
		require.NoError(t,
			os.WriteFile(path, []byte("hello world"), 0o644))

		w, err := newDiffWriter(path, false)
		require.NoError(t, err)

		io.WriteString(w, "hello world")

		var buf bytes.Buffer
		assert.NoError(t, w.Diff(&buf))
		assert.Empty(t, buf.String())
	})
}

func assertNoSuchFileError(t *testing.T, str string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		assert.Contains(t, str, "The system cannot find the file specified.")
	} else {
		assert.Contains(t, str, "no such file or directory")
	}
}

func nopGetenv(string) string {
	return ""
}
