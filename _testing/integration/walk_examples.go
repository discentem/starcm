package integrationtest

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func WalkExamples(t *testing.T, starcmBin string, testDir string) error {
	return filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".expect") {
			t.Run(path, func(t *testing.T) {
				cmd := exec.Command(
					"go",
					"run",
					starcmBin,
					"--root_file",
					strings.TrimSuffix(path, ".expect"),
					"--timestamps=false",
				)
				// .expect files should be named thing.star.expect so that TrimSuffix removes the .expect
				t.Log("Running starcm with", strings.TrimSuffix(path, ".expect"))
				actual, err := cmd.CombinedOutput()
				assert.NoError(t, err)
				expected, err := ioutil.ReadFile(path)
				assert.NoError(t, err)

				assert.Equal(t, string(expected), string(actual))
			})
		}
		return nil
	})

}
