//go:build !latex
// +build !latex

package canvas

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/go-fonts/latin-modern/lmmath"
	"github.com/go-fonts/latin-modern/lmmono10italic"
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
	"github.com/go-fonts/latin-modern/lmromanslant10bold"
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
	"github.com/go-fonts/latin-modern/lmsansquot8regular"
	canvasFont "github.com/tdewolff/font"
	"star-tex.org/x/tex"
)

var preamble = `\nopagenumbers

\def\frac#1#2{{{#1}\over{#2}}}
`

// ParseLaTeX parse a LaTeX formula (that what is between $...$) and returns a path.
func ParseLaTeX(formula string) (*Path, error) {
	r := strings.NewReader(fmt.Sprintf(`%s $%s$ \end{document}`, preamble, formula))
	w := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	engine := tex.New()
	engine.Stdout = stdout
	engine.Stderr = stdout
	engine.Stdin = bytes.NewReader([]byte{})
	if err := engine.Process(w, r); err != nil {
		fmt.Println(stdout.String())
		return nil, err
	}

	p, err := DVI2Path(w.Bytes(), newFonts())
	if err != nil {
		fmt.Println(stdout.String())
		return nil, err
	}
	return p, nil
}

type dviFonts struct {
	font     map[string]*dviFont
	mathSyms *dviFont // always available as backup for any rune
}

type dviFont struct {
	sfnt     *canvasFont.SFNT
	cmap     map[uint32]rune
	size     float64
	italic   bool
	ex       bool     // is it cmex?
	mathSyms *dviFont // always available as backup for any rune
}

func newFonts() *dviFonts {
	return &dviFonts{
		font: map[string]*dviFont{},
	}
}

func (fs *dviFonts) Get(name string, scale float64) DVIFont {
	i := 0
	for i < len(name) && 'a' <= name[i] && name[i] <= 'z' {
		i++
	}
	fontname := name[:i]
	fontsize := 10.0
	if ifontsize, err := strconv.Atoi(name[i:]); err == nil {
		fontsize = float64(ifontsize)
	}

	if fs.mathSyms == nil {
		fs.mathSyms = fs.loadFont("cmsy", cmapCMSY, 10.0, scale, lmmath.TTF)
	}

	cmap := cmapCMR
	f, ok := fs.font[name]
	if !ok {
		var fontSizes map[float64][]byte
		switch fontname {
		case "cmb", "cmbx":
			fontSizes = map[float64][]byte{
				12.0: lmroman12bold.TTF,
				10.0: lmroman10bold.TTF,
				9.0:  lmroman9bold.TTF,
				8.0:  lmroman8bold.TTF,
				7.0:  lmroman7bold.TTF,
				6.0:  lmroman6bold.TTF,
				5.0:  lmroman5bold.TTF,
			}
		case "cmbsy":
			cmap = cmapCMSY
			fontSizes = map[float64][]byte{
				fontsize: lmmath.TTF,
			}
		case "cmbxsl":
			fontSizes = map[float64][]byte{
				fontsize: lmromanslant10bold.TTF,
			}
		case "cmbxti":
			fontSizes = map[float64][]byte{
				10.0: lmroman10bolditalic.TTF,
			}
		case "cmcsc":
			cmap = cmapCMTT
			fontSizes = map[float64][]byte{
				10.0: lmromancaps10regular.TTF,
			}
		case "cmdunh":
			fontSizes = map[float64][]byte{
				10.0: lmromandunh10regular.TTF,
			}
		case "cmex":
			cmap = cmapCMEX
			fontSizes = map[float64][]byte{
				fontsize: lmmath.TTF,
			}
		case "cmitt":
			cmap = cmapCMTT
			fontSizes = map[float64][]byte{
				10.0: lmmono10italic.TTF,
			}
		case "cmmi":
			cmap = cmapCMMI
			fontSizes = map[float64][]byte{
				12.0: lmroman12italic.TTF,
				10.0: lmroman10italic.TTF,
				9.0:  lmroman9italic.TTF,
				8.0:  lmroman8italic.TTF,
				7.0:  lmroman7italic.TTF,
			}
		case "cmmib":
			cmap = cmapCMMI
			fontSizes = map[float64][]byte{
				10.0: lmroman10bolditalic.TTF,
			}
		case "cmr":
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
		case "cmsl":
			fontSizes = map[float64][]byte{
				17.0: lmromanslant17regular.TTF,
				12.0: lmromanslant12regular.TTF,
				10.0: lmromanslant10regular.TTF,
				9.0:  lmromanslant9regular.TTF,
				8.0:  lmromanslant8regular.TTF,
			}
		case "cmsltt":
			fontSizes = map[float64][]byte{
				10.0: lmmonoslant10regular.TTF,
			}
		case "cmss":
			fontSizes = map[float64][]byte{
				17.0: lmsans17regular.TTF,
				12.0: lmsans12regular.TTF,
				10.0: lmsans10regular.TTF,
				9.0:  lmsans9regular.TTF,
				8.0:  lmsans8regular.TTF,
			}
		case "cmssb", "cmssbx":
			fontSizes = map[float64][]byte{
				10.0: lmsans10bold.TTF,
			}
		case "cmssdc":
			fontSizes = map[float64][]byte{
				10.0: lmsansdemicond10regular.TTF,
			}
		case "cmssi":
			fontSizes = map[float64][]byte{
				17.0: lmsans17oblique.TTF,
				12.0: lmsans12oblique.TTF,
				10.0: lmsans10oblique.TTF,
				9.0:  lmsans9oblique.TTF,
				8.0:  lmsans8oblique.TTF,
			}
		case "cmssq":
			fontSizes = map[float64][]byte{
				8.0: lmsansquot8regular.TTF,
			}
		case "cmssqi":
			fontSizes = map[float64][]byte{
				8.0: lmsansquot8oblique.TTF,
			}
		case "cmsy":
			cmap = cmapCMSY
			fontSizes = map[float64][]byte{
				fontsize: lmmath.TTF,
			}
		case "cmtcsc":
			cmap = cmapCMTT
			fontSizes = map[float64][]byte{
				10.0: lmmonocaps10regular.TTF,
			}
		//case "cmtex":
		//cmap = nil
		case "cmti":
			fontSizes = map[float64][]byte{
				12.0: lmroman12italic.TTF,
				10.0: lmroman10italic.TTF,
				9.0:  lmroman9italic.TTF,
				8.0:  lmroman8italic.TTF,
				7.0:  lmroman7italic.TTF,
			}
		case "cmtt":
			cmap = cmapCMTT
			fontSizes = map[float64][]byte{
				12.0: lmmono12regular.TTF,
				10.0: lmmono10regular.TTF,
				9.0:  lmmono9regular.TTF,
				8.0:  lmmono8regular.TTF,
			}
		case "cmu":
			fontSizes = map[float64][]byte{
				10.0: lmromanunsl10regular.TTF,
			}
		//case "cmvtt":
		//cmap = cmapCTT
		default:
			fmt.Println("WARNING: unknown font", fontname)
		}

		// select closest matching font size
		var data []byte
		var size float64
		for isize, idata := range fontSizes {
			if data == nil || math.Abs(isize-fontsize) < math.Abs(size-fontsize) {
				data = idata
				size = isize
			}
		}

		f = fs.loadFont(fontname, cmap, fontsize, scale, data)
		fs.font[name] = f
	}
	return f
}

