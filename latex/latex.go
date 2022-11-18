package latex

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/go-fonts/latin-modern/lmmath"
	"github.com/go-fonts/latin-modern/lmmono10regular"
	"github.com/go-fonts/latin-modern/lmmono12regular"
	"github.com/go-fonts/latin-modern/lmmono8regular"
	"github.com/go-fonts/latin-modern/lmmono9regular"
	"github.com/go-fonts/latin-modern/lmmonocaps10regular"
	"github.com/go-fonts/latin-modern/lmmonoslant10regular"
	"github.com/go-fonts/latin-modern/lmroman10bold"
	"github.com/go-fonts/latin-modern/lmroman10bolditalic"
	"github.com/go-fonts/latin-modern/lmroman10italic"
	"github.com/go-fonts/latin-modern/lmroman10regular"
	"github.com/go-fonts/latin-modern/lmroman12bold"
	"github.com/go-fonts/latin-modern/lmroman12italic"
	"github.com/go-fonts/latin-modern/lmroman12regular"
	"github.com/go-fonts/latin-modern/lmroman17regular"
	"github.com/go-fonts/latin-modern/lmroman5bold"
	"github.com/go-fonts/latin-modern/lmroman5regular"
	"github.com/go-fonts/latin-modern/lmroman6bold"
	"github.com/go-fonts/latin-modern/lmroman6regular"
	"github.com/go-fonts/latin-modern/lmroman7bold"
	"github.com/go-fonts/latin-modern/lmroman7italic"
	"github.com/go-fonts/latin-modern/lmroman7regular"
	"github.com/go-fonts/latin-modern/lmroman8bold"
	"github.com/go-fonts/latin-modern/lmroman8italic"
	"github.com/go-fonts/latin-modern/lmroman8regular"
	"github.com/go-fonts/latin-modern/lmroman9bold"
	"github.com/go-fonts/latin-modern/lmroman9italic"
	"github.com/go-fonts/latin-modern/lmroman9regular"
	"github.com/go-fonts/latin-modern/lmromancaps10regular"
	"github.com/go-fonts/latin-modern/lmromandunh10regular"
	"github.com/go-fonts/latin-modern/lmromanslant10regular"
	"github.com/go-fonts/latin-modern/lmromanslant12regular"
	"github.com/go-fonts/latin-modern/lmromanslant17regular"
	"github.com/go-fonts/latin-modern/lmromanslant8regular"
	"github.com/go-fonts/latin-modern/lmromanslant9regular"
	"github.com/go-fonts/latin-modern/lmromanunsl10regular"
	"github.com/go-fonts/latin-modern/lmsans10bold"
	"github.com/go-fonts/latin-modern/lmsans10oblique"
	"github.com/go-fonts/latin-modern/lmsans10regular"
	"github.com/go-fonts/latin-modern/lmsans12oblique"
	"github.com/go-fonts/latin-modern/lmsans12regular"
	"github.com/go-fonts/latin-modern/lmsans17oblique"
	"github.com/go-fonts/latin-modern/lmsans17regular"
	"github.com/go-fonts/latin-modern/lmsans8oblique"
	"github.com/go-fonts/latin-modern/lmsans8regular"
	"github.com/go-fonts/latin-modern/lmsans9oblique"
	"github.com/go-fonts/latin-modern/lmsans9regular"
	"github.com/go-fonts/latin-modern/lmsansdemicond10regular"
	"github.com/go-fonts/latin-modern/lmsansquot8oblique"
	"github.com/tdewolff/canvas"
	canvasFont "github.com/tdewolff/canvas/font"
	"star-tex.org/x/tex"
)

const mmPerPt = 25.4 / 72.0

type Family int

const (
	LatinModern Family = iota
)

var preamble = `\nopagenumbers`

func Parse(formula string, family Family) (*canvas.Path, string, error) {
	r := strings.NewReader(fmt.Sprintf(`%s $%s$`, preamble, formula))
	w := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	engine := tex.NewEngine(stdout, bytes.NewReader([]byte{}))
	if err := engine.Process(w, r); err != nil {
		return nil, string(stdout.Bytes()), err
	}

	p, err := DVI2Path(w.Bytes(), newFonts(family))
	if err != nil {
		return nil, string(stdout.Bytes()), err
	}
	return p, string(stdout.Bytes()), nil
}

type fonts struct {
	family Family
	font   map[string]*font
}

type font struct {
	sfnt   *canvasFont.SFNT
	size   float64
	italic bool
}

func newFonts(family Family) *fonts {
	return &fonts{
		family: family,
		font:   map[string]*font{},
	}
}

