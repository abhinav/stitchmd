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
		Files map[string]string `yaml:"files"`
		Want  string            `yaml:"want"`

		Offset int  `yaml:"offset"` // -offset
		NoTOC  bool `yaml:"no-toc"` // -no-toc

		// Path to the output directory,
		// relative to the test directory.
		OutDir string `yaml:"outDir"`
	}

	groups := decodeTestGroups[testCase](t, "testdata/e2e/*.yaml")
	var allTests []testCase
	for _, group := range groups {
		for _, tt := range group.Tests {
			tt.Name = fmt.Sprintf("%s/%s", group.Name, tt.Name)
			allTests = append(allTests, tt)
		}
	}

	for _, tt := range allTests {
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

type testGroup[T any] struct {
	Name  string
	Tests []T
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
			Name:  groupname,
			Tests: tests,
		})
	}

	return groups
}
