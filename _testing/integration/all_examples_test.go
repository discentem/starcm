package integrationtest

import (
	"flag"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testDir   *string
	starcmBin *string
)

func TestAll(t *testing.T) {
	testDir = flag.String("test_dir", "../../examples", "Path to the directory containing the test files")
	starcmBin = flag.String("path_to_starcm_main.go", "../../main.go", "Path to the main.go file of the StarCM project")
	flag.Parse()

	// Start an HTTP server on port 8080
	s := &http.Server{Addr: ":8080"}

	http.Handle("/", http.FileServer(http.Dir(*testDir)))
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	err := WalkExamples(t, *starcmBin, *testDir)
	assert.NoError(t, err)
}