func (fs *dviFonts) loadFont(fontname string, cmap map[uint32]rune, fontsize, scale float64, data []byte) *dviFont {
	sfnt, err := canvasFont.ParseSFNT(data, 0)
	if err != nil {
		fmt.Println("ERROR: %w", err)
	}

	// calculate size correction if the found font has a different font size than requested
	fsize := scale * fontsize * mmPerPt / float64(sfnt.Head.UnitsPerEm)
	//fsizeCorr := fontsize / size
	isItalic := 0 < len(fontname) && fontname[len(fontname)-1] == 'i'
	fsizeCorr := 1.0
	isEx := fontname == "cmex"
	return &dviFont{sfnt, cmap, fsizeCorr * fsize, isItalic, isEx, fs.mathSyms}
}

func (f *dviFont) Draw(p canvasFont.Pather, x, y float64, cid uint32) float64 {
	face := f.sfnt
	r := f.cmap[cid]
	gid := face.GlyphIndex(r)
	if gid == 0 {
		if f.mathSyms != nil {
			face = f.mathSyms.sfnt
			gid = face.GlyphIndex(r)
			if gid == 0 {
				fmt.Println("rune not found in mathSyms:", string(r))
			}
		} else {
			fmt.Println("rune not found:", string(r))
		}
	}

	if f.italic {
		x -= f.size * float64(face.OS2.SxHeight) / 2.0 * math.Tan(-face.Post.ItalicAngle*math.Pi/180.0)
	}
	size := f.size
	xsc := 1.0
	if f.ex {
		_, ymin, _, ymax := face.GlyphBounds(gid)
		yb := float64(max(ymin, ymax))
		if exsc, has := cmexScales[cid]; has {
			size *= exsc
		}
		switch cid {
		case 0x5A, 0x49: // \int and \oint are off in large size
			yb += 200
		case 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F:
			// larger delims are too thick
			xsc = .7
		case 0x20, 0x21, 0x22, 0x23, 0x28, 0x29, 0x2A, 0x2B:
			// same for even larger ones
			xsc = .6
		case 0x3C, 0x3D: // braces middles need shifting
			yb += 150
		case 0x3A, 0x3B: // braces bottom shifting
			yb += 400
		// below are fixes for all the square root elements
		case 0x71:
			x += size * 80
			xsc = .6
		case 0x72:
			x -= size * 80
			xsc = .6
		case 0x73:
			x -= size * 80
			xsc = .5
		case 0x74:
			yb += 600
		case 0x75:
			x += size * 560
		case 0x76:
			x += size * 400
			yb -= 36
		}
		y -= size * yb
	}
	// note: need a way to separately multiply the x scaling here:
	_ = xsc
	_ = face.GlyphPath(p, gid, 0, x, y, size, canvasFont.NoHinting)
	return size * float64(face.GlyphAdvance(gid)) // in mm
}

