// +build !latex

package canvas

import "github.com/tdewolff/canvas"

func ParseLaTeX(s string) (*Path, error) {
	// TODO: native LaTeX support
	return &canvas.Path{}, nil
}
