package logging

import (
	"bytes"
	"log"
	"testing"

	"github.com/google/deck"
	"github.com/google/deck/backends/logger"
)

func TestLogKeepsPercentSigns(t *testing.T) {
	var buf bytes.Buffer
	deck.Add(logger.Init(&buf, log.Lmsgprefix))
	defer deck.Close()

	Log("100% done", nil, "info", "skipping %s", "write(label=\"50%\")")

	want := "INFO: [100% done]: skipping write(label=\"50%\")\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