const (
	// magnification factors that seem to work: not quite official tex but sorta
	mag1 = 1.2
	mag2 = 1.2 * 1.2
	mag3 = 1.2 * 1.2 * 1.2
	mag4 = 1.2 * 1.2 * 1.2 * 1.2 * 1.2
	mag5 = 3.2
)

// cmexScales has per-character (original ASCII numbers) scaling factors
// for the cmex10 font.
var cmexScales = map[uint32]float64{
	0x00: mag1,
	0x01: mag1,
	0x02: mag1,
	0x03: mag1,
	0x04: mag1,
	0x05: mag1,
	0x06: mag1,
	0x07: mag1,
	0x08: mag1,
	0x0A: mag1,
	0x0B: mag1,
	0x0C: mag1,
	0x0D: mag1,
	0x0E: mag1,
	0x0F: mag1,

	0x10: mag3, // (
	0x11: mag3, // )
	0x12: mag4, // (
	0x13: mag4, // )
	0x14: mag4, // [
	0x15: mag4, // ]
	0x16: mag4, // ⌊
	0x17: mag4, // ⌋
	0x18: mag4, // ⌈
	0x19: mag4, // ⌉
	0x1A: mag4, // {
	0x1B: mag4, // }
	0x1C: mag4, // 〈
	0x1D: mag4, // 〉
	0x1E: mag4, // ∕
	0x1F: mag4, // \

	0x20: mag5, // (
	0x21: mag5, // )
	0x22: mag5, // [
	0x23: mag5, // ]
	0x24: mag5, // ⌊
	0x25: mag5, // ⌋
	0x26: mag5, // ⌈
	0x27: mag5, // ⌉
	0x28: mag5, // {
	0x29: mag5, // }
	0x2A: mag5, // 〈
	0x2B: mag5, // 〉
	0x2C: mag5, // ∕
	0x2D: mag5, // \
	0x2E: mag3, // ∕
	0x2F: mag3, // \

	0x30: mag2, // ⎛
	0x31: mag2, // ⎞
	0x32: mag2, // ⌈
	0x33: mag2, // ⌉
	0x34: mag2, // ⌊
	0x35: mag2, // ⌋
	0x36: mag2, // ⎢
	0x37: mag2, // ⎥

	0x38: mag2, // ⎧ // big braces start
	0x39: mag2, // ⎫
	0x3A: mag2, // ⎩
	0x3B: mag2, // ⎭
	0x3C: mag2, // ⎨
	0x3D: mag2, // ⎬
	0x3E: mag2, // ⎪
	0x3F: mag2, // ∣ ?? unclear

	0x40: mag2, // ⎝ // big parens
	0x41: mag2, // ⎠
	0x42: mag2, // ⎜
	0x43: mag2, // ⎟
	0x44: mag2, // 〈
	0x45: mag2, // 〉
	0x47: mag2, // ⨆
	0x49: mag2, // ∮
	0x4B: mag2, // ⨀
	0x4D: mag2, // ⨁
	0x4F: mag2, // ⨂

	0x58: mag2, // ∑
	0x59: mag2, // ∏
	0x5A: mag2, // ∫
	0x5B: mag2, // ⋃
	0x5C: mag2, // ⋂
	0x5D: mag2, // ⨄
	0x5E: mag2, // ⋀
	0x5F: mag2, // ⋁

	0x61: mag2, // ∐
	0x63: mag2, // ̂
	0x64: mag4, // ̂
	0x66: mag2, // ˜
	0x67: mag4, // ˜
	0x68: mag3, // [
	0x69: mag3, // ]
	0x6B: mag2, // ⌋
	0x6C: mag2, // ⌈
	0x6D: mag2, // ⌉
	0x6E: mag3, // {
	0x6F: mag3, // }

	0x71: mag3, // √
	0x72: mag4, // √
	0x73: mag5, // √
	0x74: mag1, // ⎷
	0x75: mag1, // ⏐
	0x76: mag1, // ⌜
}

