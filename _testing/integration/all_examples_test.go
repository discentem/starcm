package integrationtest

import (
	"flag"
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
	err := WalkExamples(t, *testDir, *starcmBin)
	assert.NoError(t, err)
}
