// +build !latex

package canvas

func ParseLaTeX(s string) (*Path, error) {
	// TODO: native LaTeX support
	return &Path{}, nil
}
