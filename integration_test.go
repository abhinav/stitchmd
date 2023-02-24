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

			for filename, content := range tt.Files {
				path := filepath.Join(dir, filename)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
			}

			var got, stderr bytes.Buffer
			cmd := mainCmd{
				Stdin:  strings.NewReader(tt.Give),
				Stdout: &got,
				Stderr: &stderr,
				Getwd: func() (string, error) {
					t.Errorf("did not expect Getwd to be called")
					return dir, nil
				},
			}

			require.NoError(t, cmd.run(&params{
				Dir: dir,
			}))

			assert.Equal(t, tt.Want, got.String())
			assert.Empty(t, stderr.String(), "stderr")
		})
	}
}
