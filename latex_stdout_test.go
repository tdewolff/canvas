//go:build !latex

package canvas

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/tdewolff/test"
)

func TestLaTeXErrorDoesNotPrint(t *testing.T) {
	// A failing formula must report TeX's log in the error, not print it:
	// callers may be writing their own output (e.g. a PDF) to stdout.
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w
	_, err := ParseLaTeX(`\undefinedcommand x`)
	os.Stdout = stdout
	w.Close()
	printed, _ := io.ReadAll(r)

	test.T(t, err != nil, true, "error")
	test.T(t, strings.Contains(err.Error(), "Undefined control sequence"), true, "TeX log in error")
	test.T(t, string(printed), "", "stdout")
}
