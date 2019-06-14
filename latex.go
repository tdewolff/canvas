package canvas

import (
	"bytes"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/tdewolff/parse/xml"
	"github.com/tdewolff/strconv"
)

// ParseLaTeX parses a LaTeX formatted string into a path. It requires latex and dvisvgm to be installed on the machine.
// The content is surrounded by
//   \documentclass{article}
//   \begin{document}
//   \thispagestyle{empty}
//   {{input}}
//   \end{document}
func ParseLaTeX(s string) (*Path, error) {
	tmpDir, err := ioutil.TempDir("", "tdewolff-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	// Fast track to cached paths
	hash := string(fnv.New32a().Sum([]byte(s)))
	b, err := ioutil.ReadFile(path.Join(tmpDir, hash))
	if err == nil {
		p, err := ParseSVG(string(b))
		if err == nil {
			return p, nil
		}
	}

	stdout := &bytes.Buffer{}
	document := `\documentclass{article}
\begin{document}
\thispagestyle{empty}
` + s + `
\end{document}`

	cmd := exec.Command("latex", "-jobname=canvas", "-halt-on-error")
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader(document)
	cmd.Stdout = stdout
	if err := cmd.Start(); err != nil {
		fmt.Println(stdout.String())
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		fmt.Println(stdout.String())
		return nil, err
	}

	stdout.Reset()

	cmd = exec.Command("dvisvgm", "--no-fonts", "canvas.dvi")
	cmd.Dir = tmpDir
	cmd.Stdout = stdout
	if err := cmd.Start(); err != nil {
		fmt.Println(stdout.String())
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		fmt.Println(stdout.String())
		return nil, err
	}

	r, err := os.Open(path.Join(tmpDir, "canvas.svg"))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	svgPaths := map[string]*Path{}
	x0, y0 := math.Inf(1), math.Inf(1)
	width, height := 0.0, 0.0

	p := &Path{}
	l := xml.NewLexer(r)
	for {
		tt, _ := l.Next()
		switch tt {
		case xml.ErrorToken:
			if l.Err() != io.EOF {
				return nil, l.Err()
			}
			if !p.Empty() {
				_ = width
				p = p.Transform(Identity.ReflectY().Translate(-x0, y0+height))
			}
			_ = ioutil.WriteFile(path.Join(tmpDir, hash), []byte(p.String()), 0644)
			return p, nil
		case xml.StartTagToken:
			tag := string(l.Text())
			attrs := map[string][]byte{}
			for {
				ttAttr, _ := l.Next()
				if ttAttr != xml.AttributeToken {
					break
				}
				val := l.AttrVal()
				if len(val) > 1 && (val[0] == '\'' || val[0] == '"') && val[0] == val[len(val)-1] {
					val = val[1 : len(val)-1]
				}
				attrs[string(l.Text())] = val
			}

			if tag == "svg" {
				var n int
				width, n = strconv.ParseFloat(attrs["width"])
				if n == 0 {
					return nil, errors.New("unexpected SVG format: expected valid width attribute on svg tag")
				}

				height, n = strconv.ParseFloat(attrs["height"])
				if n == 0 {
					return nil, errors.New("unexpected SVG format: expected valid height attribute on svg tag")
				}
			} else if tag == "path" {
				id, ok := attrs["id"]
				if !ok {
					return nil, errors.New("unexpected SVG format: expected id attribute on path tag")
				}

				d, ok := attrs["d"]
				if !ok {
					return nil, errors.New("unexpected SVG format: expected d attribute on path tag")
				}

				svgPath, err := ParseSVG(string(d))
				if err != nil {
					return nil, err
				}
				svgPaths[string(id)] = svgPath
			} else if tag == "use" {
				x, n := strconv.ParseFloat(attrs["x"])
				if n == 0 {
					return nil, errors.New("unexpected SVG format: expected valid x attribute on use tag")
				}

				y, n := strconv.ParseFloat(attrs["y"])
				if n == 0 {
					return nil, errors.New("unexpected SVG format: expected valid y attribute on use tag")
				}

				id, ok := attrs["xlink:href"]
				if !ok || len(id) == 0 || id[0] != '#' {
					return nil, errors.New("unexpected SVG format: expected valid xlink:href attribute on use tag")
				}

				svgPath, ok := svgPaths[string(id[1:])]
				if !ok {
					return nil, errors.New("unexpected SVG format: xlink:href does not point to existing path")
				}

				p.Append(svgPath.Translate(x, y))
				x0 = math.Min(x0, x)
				y0 = math.Min(y0, y)
			} else if tag == "rect" {
				x, n := strconv.ParseFloat(attrs["x"])
				if n == 0 {
					return nil, errors.New("unexpected SVG format: expected valid x attribute on rect tag")
				}

				y, n := strconv.ParseFloat(attrs["y"])
				if n == 0 {
					return nil, errors.New("unexpected SVG format: expected valid y attribute on rect tag")
				}

				w, n := strconv.ParseFloat(attrs["width"])
				if n == 0 {
					return nil, errors.New("unexpected SVG format: expected valid width attribute on rect tag")
				}

				h, n := strconv.ParseFloat(attrs["height"])
				if n == 0 {
					return nil, errors.New("unexpected SVG format: expected valid height attribute on rect tag")
				}
				p.Append(Rectangle(x, y, w, h))
				x0 = math.Min(x0, x)
				y0 = math.Min(y0, y)
			}
		}
	}
	return nil, nil
}

// func valueKind(vk pdf.ValueKind) string {
// 	switch vk {
// 	case pdf.Null:
// 		return "Null"
// 	case pdf.Bool:
// 		return "Bool"
// 	case pdf.Integer:
// 		return "Integer"
// 	case pdf.Real:
// 		return "Real"
// 	case pdf.String:
// 		return "String"
// 	case pdf.Name:
// 		return "Name"
// 	case pdf.Dict:
// 		return "Dict"
// 	case pdf.Array:
// 		return "Array"
// 	case pdf.Stream:
// 		return "Stream"
// 	}
// 	return "?"
// }
//
// func printValue(indent, key string, v pdf.Value) {
// 	fmt.Println(indent, key+": ", valueKind(v.Kind()), v)
// 	if v.Kind() == pdf.Dict || v.Kind() == pdf.Stream {
// 		for _, key := range v.Keys() {
// 			printValue(indent+"  ", key, v.Key(key))
// 		}
// 	}
// 	if v.Kind() == pdf.Stream {
// 		s, _ := ioutil.ReadAll(v.Reader())
// 		fmt.Println(indent+"  stream", len(s))
//
// 	}
// }
//
// func parseLaTeX2(s string) (*Path, error) {
// 	document := `\documentclass{article}
// \begin{document}
// \thispagestyle{empty}
// $` + s + `$
// \end{document}`
//
// 	tmpDir := "."
// 	//tmpDir, err := ioutil.TempDir("", "tdewolff-")
// 	//if err != nil {
// 	//	return nil, err
// 	//}
// 	//defer os.RemoveAll(tmpDir)
//
// 	stdout := &bytes.Buffer{}
//
// 	cmd := exec.Command("pdflatex", "-jobname=canvas", "-halt-on-error")
// 	cmd.Dir = tmpDir
// 	cmd.Stdin = strings.NewReader(document)
// 	cmd.Stdout = stdout
// 	if err := cmd.Start(); err != nil {
// 		fmt.Println(stdout.String())
// 		return nil, err
// 	}
// 	if err := cmd.Wait(); err != nil {
// 		fmt.Println(stdout.String())
// 		return nil, err
// 	}
//
// 	r, err := pdf.Open(path.Join(tmpDir, "canvas.pdf"))
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	if r.NumPage() != 1 {
// 		return nil, fmt.Errorf("bad PDF")
// 	}
//
// 	content := r.Page(1).Content()
// 	res := r.Page(1).Resources()
// 	if res.Kind() != pdf.Dict {
// 		return nil, fmt.Errorf("bad PDF")
// 	}
//
// 	fmt.Println("Content:", content.Text)
// 	fmt.Println(content.Text[0])
// 	printValue("", "", res)
//
// 	b, err := ioutil.ReadAll(res.Key("Font").Key("F11").Key("FontDescriptor").Key("FontFile").Reader())
// 	fmt.Println(err)
// 	font, err := conradSFNT.Parse(bytes.NewReader(b))
// 	fmt.Println(font, err)
//
// 	f, err := os.Open("cmmi10.pfb")
// 	fmt.Println(err)
// 	font, err = conradSFNT.Parse(f)
// 	fmt.Println(font, err)
// 	return nil, nil
// }
