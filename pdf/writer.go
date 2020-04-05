package pdf

import (
	"io"

	"github.com/tdewolff/canvas"
)

// Writer writes the canvas as a PDF file.
func Writer(w io.Writer, c *canvas.Canvas) error {
	pdf := canvas.NewPDF(w, c.W, c.H)
	c.Render(pdf)
	return pdf.Close()
}
