package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestIntegration_e2e(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Name  string            `yaml:"name"`
		Give  string            `yaml:"give"`
		Files map[string]string `yaml:"files,omitempty"`
		Want  string            `yaml:"want"`

		Offset int  `yaml:"offset"` // -offset
		NoTOC  bool `yaml:"no-toc"` // -no-toc

		// Path to the output directory,
		// relative to the test directory.
		OutDir string `yaml:"outDir,omitempty"`
	}

	groups := decodeTestGroups[testCase](t, "testdata/e2e/*.yaml")
	var tests []testCase
	for _, group := range groups {
		for _, tt := range group.Tests {
			tt.Name = fmt.Sprintf("%s/%s", group.Name, tt.Name)
			tests = append(tests, tt)
		}
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			input := filepath.Join(dir, "summary.md")
			require.NoError(t, os.WriteFile(input, []byte(tt.Give), 0o644))

			output := filepath.Join(dir, "output.md")
			if tt.OutDir != "" {
				outDir := filepath.FromSlash(tt.OutDir)
				output = filepath.Join(dir, outDir, "output.md")
			}

			for filename, content := range tt.Files {
				path := filepath.Join(dir, filename)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
			}

			var stdout, stderr bytes.Buffer
			defer func() {
				if t.Failed() {
					t.Logf("stdout:\n%s", stdout.String())
					t.Logf("stderr:\n%s", stderr.String())
				}
			}()

			cmd := mainCmd{
				Stdin:  new(bytes.Buffer),
				Stdout: &stdout,
				Stderr: &stderr,
				Getwd: func() (string, error) {
					return dir, nil
				},
				Getenv: nopGetenv,
			}

			require.NoError(t, cmd.run(&params{
				Input:  input,
				Output: output,
				Offset: tt.Offset,
				NoTOC:  tt.NoTOC,
			}))

			got, err := os.ReadFile(output)
			require.NoError(t, err)

			assert.Equal(t, tt.Want, string(got))
			assert.Empty(t, stderr.String(), "stderr")
			assert.Empty(t, stdout.String(), "stdout")
		})
	}
}

func TestIntegration_diff(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Name  string            `yaml:"name"`
		Give  string            `yaml:"give"`
		Files map[string]string `yaml:"files,omitempty"`
		Old   *string           `yaml:"old,omitempty"`
		Diff  string            `yaml:"diff,omitempty"`
	}

	groups := decodeTestGroups[testCase](t, "testdata/diff.yaml")
	var tests []testCase
	for _, group := range groups {
		tests = append(tests, group.Tests...)
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			input := filepath.Join(dir, "summary.md")
			require.NoError(t, os.WriteFile(input, []byte(tt.Give), 0o644))

			output := filepath.Join(dir, "output.md")
			if tt.Old != nil {
				require.NoError(t,
					os.WriteFile(output, []byte(*tt.Old), 0o644))
			}

			for filename, content := range tt.Files {
				path := filepath.Join(dir, filename)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
			}

			var stdout, stderr bytes.Buffer
			defer func() {
				if t.Failed() {
					t.Logf("stderr:\n%s", stderr.String())
				}
			}()

			cmd := mainCmd{
				Stdin:  new(bytes.Buffer),
				Stdout: &stdout,
				Stderr: &stderr,
				Getwd: func() (string, error) {
					return dir, nil
				},
				Getenv: nopGetenv,
			}

			require.NoError(t, cmd.run(&params{
				Input:  input,
				Output: output,
				Diff:   true,
			}))

			// Drop the file names from the diff.
			diffLines := strings.Split(stdout.String(), "\n")
			if len(diffLines) > 0 && strings.HasPrefix(diffLines[0], "--- ") {
				diffLines = diffLines[1:]
			}
			if len(diffLines) > 0 && strings.HasPrefix(diffLines[0], "+++ ") {
				diffLines = diffLines[1:]
			}

			got := strings.Join(diffLines, "\n")
			assert.Equal(t, tt.Diff, got)
		})
	}
}

type testGroup[T any] struct {
	Name  string
	Tests []T

	filename string
}

func decodeTestGroups[T any](t testing.TB, glob string) []testGroup[T] {
	t.Helper()

	testfiles, err := filepath.Glob(glob)
	require.NoError(t, err)
	require.NotEmpty(t, testfiles)

	var groups []testGroup[T]
	for _, testfile := range testfiles {
		testdata, err := os.ReadFile(testfile)
		require.NoError(t, err)

		groupname := strings.TrimSuffix(filepath.Base(testfile), ".yaml")

		var tests []T
		require.NoError(t, yaml.Unmarshal(testdata, &tests))
		groups = append(groups, testGroup[T]{
			Name:     groupname,
			Tests:    tests,
			filename: testfile,
		})
	}

	return groups
}
