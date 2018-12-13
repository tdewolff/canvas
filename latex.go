package canvas

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	pdf "rsc.io/pdf"
)

var ErrBadPDF = errors.New("unexpected PDF content")

func valueKind(vk pdf.ValueKind) string {
	switch vk {
	case pdf.Null:
		return "Null"
	case pdf.Bool:
		return "Bool"
	case pdf.Integer:
		return "Integer"
	case pdf.Real:
		return "Real"
	case pdf.String:
		return "String"
	case pdf.Name:
		return "Name"
	case pdf.Dict:
		return "Dict"
	case pdf.Array:
		return "Array"
	case pdf.Stream:
		return "Stream"
	}
	return "?"
}

func printValue(indent, key string, v pdf.Value) {
	fmt.Println(indent, key+": ", valueKind(v.Kind()), v)
	if v.Kind() == pdf.Dict || v.Kind() == pdf.Stream {
		for _, key := range v.Keys() {
			printValue(indent+"  ", key, v.Key(key))
		}
	}
	if v.Kind() == pdf.Stream {
		s, _ := ioutil.ReadAll(v.Reader())
		fmt.Println(indent+"  stream", len(s))

	}
}

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

	r, err := pdf.Open(path.Join(tmpDir, "canvas.pdf"))
	//b, err := ioutil.ReadFile(path.Join(tmpDir, "canvas.pdf"))
	if err != nil {
		return nil, err
	}

	if r.NumPage() != 1 {
		return nil, ErrBadPDF
	}

	content := r.Page(1).Content()
	res := r.Page(1).Resources()
	if res.Kind() != pdf.Dict {
		return nil, ErrBadPDF
	}

	fmt.Println("Content:", content.Text)
	fmt.Println(content.Text[0])
	printValue("", "", res)

	//pdfReader, err := unipdf.NewPdfReader(bytes.NewReader(pdf))
	//if err != nil {
	//	return nil, err
	//}

	//fmt.Println(pdfReader.GetNumPages())

	//page, err := pdfReader.GetPage(1)
	//if err != nil {
	//	return nil, err
	//}

	//fmt.Printf("%+v\n", page)
	//fmt.Println(page.GetAllContentStreams())
	//fmt.Println(page.GetContentStreams())
	return nil, nil
}
