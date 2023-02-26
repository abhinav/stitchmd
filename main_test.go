package main

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
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

func assertNoSuchFileError(t *testing.T, str string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		assert.Contains(t, str, "The system cannot find the file specified.")
	} else {
		assert.Contains(t, str, "no such file or directory")
	}
}
