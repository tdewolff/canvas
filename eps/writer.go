package eps

import (
	"io"

	"github.com/tdewolff/canvas"
)

// Writer writes the canvas as an EPS file.
// Be aware that EPS does not support transparency of colors.
func Writer(w io.Writer, c *canvas.Canvas) error {
	eps := New(w, c.W, c.H)
	c.Render(eps)
	return nil
}