var cmapCMR = map[uint32]rune{
	0x00: '\u0393',
	0x01: '\u0394',
	0x02: '\u0398',
	0x03: '\u039B',
	0x04: '\u039E',
	0x05: '\u03A0',
	0x06: '\u03A3',
	0x07: '\u03A5',
	0x08: '\u03A6',
	0x09: '\u03A8',
	0x0A: '\u03A9',
	0x0B: '\uFB00',
	0x0C: '\uFB01',
	0x0D: '\uFB02',
	0x0E: '\uFB03',
	0x0F: '\uFB04',
	0x10: '\u0131',
	0x11: '\u0237',
	0x12: '\u0300',
	0x13: '\u0301',
	0x14: '\u030C',
	0x15: '\u0306',
	0x16: '\u0305',
	0x17: '\u030A',
	0x18: '\u0327',
	0x19: '\u00DF',
	0x1A: '\u00E6',
	0x1B: '\u0153',
	0x1C: '\u00F8',
	0x1D: '\u00C6',
	0x1E: '\u0152',
	0x1F: '\u00D8',
	0x20: '\u0337',
	0x21: '\u0021',
	0x22: '\u201D',
	0x23: '\u0023',
	0x24: '\u0024',
	0x25: '\u0025',
	0x26: '\u0026',
	0x27: '\u0027',
	0x28: '\u0028',
	0x29: '\u0029',
	0x2A: '\u002A',
	0x2B: '\u002B',
	0x2C: '\u002C',
	0x2D: '\u002D',
	0x2E: '\u002E',
	0x2F: '\u002F',
	0x30: '\u0030',
	0x31: '\u0031',
	0x32: '\u0032',
	0x33: '\u0033',
	0x34: '\u0034',
	0x35: '\u0035',
	0x36: '\u0036',
	0x37: '\u0037',
	0x38: '\u0038',
	0x39: '\u0039',
	0x3A: '\u003A',
	0x3B: '\u003B',
	0x3C: '\u00A1',
	0x3D: '\u003D',
	0x3E: '\u00BF',
	0x3F: '\u003F',
	0x40: '\u0040',
	0x41: '\u0041',
	0x42: '\u0042',
	0x43: '\u0043',
	0x44: '\u0044',
	0x45: '\u0045',
	0x46: '\u0046',
	0x47: '\u0047',
	0x48: '\u0048',
	0x49: '\u0049',
	0x4A: '\u004A',
	0x4B: '\u004B',
	0x4C: '\u004C',
	0x4D: '\u004D',
	0x4E: '\u004E',
	0x4F: '\u004F',
	0x50: '\u0050',
	0x51: '\u0051',
	0x52: '\u0052',
	0x53: '\u0053',
	0x54: '\u0054',
	0x55: '\u0055',
	0x56: '\u0056',
	0x57: '\u0057',
	0x58: '\u0058',
	0x59: '\u0059',
	0x5A: '\u005A',
	0x5B: '\u005B',
	0x5C: '\u201C',
	0x5D: '\u005D',
	0x5E: '\u0302',
	0x5F: '\u0307',
	0x60: '\u2018',
	0x61: '\u0061',
	0x62: '\u0062',
	0x63: '\u0063',
	0x64: '\u0064',
	0x65: '\u0065',
	0x66: '\u0066',
	0x67: '\u0067',
	0x68: '\u0068',
	0x69: '\u0069',
	0x6A: '\u006A',
	0x6B: '\u006B',
	0x6C: '\u006C',
	0x6D: '\u006D',
	0x6E: '\u006E',
	0x6F: '\u006F',
	0x70: '\u0070',
	0x71: '\u0071',
	0x72: '\u0072',
	0x73: '\u0073',
	0x74: '\u0074',
	0x75: '\u0075',
	0x76: '\u0076',
	0x77: '\u0077',
	0x78: '\u0078',
	0x79: '\u0079',
	0x7A: '\u007A',
	0x7B: '\u2013',
	0x7C: '\u2014',
	0x7D: '\u030B',
	0x7E: '\u0303',
	0x7F: '\u0308',
}

