package tex

import (
	"io"

	"github.com/tdewolff/canvas"
)

// Writer writes the canvas as a TeX file using PGF (\usepackage{pgf}).
// Be aware that TeX/PGF does not support transparency of colors.
func Writer(w io.Writer, c *canvas.Canvas) error {
	tex := canvas.NewTeX(w, c.W, c.H)
	c.Render(tex)
	return tex.Close()
}
