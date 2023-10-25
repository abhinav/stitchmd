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

		Offset  int    `yaml:"offset"`  // -offset
		NoTOC   bool   `yaml:"no-toc"`  // -no-toc
		Preface string `yaml:"preface"` // -preface
		Unsafe  bool   `yaml:"unsafe"`  // -unsafe

		// Directory to run the command in.
		// summary and preface are stored in this directory.
		// Other files are stored in the paths they specify.
		Dir string `yaml:"dir"`

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
			cwd := dir
			if tt.Dir != "" {
				cwd = filepath.Join(dir, tt.Dir)
				require.NoError(t, os.MkdirAll(cwd, 0o755))
			}

			input := filepath.Join(cwd, "summary.md")
			require.NoError(t, os.WriteFile(input, []byte(tt.Give), 0o644))

			output := filepath.Join(cwd, "output.md")
			if tt.OutDir != "" {
				outDir := filepath.FromSlash(tt.OutDir)
				output = filepath.Join(cwd, outDir, "output.md")
			}

			var preface string
			if tt.Preface != "" {
				preface = filepath.Join(cwd, "preface.md")
				require.NoError(t, os.WriteFile(preface, []byte(tt.Preface), 0o644))
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
				Input:   input,
				Output:  output,
				Offset:  tt.Offset,
				NoTOC:   tt.NoTOC,
				Preface: preface,
				Unsafe:  tt.Unsafe,
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
		Name    string            `yaml:"name"`
		Give    string            `yaml:"give"`
		Files   map[string]string `yaml:"files,omitempty"`
		Preface string            `yaml:"preface"` // -preface
		Old     *string           `yaml:"old,omitempty"`
		Diff    string            `yaml:"diff,omitempty"`
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

			var preface string
			if tt.Preface != "" {
				preface = filepath.Join(dir, "preface.md")
				require.NoError(t, os.WriteFile(preface, []byte(tt.Preface), 0o644))
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
				Input:   input,
				Output:  output,
				Preface: preface,
				Diff:    true,
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

func TestIntegration_errors(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Name string `yaml:"name"`

		// Summary file contents.
		Give string `yaml:"give"`

		// Directory to run the command in.
		Dir string `yaml:"dir"`

		// Files to create in the test directory.
		Files map[string]string `yaml:"files,omitempty"`

		// Expected error messages.
		Want []string `yaml:"want"`
	}

	groups := decodeTestGroups[testCase](t, "testdata/errors.yaml")
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

			require.NotEmpty(t, tt.Want, "test case must have at least one error")

			dir := t.TempDir()
			cwd := dir
			if tt.Dir != "" {
				cwd = filepath.Join(dir, tt.Dir)
				require.NoError(t, os.MkdirAll(cwd, 0o755))
			}

			input := filepath.Join(cwd, "summary.md")
			require.NoError(t, os.WriteFile(input, []byte(tt.Give), 0o644))

			for filename, content := range tt.Files {
				path := filepath.Join(dir, filename)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
			}

			var stdout, stderr bytes.Buffer
			defer func() {
				if t.Failed() {
					t.Logf("stdout:\n%s", stdout.String())
				}
			}()

			cmd := mainCmd{
				Stdin:  new(bytes.Buffer),
				Stdout: &stdout,
				Stderr: &stderr,
				Getwd: func() (string, error) {
					return cwd, nil
				},
				Getenv: nopGetenv,
			}

			err := cmd.run(&params{Input: input})
			require.Error(t, err)

			got := stderr.String()
			for _, want := range tt.Want {
				if want, ok := strings.CutPrefix(want, "/"); ok {
					assert.Regexp(t, want, got)
				} else {
					assert.Contains(t, got, want)
				}
			}
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