var cmapCMMI = map[uint32]rune{
	0x00: '\u0393',
	0x01: '\u0394',
	0x02: '\u0398',
	0x03: '\u039B',
	0x04: '\u039E',
	0x05: '\u03A0',
	0x06: '\u03A3',
	0x07: '\u03A5',
	0x08: '\u03A6',
	0x09: '\u03A8',
	0x0A: '\u03A9',
	0x0B: '\u03B1',
	0x0C: '\u03B2',
	0x0D: '\u03B3',
	0x0E: '\u03B4',
	0x0F: '\u03B5',
	0x10: '\u03B6',
	0x11: '\u03B7',
	0x12: '\u03B8',
	0x13: '\u03B9',
	0x14: '\u03BA',
	0x15: '\u03BB',
	0x16: '\u03BC',
	0x17: '\u03BD',
	0x18: '\u03BE',
	0x19: '\u03C0',
	0x1A: '\u03C1',
	0x1B: '\u03C3',
	0x1C: '\u03C4',
	0x1D: '\u03C5',
	0x1E: '\u03C6',
	0x1F: '\u03C7',
	0x20: '\u03C8',
	0x21: '\u03C9',
	0x22: '\u025B',
	0x23: '\u03D1',
	0x24: '\u03D6',
	0x25: '\u03F1',
	0x26: '\u03C2',
	0x27: '\u03D5',
	0x28: '\u21BC',
	0x29: '\u21BD',
	0x2A: '\u21C0',
	0x2B: '\u21C1',
	0x2C: '\u21AA',
	0x2D: '\u21A9',
	0x2E: '\u25B9',
	0x2F: '\u25C3',
	0x30: '\u0030',
	0x31: '\u0031',
	0x32: '\u0032',
	0x33: '\u0033',
	0x34: '\u0034',
	0x35: '\u0035',
	0x36: '\u0036',
	0x37: '\u0037',
	0x38: '\u0038',
	0x39: '\u0039',
	0x3A: '\u002E',
	0x3B: '\u002C',
	0x3C: '\u003C',
	0x3D: '\u002F',
	0x3E: '\u003E',
	0x3F: '\u22C6',
	0x40: '\u2202',
	0x41: '\u0041',
	0x42: '\u0042',
	0x43: '\u0043',
	0x44: '\u0044',
	0x45: '\u0045',
	0x46: '\u0046',
	0x47: '\u0047',
	0x48: '\u0048',
	0x49: '\u0049',
	0x4A: '\u004A',
	0x4B: '\u004B',
	0x4C: '\u004C',
	0x4D: '\u004D',
	0x4E: '\u004E',
	0x4F: '\u004F',
	0x50: '\u0050',
	0x51: '\u0051',
	0x52: '\u0052',
	0x53: '\u0053',
	0x54: '\u0054',
	0x55: '\u0055',
	0x56: '\u0056',
	0x57: '\u0057',
	0x58: '\u0058',
	0x59: '\u0059',
	0x5A: '\u005A',
	0x5B: '\u266D',
	0x5C: '\u266E',
	0x5D: '\u266F',
	0x5E: '\u2323',
	0x5F: '\u2322',
	0x60: '\u2113',
	0x61: '\u0061',
	0x62: '\u0062',
	0x63: '\u0063',
	0x64: '\u0064',
	0x65: '\u0065',
	0x66: '\u0066',
	0x67: '\u0067',
	0x68: '\u0068',
	0x69: '\u0069',
	0x6A: '\u006A',
	0x6B: '\u006B',
	0x6C: '\u006C',
	0x6D: '\u006D',
	0x6E: '\u006E',
	0x6F: '\u006F',
	0x70: '\u0070',
	0x71: '\u0071',
	0x72: '\u0072',
	0x73: '\u0073',
	0x74: '\u0074',
	0x75: '\u0075',
	0x76: '\u0076',
	0x77: '\u0077',
	0x78: '\u0078',
	0x79: '\u0079',
	0x7A: '\u007A',
	0x7B: '\u0131',
	0x7C: '\u0237',
	0x7D: '\u2118',
	0x7E: '\u20D7',
	0x7F: '\u0311',
}

