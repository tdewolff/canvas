// +build latex

package canvas

// TODO: make LaTeX work for WASM target?

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/strconv"
	"github.com/tdewolff/parse/v2/xml"
)

var execCommand = exec.Command
var tempDir = path.Join(os.TempDir(), "tdewolff-canvas")

// ParseLaTeX parses a LaTeX formatted string into a path. It requires latex and dvisvgm to be installed on the machine.
// The content is surrounded by:
//   \documentclass{article}
//   \begin{document}
//   \thispagestyle{empty}
//   {{input}}
//   \end{document}
func ParseLaTeX(s string) (*Path, error) {
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	hash := fmt.Sprintf("%x", md5.Sum([]byte(s)))

	// fast track to cached paths
	b, err := ioutil.ReadFile(path.Join(tempDir, hash))
	if err == nil {
		p, err := ParseSVG(string(b))
		if err == nil {
			return p, nil
		}
	}

	document := `\documentclass{article}
\begin{document}
\thispagestyle{empty}
` + s + `
\end{document}`

	stdout := &bytes.Buffer{}
	cmd := execCommand("latex", "-jobname="+hash, "-halt-on-error")
	cmd.Dir = tempDir
	cmd.Stdin = strings.NewReader(document)
	cmd.Stdout = stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, stdout.String())
	}

	stdout.Reset()
	cmd = execCommand("dvisvgm", "--no-fonts", hash+".dvi")
	cmd.Dir = tempDir
	cmd.Stdout = stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, stdout.String())
	}

	r, err := os.Open(path.Join(tempDir, hash+".svg"))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	svgPaths := map[string]*Path{}
	x0, y0 := math.Inf(1), math.Inf(1)
	height := 0.0

	p := &Path{}
	l := xml.NewLexer(parse.NewInput(r))
	for {
		tt, _ := l.Next()
		switch tt {
		case xml.ErrorToken:
			if l.Err() != io.EOF {
				return nil, l.Err()
			}
			p = p.Transform(Identity.Translate(-x0, y0+height).ReflectY())
			//_ = ioutil.WriteFile(path.Join(tempDir, hash), []byte(p.String()), 0644)
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

				p = p.Append(svgPath.Translate(x, y))
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
				p = p.Append(Rectangle(w, h).Translate(x, y))
				x0 = math.Min(x0, x)
				y0 = math.Min(y0, y)
			}
		}
	}
}
