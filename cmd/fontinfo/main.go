package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"

	"github.com/tdewolff/argp"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/font"
	"golang.org/x/image/draw"
	"golang.org/x/image/tiff"
	"golang.org/x/image/vector"
)

type ShowOptions struct {
	Index   int     `short:"i" desc:"Font index for font collections"`
	GlyphID uint16  `short:"g" desc:"Glyph ID"`
	Char    string  `short:"c" desc:"Unicode character"`
	Width   int     `desc:"Image width"`
	PPEM    uint16  `default:"40" desc:"Pixels per em-square"`
	Scale   int     `default:"4" desc:"Image scale"`
	Ratio   float64 `desc:"Image width/height ratio"`
	Output  string  `short:"o" desc:"Output filename"`
}

type InfoOptions struct {
	Index   int    `short:"i" desc:"Font index for font collections"`
	Table   string `short:"t" desc:"OpenType table name"`
	GlyphID uint16 `short:"g" desc:"Glyph ID"`
	Char    string `short:"c" desc:"Unicode character"`
	Output  string `short:"o" desc:"Output filename"`
}

var (
	showOptions ShowOptions
	infoOptions InfoOptions
)

func main() {
	root := argp.New("Toolkit for TTF and OTF files")
	show := root.AddCommand(show, "show", "Show glyphs in terminal or output to image")
	show.AddStruct(&showOptions)

	info := root.AddCommand(info, "info", "Get font info")
	info.AddStruct(&infoOptions)

	root.Parse()
	root.PrintHelp()
}

func show(args []string) error {
	terminal := showOptions.Output == "" || showOptions.Output == "-"
	if len(args) != 1 {
		return fmt.Errorf("must pass one font file")
	}

	b, err := ioutil.ReadFile(args[0])
	if err != nil {
		return err
	}

	sfnt, err := font.ParseSFNT(b, showOptions.Index)
	if err != nil {
		return err
	}

	if showOptions.Char != "" {
		rs := []rune(showOptions.Char)
		if len(rs) != 1 {
			return fmt.Errorf("char must be one Unicode character")
		}
		showOptions.GlyphID = sfnt.GlyphIndex(rs[0])
	}

	if showOptions.Width != 0 {
		if showOptions.Width < 0 {
			return fmt.Errorf("width must be positive")
		}
		showOptions.PPEM = uint16(float64(showOptions.Width) * float64(sfnt.Head.UnitsPerEm) / float64(sfnt.GlyphAdvance(showOptions.GlyphID)))
	}

	ascent := sfnt.Hhea.Ascender
	descent := -sfnt.Hhea.Descender
	width := int(float64(showOptions.PPEM)*float64(sfnt.GlyphAdvance(showOptions.GlyphID))/float64(sfnt.Head.UnitsPerEm) + 0.5)
	height := int(float64(showOptions.PPEM)*float64(ascent+descent)/float64(sfnt.Head.UnitsPerEm) + 0.5)
	//baseline := int(float64(ppem)*float64(ascent)/float64(sfnt.Head.UnitsPerEm) + 0.5)
	xpadding := int(float64(width)*0.2 + 0.5)
	ypadding := xpadding
	if terminal {
		ypadding = 0
	}

	if 2048 < width {
		return fmt.Errorf("width cannot exceed 2048")
	}

	f := float64(showOptions.PPEM) / float64(sfnt.Head.UnitsPerEm)

	p := &canvas.Path{}
	err = sfnt.GlyphPath(p, showOptions.GlyphID, showOptions.PPEM, 0, int32(descent), 1.0, font.NoHinting)
	if err != nil {
		return err
	}

	rect := image.Rect(0, 0, width+2*xpadding, height+2*ypadding)
	glyphRect := image.Rect(xpadding, ypadding, width+xpadding, height+ypadding)

	img := image.NewRGBA(rect)
	draw.Draw(img, rect, image.NewUniform(canvas.White), image.ZP, draw.Over)

	ras := vector.NewRasterizer(width, height)
	p.ToRasterizer(ras, canvas.DPMM(f))
	ras.Draw(img, glyphRect, image.NewUniform(canvas.Black), image.ZP)

	if showOptions.Ratio == 0.0 {
		if terminal {
			showOptions.Ratio = 2.0
		} else {
			showOptions.Ratio = 1.0
		}
	}

	if showOptions.Ratio != 1.0 {
		origImg := img
		origRect := rect
		rect := image.Rect(0, 0, int(float64(origRect.Max.X)*showOptions.Ratio+0.5), origRect.Max.Y)
		img = image.NewRGBA(rect)
		draw.ApproxBiLinear.Scale(img, rect, origImg, origRect, draw.Over, nil)
	}

	if terminal {
		if 80 < width {
			return fmt.Errorf("width cannot exceed 80 for terminal output")
		}
		printASCII(img)
		return nil
	}

	if showOptions.Scale != 1 {
		origImg := img
		origRect := rect
		rect := image.Rect(0, 0, (origRect.Max.X)*showOptions.Scale, (origRect.Max.Y)*showOptions.Scale)
		img = image.NewRGBA(rect)
		draw.NearestNeighbor.Scale(img, rect, origImg, origRect, draw.Over, nil)
	}

	ext := filepath.Ext(showOptions.Output)
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" && ext != ".tiff" {
		return fmt.Errorf("output extension must be PNG, JPG, GIF, or TIFF")
	}

	w, err := os.Create(showOptions.Output)
	if err != nil {
		return err
	}

	switch ext {
	case ".png":
		err = png.Encode(w, img)
	case ".jpg", ".jpeg":
		err = jpeg.Encode(w, img, nil)
	case ".gif":
		err = gif.Encode(w, img, nil)
	case ".tiff":
		err = tiff.Encode(w, img, nil)
	}

	if err != nil {
		return err
	}
	return nil
}

