package examples

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAll(t *testing.T) {
	starcm := flag.String("path_to_starcm_main.go", "../../main.go", "Path to the main.go file of the StarCM project")
	starcmFile := "../../examples/download/download_and_load.star"
	t.Run(starcmFile, func(t *testing.T) {

		// Start http server
		go func() {
			http.Handle("/", http.FileServer(http.Dir("../../examples")))
			if err := http.ListenAndServe(":8080", nil); err != nil {
				log.Fatal(err)
			}
		}()

		cmd := exec.Command(
			"go",
			"run",
			*starcm,
			"--root_file",
			starcmFile,
			"--timestamps=false",
			"--inmem_downloads=true",
		)
		t.Log("Running starcm with", starcmFile)
		actual, err := cmd.CombinedOutput()
		assert.NoError(t, err)

		expected, err := os.ReadFile(fmt.Sprintf("%s.expected", starcmFile))
		assert.NoError(t, err)

		assert.Equal(t, string(expected), string(actual))
	})
}