var cmapCMSY = map[uint32]rune{
	0x00: '\u2212',
	0x01: '\u00B7',
	0x02: '\u00D7',
	0x03: '\u22C6',
	0x04: '\u00F7',
	0x05: '\u22C4',
	0x06: '\u00B1',
	0x07: '\u2213',
	0x08: '\u2295',
	0x09: '\u2296',
	0x0A: '\u2297',
	0x0B: '\u2298',
	0x0C: '\u2299',
	0x0D: '\u25CB',
	0x0E: '\u2218',
	0x0F: '\u2219',
	0x10: '\u2243',
	0x11: '\u224D',
	0x12: '\u2286',
	0x13: '\u2287',
	0x14: '\u2264',
	0x15: '\u2265',
	0x16: '\u227C',
	0x17: '\u227D',
	0x18: '\u223C',
	0x19: '\u2245',
	0x1A: '\u2282',
	0x1B: '\u2283',
	0x1C: '\u226A',
	0x1D: '\u226B',
	0x1E: '\u227A',
	0x1F: '\u227B',
	0x20: '\u2190',
	0x21: '\u2192',
	0x22: '\u2191',
	0x23: '\u2193',
	0x24: '\u2194',
	0x25: '\u2197',
	0x26: '\u2198',
	0x27: '\u2242',
	0x28: '\u21D0',
	0x29: '\u21D2',
	0x2A: '\u21D1',
	0x2B: '\u21D3',
	0x2C: '\u21D4',
	0x2D: '\u2196',
	0x2E: '\u2199',
	0x2F: '\u221D',
	0x30: '\u2032',
	0x31: '\u221E',
	0x32: '\u2208',
	0x33: '\u220B',
	0x34: '\u25B3',
	0x35: '\u25BD',
	0x36: '\u0338',
	0x37: '\u21A6',
	0x38: '\u2200',
	0x39: '\u2203',
	0x3A: '\u00AC',
	0x3B: '\u2205',
	0x3C: '\u211C',
	0x3D: '\u2111',
	0x3E: '\u22A4',
	0x3F: '\u22A5',
	0x40: '\u2135',
	0x41: '\U0001D49C',
	0x42: '\u212C',
	0x43: '\U0001D49E',
	0x44: '\U0001D49F',
	0x45: '\u2130',
	0x46: '\u2131',
	0x47: '\U0001D4A2',
	0x48: '\u210B',
	0x49: '\u2110',
	0x4A: '\U0001D4A5',
	0x4B: '\U0001D4A6',
	0x4C: '\u2112',
	0x4D: '\u2133',
	0x4E: '\U0001D4A9',
	0x4F: '\U0001D4AA',
	0x50: '\U0001D4AB',
	0x51: '\U0001D4AC',
	0x52: '\u211B',
	0x53: '\U0001D4AE',
	0x54: '\U0001D4AF',
	0x55: '\U0001D4B0',
	0x56: '\U0001D4B1',
	0x57: '\U0001D4B2',
	0x58: '\U0001D4B3',
	0x59: '\U0001D4B4',
	0x5A: '\U0001D4B5',
	0x5B: '\u222A',
	0x5C: '\u2229',
	0x5D: '\u228E',
	0x5E: '\u2227',
	0x5F: '\u2228',
	0x60: '\u22A2',
	0x61: '\u22A3',
	0x62: '\u230A',
	0x63: '\u230B',
	0x64: '\u2308',
	0x65: '\u2309',
	0x66: '\u007B',
	0x67: '\u007D',
	0x68: '\u2329',
	0x69: '\u232A',
	0x6A: '\u2223',
	0x6B: '\u2225',
	0x6C: '\u2195',
	0x6D: '\u21D5',
	0x6E: '\u2216',
	0x6F: '\u2240',
	0x70: '\u221A',
	0x71: '\u2210',
	0x72: '\u2207',
	0x73: '\u222B',
	0x74: '\u2294',
	0x75: '\u2293',
	0x76: '\u2291',
	0x77: '\u2292',
	0x78: '\u00A7',
	0x79: '\u2020',
	0x7A: '\u2021',
	0x7B: '\u00B6',
	0x7C: '\u2663',
	0x7D: '\u2662',
	0x7E: '\u2661',
	0x7F: '\u2660',
}

