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

func TestIntegration(t *testing.T) {
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

	testfiles, err := filepath.Glob("testdata/integration/*.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, testfiles)

	var allTests []testCase
	for _, testfile := range testfiles {
		testdata, err := os.ReadFile(testfile)
		require.NoError(t, err)

		groupname := strings.TrimSuffix(filepath.Base(testfile), ".yaml")

		var tests []testCase
		require.NoError(t, yaml.Unmarshal(testdata, &tests))
		for i, tt := range tests {
			tt.Name = fmt.Sprintf("%s/%s", groupname, tt.Name)
			tests[i] = tt
		}
		allTests = append(allTests, tests...)
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
