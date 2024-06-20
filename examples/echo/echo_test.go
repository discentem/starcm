package echo

import (
	"flag"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEcho(t *testing.T) {
	starcm := flag.String("path_to_starcm_main.go", "../../main.go", "Path to the main.go file of the StarCM project")

	cmd := exec.Command(
		"go",
		"run",
		*starcm,
		"--root_file",
		"echo.star",
		"--timestamps=false",
	)
	b, err := cmd.CombinedOutput()
	assert.NoError(t, err)

	expected := `INFO: starting starcm...
INFO: [hello_from_starcm]: Executing...
hello from echo.star!
`

	assert.Equal(t, expected, string(b))
}