// See Figure 6, Appendix F, The TeXbook, D. E. Knuth, 1984, 1986, 1991
var cmapCMEX = map[uint32]rune{
	0x00: '\u0028', // (
	0x01: '\u0029', // )
	0x02: '\u005B', // [
	0x03: '\u005D', // ]
	0x04: '\u230A', // ⌊
	0x05: '\u230B', // ⌋
	0x06: '\u2308', // ⌈
	0x07: '\u2309', // ⌉
	0x08: '\u007B', // {
	0x09: '\u007D', // }
	0x0A: '\u2329', // 〈
	0x0B: '\u232A', // 〉
	0x0C: '\u2223', // ∣
	0x0D: '\u2225', // ∥
	0x0E: '\u2215', // ∕
	0x0F: '\u2216', // \

	0x10: '\u0028', // ( mag3
	0x11: '\u0029', // ) mag3
	0x12: '\u0028', // ( mag4
	0x13: '\u0029', // ) mag4
	0x14: '\u005B', // [ mag4
	0x15: '\u005D', // ] mag4
	0x16: '\u230A', // ⌊ mag4
	0x17: '\u230B', // ⌋ mag4
	0x18: '\u2308', // ⌈ mag4
	0x19: '\u2309', // ⌉ mag4
	0x1A: '\u007B', // { mag4
	0x1B: '\u007D', // } mag4
	0x1C: '\u2329', // 〈 mag4
	0x1D: '\u232A', // 〉 mag4
	0x1E: '\u2215', // ∕ mag4
	0x1F: '\u2216', // \ mag4

	0x20: '\u0028', // ( mag5
	0x21: '\u0029', // ) mag5
	0x22: '\u005B', // [ mag5
	0x23: '\u005D', // ] mag5
	0x24: '\u230A', // ⌊ mag5
	0x25: '\u230B', // ⌋ mag5
	0x26: '\u2308', // ⌈ mag5
	0x27: '\u2309', // ⌉ mag5
	0x28: '\u007B', // { mag5
	0x29: '\u007D', // } mag5
	0x2A: '\u2329', // 〈 mag5
	0x2B: '\u232A', // 〉 mag5
	0x2C: '\u2215', // ∕ mag5
	0x2D: '\u2216', // \ mag5
	0x2E: '\u2215', // ∕ mag3
	0x2F: '\u2216', // \ mag3

	0x30: '\u239B', // ⎛ mag2
	0x31: '\u239E', // ⎞ mag2
	0x32: '\u23A1', // ⎡ mag2 note: thicker series here
	0x33: '\u23A4', // ⎤ "
	0x34: '\u23A3', // ⎣ "
	0x35: '\u23A6', // ⎦ "
	0x36: '\u23A2', // ⎢ "
	0x37: '\u23A5', // ⎥ "

	0x38: '\u23A7', // ⎧ braces mag2
	0x39: '\u23AB', // ⎫
	0x3A: '\u23A9', // ⎩
	0x3B: '\u23AD', // ⎭
	0x3C: '\u23A8', // ⎨
	0x3D: '\u23AC', // ⎬
	0x3E: '\u23AA', // ⎪
	0x3F: '\u2223', // ∣ ?? unclear

	0x40: '\u239D', // ⎝ mag2
	0x41: '\u23A0', // ⎠ mag2
	0x42: '\u239C', // ⎜ mag2
	0x43: '\u239F', // ⎟ mag2
	0x44: '\u2329', // 〈 mag2
	0x45: '\u232A', // 〉 mag2
	0x46: '\u2A06', // ⨆
	0x47: '\u2A06', // ⨆ mag2
	0x48: '\u222E', // ∮
	0x49: '\u222E', // ∮ mag2
	0x4A: '\u2A00', // ⨀
	0x4B: '\u2A00', // ⨀ mag2
	0x4C: '\u2A01', // ⨁
	0x4D: '\u2A01', // ⨁ mag2
	0x4E: '\u2A02', // ⨂
	0x4F: '\u2A02', // ⨂ mag2

	0x50: '\u2211', // ∑
	0x51: '\u220F', // ∏
	0x52: '\u222B', // ∫
	0x53: '\u22C3', // ⋃
	0x54: '\u22C2', // ⋂
	0x55: '\u2A05', // ⨄
	0x56: '\u22C0', // ⋀
	0x57: '\u22C1', // ⋁
	0x58: '\u2211', // ∑ mag2
	0x59: '\u220F', // ∏ mag2
	0x5A: '\u222B', // ∫ mag2
	0x5B: '\u22C3', // ⋃ mag2
	0x5C: '\u22C2', // ⋂ mag2
	0x5D: '\u2A03', // ⨄ mag2
	0x5E: '\u22C0', // ⋀ mag2
	0x5F: '\u22C1', // ⋁ mag2

	0x60: '\u2210', // ∐
	0x61: '\u2210', // ∐ mag2
	0x62: '\u0302', // ̂
	0x63: '\u0302', // ̂ mag2
	0x64: '\u0302', // ̂ mag3?
	0x65: '\u0303', // ˜
	0x66: '\u0303', // ˜ mag2
	0x67: '\u0303', // ˜ mag3?
	0x68: '\u005B', // [ mag3
	0x69: '\u005D', // ] mag3
	0x6A: '\u230A', // ⌊ mag2
	0x6B: '\u230B', // ⌋ mag2
	0x6C: '\u2308', // ⌈ mag2
	0x6D: '\u2309', // ⌉ mag2
	0x6E: '\u007B', // { mag2
	0x6F: '\u007D', // } mag2

	0x70: '\u221A', // √
	0x71: '\u221A', // √ mag3
	0x72: '\u221A', // √ mag4
	0x73: '\u221A', // √ mag5
	0x74: '\u23B7', // ⎷ mag1
	0x75: '\u23D0', // ⏐ mag1
	0x76: '\u250C', // ┌ mag1
	0x77: '\u2225', // ∥
	0x78: '\u2191', // ↑
	0x79: '\u2193', // ↓
	0x7A: '\u23DC', // ⏜ only left half
	0x7B: '\u23DC', // ⏜ only right half
	0x7C: '\u23DD', // ⏝ only left
	0x7D: '\u23DD', // ⏝ only right
	0x7E: '\u21D1', // ⇑
	0x7F: '\u21D3', // ⇓
}

