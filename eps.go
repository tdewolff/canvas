package canvas

import (
	"fmt"
	"image/color"
	"io"
)

var psEllipseDef = `/ellipse {
/rot exch def
/endangle exch def
/startangle exch def
/yrad exch def
/xrad exch def
/y exch def
/x exch def
/savematrix matrix currentmatrix def
x y translate
rot rotate
xrad yrad scale
0 0 1 startangle endangle arc
savematrix setmatrix
} def /ellipsen {
/rot exch def
/endangle exch def
/startangle exch def
/yrad exch def
/xrad exch def
/y exch def
/x exch def
/savematrix matrix currentmatrix def
x y translate
rot rotate
xrad yrad scale
0 0 1 startangle endangle arcn
savematrix setmatrix
} def`

type epsWriter struct {
	io.Writer
	color color.RGBA
}

func newEPSWriter(writer io.Writer, width, height float64) *epsWriter {
	w := &epsWriter{
		Writer: writer,
		color:  Black,
	}

	fmt.Fprintf(w, "%%!PS-Adobe-3.0 EPSF-3.0\n%%%%BoundingBox: 0 0 %v %v\n", dec(width), dec(height))
	fmt.Fprintf(w, psEllipseDef)

	// TODO: (EPS) generate and add preview

	return w
}

func (w *epsWriter) SetColor(color color.RGBA) {
	if color != w.color {
		fmt.Fprintf(w, " %v %v %v setrgbcolor", dec(float64(color.R)/255.0), dec(float64(color.G)/255.0), dec(float64(color.B)/255.0))
		w.color = color
	}
}

func (w *epsWriter) Close() error {
	return nil
}

func (l pathLayer) WriteEPS(w *epsWriter) {
	// TODO: (EPS) test ellipse, rotations etc
	// TODO: (EPS) add drawState support
	w.SetColor(l.fillColor)
	w.Write([]byte(" "))
	w.Write([]byte(l.path.ToPS()))
	w.Write([]byte(" fill"))
}

func (l textLayer) WriteEPS(w *epsWriter) {
	// TODO: (EPS) write text natively
	paths, colors := l.text.ToPaths()
	for i, path := range paths {
		state := defaultDrawState
		state.fillColor = colors[i]
		pathLayer{path.Transform(l.m), state, false}.WriteEPS(w)
	}
}

func (l imageLayer) WriteEPS(w *epsWriter) {
	// TODO: (EPS) write image
}
