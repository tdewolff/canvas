package canvas

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	unipdf "github.com/unidoc/unidoc/pdf/model"
)

func ParseLaTeX(s string) (*Path, error) {
	document := `\documentclass{article}
\begin{document}
$` + s + `$
\end{document}`

	tmpDir, err := ioutil.TempDir("", "tdewolff-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("pdflatex", "-jobname=canvas", "-halt-on-error")
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader(document)

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	pdf, err := ioutil.ReadFile(path.Join(tmpDir, "canvas.pdf"))
	if err != nil {
		return nil, err
	}

	pdfReader, err := unipdf.NewPdfReader(bytes.NewReader(pdf))
	if err != nil {
		return nil, err
	}

	fmt.Println(pdfReader.GetNumPages())

	page, err := pdfReader.GetPage(1)
	if err != nil {
		return nil, err
	}

	fmt.Println(page.GetContentStreams())
	panic("not implemented")
	return nil, nil
}