var cmapCMTT = map[uint32]rune{
	0x00: '\u0393',
	0x01: '\u0394',
	0x02: '\u0398',
	0x03: '\u039B',
	0x04: '\u039E',
	0x05: '\u03A0',
	0x06: '\u03A3',
	0x07: '\u03A5',
	0x08: '\u03A6',
	0x09: '\u03A8',
	0x0A: '\u03A9',
	0x0B: '\u2191',
	0x0C: '\u2193',
	0x0D: '\u0027',
	0x0E: '\u00A1',
	0x0F: '\u00BF',
	0x10: '\u0131',
	0x11: '\u0237',
	0x12: '\u0300',
	0x13: '\u0301',
	0x14: '\u030C',
	0x15: '\u0306',
	0x16: '\u0305',
	0x17: '\u030A',
	0x18: '\u0327',
	0x19: '\u00DF',
	0x1A: '\u00E6',
	0x1B: '\u0153',
	0x1C: '\u00F8',
	0x1D: '\u00C6',
	0x1E: '\u0152',
	0x1F: '\u00D8',
	0x20: '\u0337',
	0x21: '\u0021',
	0x22: '\u201D',
	0x23: '\u0023',
	0x24: '\u0024',
	0x25: '\u0025',
	0x26: '\u0026',
	0x27: '\u0027',
	0x28: '\u0028',
	0x29: '\u0029',
	0x2A: '\u002A',
	0x2B: '\u002B',
	0x2C: '\u002C',
	0x2D: '\u002D',
	0x2E: '\u002E',
	0x2F: '\u002F',
	0x30: '\u0030',
	0x31: '\u0031',
	0x32: '\u0032',
	0x33: '\u0033',
	0x34: '\u0034',
	0x35: '\u0035',
	0x36: '\u0036',
	0x37: '\u0037',
	0x38: '\u0038',
	0x39: '\u0039',
	0x3A: '\u003A',
	0x3B: '\u003B',
	0x3C: '\u003C',
	0x3D: '\u003D',
	0x3E: '\u003E',
	0x3F: '\u003F',
	0x40: '\u0040',
	0x41: '\u0041',
	0x42: '\u0042',
	0x43: '\u0043',
	0x44: '\u0044',
	0x45: '\u0045',
	0x46: '\u0046',
	0x47: '\u0047',
	0x48: '\u0048',
	0x49: '\u0049',
	0x4A: '\u004A',
	0x4B: '\u004B',
	0x4C: '\u004C',
	0x4D: '\u004D',
	0x4E: '\u004E',
	0x4F: '\u004F',
	0x50: '\u0050',
	0x51: '\u0051',
	0x52: '\u0052',
	0x53: '\u0053',
	0x54: '\u0054',
	0x55: '\u0055',
	0x56: '\u0056',
	0x57: '\u0057',
	0x58: '\u0058',
	0x59: '\u0059',
	0x5A: '\u005A',
	0x5B: '\u005B',
	0x5C: '\u005C',
	0x5D: '\u005D',
	0x5E: '\u0302',
	0x5F: '\u005F',
	0x60: '\u2018',
	0x61: '\u0061',
	0x62: '\u0062',
	0x63: '\u0063',
	0x64: '\u0064',
	0x65: '\u0065',
	0x66: '\u0066',
	0x67: '\u0067',
	0x68: '\u0068',
	0x69: '\u0069',
	0x6A: '\u006A',
	0x6B: '\u006B',
	0x6C: '\u006C',
	0x6D: '\u006D',
	0x6E: '\u006E',
	0x6F: '\u006F',
	0x70: '\u0070',
	0x71: '\u0071',
	0x72: '\u0072',
	0x73: '\u0073',
	0x74: '\u0074',
	0x75: '\u0075',
	0x76: '\u0076',
	0x77: '\u0077',
	0x78: '\u0078',
	0x79: '\u0079',
	0x7A: '\u007A',
	0x7B: '\u007B',
	0x7C: '\u007C',
	0x7D: '\u007D',
	0x7E: '\u0303',
	0x7F: '\u0308',
}
