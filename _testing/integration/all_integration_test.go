package integrationtest

import (
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testDir *string
	starcm  *string
)

func TestMain(m *testing.M) {
	testDir = flag.String("test_dir", "../../examples", "Path to the directory containing the test files")
	starcm = flag.String("path_to_starcm_main.go", "../../main.go", "Path to the main.go file of the StarCM project")
	flag.Parse()
	os.Exit(m.Run())
}

func TestAll(t *testing.T) {
	err := filepath.Walk(*testDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".expect") {
			t.Run(path, func(t *testing.T) {
				cmd := exec.Command(
					"go",
					"run",
					*starcm,
					"--root_file",
					strings.TrimSuffix(path, ".expect"),
					"--timestamps=false",
				)
				t.Log("Running starcm with", path)
				actual, err := cmd.CombinedOutput()
				assert.NoError(t, err)

				expected, err := ioutil.ReadFile(path)
				assert.NoError(t, err)

				assert.Equal(t, string(expected), string(actual))
			})
		}
		return nil
	})

	assert.NoError(t, err)
}
