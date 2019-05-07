package crawler

import (
	"io"
	"strings"
	"testing"
)

func TestMakeError(t *testing.T) {
	err := makeError("test: %s", "interface")
	if err == nil || err.Error() != "test: interface" {
		t.FailNow()
	}
}

func TestExtractHref(t *testing.T) {
	var r io.Reader
	r = strings.NewReader("<html><body><a href='boom'></body></html>")
	resp := extractHref(r)
	if len(resp) != 1 || resp[0] != "boom" {
		t.FailNow()
	}
}
