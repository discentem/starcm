package main

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/deck"
	"github.com/google/deck/backends/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func WalkExamples(t *testing.T, testDir string) error {
	return filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".expect") {
			t.Run(path, func(t *testing.T) {
				t.Log("Running starcm with", strings.TrimSuffix(path, ".expect"))
				l := log.Default()
				l.SetFlags(log.LUTC)
				w := bytes.Buffer{}
				fsys := afero.NewOsFs()
				// .expect files should be named thing.star.expect so that TrimSuffix removes the .expect
				err := run(strings.TrimSuffix(path, ".expect"), 1, false, fsys, []deck.Backend{logger.Init(&w, l.Flags())})
				assert.NoError(t, err)
				expected, err := afero.ReadFile(fsys, path)
				assert.NoError(t, err)

				assert.Equal(t, string(expected), w.String())
			})
		}
		return nil
	})

}

func TestExamples(t *testing.T) {
	testDir := "examples"
	// Start an HTTP server on port 8080 as some of the examples require it
	s := &http.Server{Addr: ":8080"}

	http.Handle("/", http.FileServer(http.Dir(testDir)))
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	err := WalkExamples(t, testDir)
	assert.NoError(t, err)
}