func printASCII(img image.Image) {
	palette := []byte("$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjft/\\|()1{}[]?-_+~<>i!lI;:,\"^`'. ")

	size := img.Bounds().Max
	for j := 0; j < size.Y; j++ {
		for i := 0; i < size.X; i++ {
			r, g, b, _ := img.At(i, j).RGBA()
			y, _, _ := color.RGBToYCbCr(uint8(r>>8), uint8(g>>8), uint8(b>>8))
			idx := int(float64(y)/255.0*float64(len(palette)-1) + 0.5)
			fmt.Print(string(palette[idx]))
		}
		fmt.Print("\n")
	}
}

func info(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("must pass one font file")
	}

	b, err := ioutil.ReadFile(args[0])
	if err != nil {
		return err
	}

	sfnt, err := font.ParseSFNT(b, infoOptions.Index)
	if err != nil {
		return err
	}

	fmt.Printf("File: %s\n\n", args[0])
	version := "TrueType"
	if sfnt.Version == "OTTO" {
		version = "CFF"
	} else if sfnt.Version == "ttcf" {
		version = "Collection"
	}
	fmt.Printf("sfntVersion: 0x%08X (%s)\n", sfnt.Version, version)

	nLen := int(math.Log10(float64(len(sfnt.Data))) + 1)

	fmt.Printf("\nTable directory:\n")
	r := font.NewBinaryReader(sfnt.Data)
	_ = r.ReadBytes(4)
	numTables := int(r.ReadUint16())
	_ = r.ReadBytes(6)
	for i := 0; i < numTables; i++ {
		tag := r.ReadString(4)
		checksum := r.ReadUint32()
		offset := r.ReadUint32()
		length := r.ReadUint32()
		fmt.Printf("  %2d  %s  checksum=0x%08X  offset=%*d  length=%*d\n", i, tag, checksum, nLen, offset, nLen, length)
	}
	return nil
}
