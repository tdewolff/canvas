package startex_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

func TestStarTex(t *testing.T) {
	tests := []struct {
		name string
		tex  string
	}{
		{`sum-text`, `y = \sum_{i=0}^{100} f(x_i)`},
		{`sum-disp`, `$y = \sum_{i=0}^{100} f(x_i)$`},
		{`int-text`, `y = \int_{i=0}^{100} f(x_i)`},
		{`int-disp`, `$y = \int_{i=0}^{100} f(x_i)$`},
		{`ops-text`, `y = \prod_i^j \coprod \int \oint \bigcap \bigcup`},
		{`ops-disp`, `$y = \prod_i^j \coprod \int \oint \bigcap \bigcup$`},
		{`ops2-text`, `y = \bigsqcup \bigvee \bigwedge \bigodot \bigotimes \bigoplus \biguplus`},
		{`ops2-disp`, `$y = \bigsqcup \bigvee \bigwedge \bigodot \bigotimes \bigoplus \biguplus$`},
		{`lb-sum-text`, `y = \left( \sum_{i=0}^{100} f(x_i) \right)`},
		{`lb-sum-disp`, `$y = \left( \sum_{i=0}^{100} f(x_i) \right)$`},
		{`parens-all`, `$\left(\vbox to 27pt{}\left(\vbox to 24pt{}\left(\vbox to 21pt{}
\Biggl(\biggl(\Bigl(\bigl(({\scriptstyle({\scriptscriptstyle(\hskip3pt
)})})\bigr)\Bigr)\biggr)\Biggr)\right)\right)\right)$`},
		{`brackets-all`, `$\left[\vbox to 27pt{}\left[\vbox to 24pt{}\left[\vbox to 21pt{}
\Biggl[\biggl[\Bigl[\bigl[{\scriptstyle[{\scriptscriptstyle[\hskip3pt
]}]}]\bigr]\Bigr]\biggr]\Biggr]\right]\right]\right]$`},
		{`braces-all`, `$\left\{\vbox to 27pt{}\left\{\vbox to 24pt{}\left\{\vbox to 21pt{}
\Biggl\{\biggl\{\Bigl\{\bigl\{\{{\scriptstyle\{{\scriptscriptstyle\{\hskip3pt
\}}\}}\}\bigr\}\Bigr\}\biggr\}\Biggr\}\right\}\right\}\right\}$`},
		{`sqrt-all`, `$\sqrt{1+\sqrt{1+\sqrt{1+\sqrt{1+\sqrt{1+\sqrt{1+\sqrt{1+x}}}}}}}$`},
		{`frac-text`, `a = \left( \frac{\overline{f}(x^2)}{\prod_i^j \sum_i^j f_i(x_j^2)} \right)`},
		{`frac-disp`, `$a = \left( \frac{\overline{f}(x^2)}{\prod_i^j \sum_i^j f_i(x_j^2)} \right)$`},
		{`partial-text`, `y = \partial x`},
	}

	os.Mkdir("testdata", 0777)

	for _, test := range tests {
		// if test.name != "partial-text" {
		// 	continue
		// }
		src := test.tex
		p, err := canvas.ParseLaTeX(src)
		if err != nil {
			panic(err)
		}
		w := 80.0
		h := 20.0
		c := canvas.New(w, h)
		ctx := canvas.NewContext(c)

		ctx.SetFillColor(canvas.White)
		ctx.DrawPath(0.0, 0.0, canvas.Rectangle(w, h))

		ctx.SetFillColor(canvas.Black)
		x := 1.0
		y := 5.0
		if strings.Contains(test.name, "-all") {
			y = 20
		}

		ctx.DrawPath(x, y, p)

		fname := filepath.Join("testdata", test.name+".png")
		renderers.Write(fname, c, canvas.DPMM(8))
	}
}
