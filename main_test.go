package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

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
	}).Run([]string{"-"})
	assert.Equal(t, 1, exitCode)
	assertNoSuchFileError(t, stderr.String())
	assert.Contains(t, stderr.String(), "error reading markdown")
}

func TestDiffWriter(t *testing.T) {
	t.Parallel()

	// If the file does not exist,
	// we should consider it to be empty.
	t.Run("does not exist", func(t *testing.T) {
		t.Parallel()

		w, err := newDiffWriter("does-not-exist.md")
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

		w, err := newDiffWriter(path)
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

		w, err := newDiffWriter(path)
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