func (fs *fonts) Get(name string, scale float64) Font {
	i := 0
	for i < len(name) && 'a' <= name[i] && name[i] <= 'z' {
		i++
	}
	fontname := name[:i]
	fontsize := 10.0
	if ifontsize, err := strconv.Atoi(name[i:]); err == nil {
		fontsize = float64(ifontsize)
	}

	f, ok := fs.font[name]
	if !ok {
		var fontSizes map[float64][]byte
		//if fs.family == STIX {
		//} else {
		// default is Latin Modern
		switch fontname {
		//case "cmbx":
		//case "cmssbx":
		case "cmmi", "cmti":
			fontSizes = map[float64][]byte{
				12.0: lmroman12italic.TTF,
				10.0: lmroman10italic.TTF,
				9.0:  lmroman9italic.TTF,
				8.0:  lmroman8italic.TTF,
				7.0:  lmroman7italic.TTF,
			}
		case "cmss":
			fontSizes = map[float64][]byte{
				17.0: lmsans17regular.TTF,
				12.0: lmsans12regular.TTF,
				10.0: lmsans10regular.TTF,
				9.0:  lmsans9regular.TTF,
				8.0:  lmsans8regular.TTF,
			}
		case "cmssqi":
			fontSizes = map[float64][]byte{
				8.0: lmsansquot8oblique.TTF,
			}
		case "cmssi":
			fontSizes = map[float64][]byte{
				17.0: lmsans17oblique.TTF,
				12.0: lmsans12oblique.TTF,
				10.0: lmsans10oblique.TTF,
				9.0:  lmsans9oblique.TTF,
				8.0:  lmsans8oblique.TTF,
			}
		case "cmssb":
			fontSizes = map[float64][]byte{
				10.0: lmsans10bold.TTF,
			}
		case "cmssdc":
			fontSizes = map[float64][]byte{
				10.0: lmsansdemicond10regular.TTF,
			}
		case "cmb":
			fontSizes = map[float64][]byte{
				12.0: lmroman12bold.TTF,
				10.0: lmroman10bold.TTF,
				9.0:  lmroman9bold.TTF,
				8.0:  lmroman8bold.TTF,
				7.0:  lmroman7bold.TTF,
				6.0:  lmroman6bold.TTF,
				5.0:  lmroman5bold.TTF,
			}
		case "cmtt":
			fontSizes = map[float64][]byte{
				12.0: lmmono12regular.TTF,
				10.0: lmmono10regular.TTF,
				9.0:  lmmono9regular.TTF,
				8.0:  lmmono8regular.TTF,
			}
		case "cmsltt":
			fontSizes = map[float64][]byte{
				10.0: lmmonoslant10regular.TTF,
			}
		case "cmsl":
			fontSizes = map[float64][]byte{
				17.0: lmromanslant17regular.TTF,
				12.0: lmromanslant12regular.TTF,
				10.0: lmromanslant10regular.TTF,
				9.0:  lmromanslant9regular.TTF,
				8.0:  lmromanslant8regular.TTF,
			}
		case "cmu":
			fontSizes = map[float64][]byte{
				10.0: lmromanunsl10regular.TTF,
			}
		case "cmmib":
			fontSizes = map[float64][]byte{
				10.0: lmroman10bolditalic.TTF,
			}
		case "cmtcsc":
			fontSizes = map[float64][]byte{
				10.0: lmmonocaps10regular.TTF,
			}
		case "cmcsc":
			fontSizes = map[float64][]byte{
				10.0: lmromancaps10regular.TTF,
			}
		case "cmdunh":
			fontSizes = map[float64][]byte{
				10.0: lmromandunh10regular.TTF,
			}
		case "cmsy", "cmex", "cmbsy":
			fontSizes = map[float64][]byte{
				fontsize: lmmath.TTF,
			}
		default:
			// cmr
			if fontname != "cmr" {
				fmt.Println("WARNING: unknown font", fontname)
			}
			fontSizes = map[float64][]byte{
				17.0: lmroman17regular.TTF,
				12.0: lmroman12regular.TTF,
				10.0: lmroman10regular.TTF,
				9.0:  lmroman9regular.TTF,
				8.0:  lmroman8regular.TTF,
				7.0:  lmroman7regular.TTF,
				6.0:  lmroman6regular.TTF,
				5.0:  lmroman5regular.TTF,
			}
		}
		//}

		// select closest matching font size
		var data []byte
		var size float64
		for isize, idata := range fontSizes {
			if data == nil || math.Abs(isize-fontsize) < math.Abs(size-fontsize) {
				data = idata
				size = isize
			}
		}

		// load font
		sfnt, err := canvasFont.ParseSFNT(data, 0)
		if err != nil {
			fmt.Println("ERROR: %w", err)
		}

		// calculate size correction if the found font has a different font size than requested
		fsize := scale * fontsize * mmPerPt / float64(sfnt.Head.UnitsPerEm)
		fsizeCorr := fontsize / size
		isItalic := 0 < len(name) && name[len(name)-1] == 'i'

		f = &font{sfnt, fsizeCorr * fsize, isItalic}
		fs.font[name] = f
	}
	return f
}

func (f *font) Draw(p canvasFont.Pather, x, y float64, r rune) float64 {
	gid := f.sfnt.GlyphIndex(r)
	xoffset := int32(x / f.size)
	yoffset := int32(y / f.size)
	if f.italic {
		xoffset -= int32(float64(f.sfnt.OS2.SxHeight) / 2.0 * math.Tan(-float64(f.sfnt.Post.ItalicAngle)*math.Pi/180.0))
	}
	_ = f.sfnt.GlyphPath(p, gid, 0, xoffset, yoffset, f.size, canvasFont.NoHinting)
	return f.size * float64(f.sfnt.GlyphAdvance(gid)) // in mm
}
