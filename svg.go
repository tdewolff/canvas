package canvas

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
)

type svgWriter struct {
	io.Writer
	height float64
	fonts  map[*Font]bool
	maskID int
}

func newSVGWriter(writer io.Writer, w, h float64) *svgWriter {
	fmt.Fprintf(writer, `<svg version="1.1" width="%v" height="%v" viewBox="0 0 %v %v" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`, dec(w), dec(h), dec(w), dec(h))
	return &svgWriter{writer, h, map[*Font]bool{}, 0}
}

func (w *svgWriter) Close() error {
	fmt.Fprintf(w, "</svg>")
	return nil
}

func (w *svgWriter) EmbedFonts(fonts []*Font) {
	is := []int{}
	for i, font := range fonts {
		if _, ok := w.fonts[font]; !ok {
			is = append(is, i)
			w.fonts[font] = true
		}
	}

	if 0 < len(is) {
		fmt.Fprintf(w, "<style>")
		for _, i := range is {
			mimetype, raw := fonts[i].Raw()
			fmt.Fprintf(w, "\n@font-face{font-family:'%s';src:url('data:%s;base64,", fonts[i].name, mimetype)
			encoder := base64.NewEncoder(base64.StdEncoding, w)
			encoder.Write(raw)
			encoder.Close()
			fmt.Fprintf(w, "');}")
		}
		fmt.Fprintf(w, "\n</style>")
	}
}

func (w *svgWriter) DrawImage(img image.Image, enc ImageEncoding, m Matrix) {
	refMask := ""
	mimetype := "image/png"
	if enc == Lossy {
		mimetype = "image/jpg"
		if opaqueImg, ok := img.(interface{ Opaque() bool }); !ok || !opaqueImg.Opaque() {
			hasMask := false
			size := img.Bounds().Size()
			opaque := image.NewRGBA(img.Bounds())
			mask := image.NewGray(img.Bounds())
			for y := 0; y < size.Y; y++ {
				for x := 0; x < size.X; x++ {
					R, G, B, A := img.At(x, y).RGBA()
					if A != 0 {
						r := byte((R * 65535 / A) >> 8)
						g := byte((G * 65535 / A) >> 8)
						b := byte((B * 65535 / A) >> 8)
						opaque.SetRGBA(x, y, color.RGBA{r, g, b, 255})
						mask.SetGray(x, y, color.Gray{byte(A >> 8)})
					}
					if A>>8 != 255 {
						hasMask = true
					}
				}
			}
			if hasMask {
				img = opaque
				refMask = fmt.Sprintf("m%v", w.maskID)
				w.maskID++

				fmt.Fprintf(w, `<mask id="%s"><image width="%d" height="%d" xlink:href="data:image/jpg;base64,`, refMask, size.X, size.Y)
				encoder := base64.NewEncoder(base64.StdEncoding, w)
				if err := jpeg.Encode(encoder, mask, nil); err != nil {
					panic(err)
				}
				if err := encoder.Close(); err != nil {
					panic(err)
				}
				fmt.Fprintf(w, `"/></mask>`)
			}
		}
	}

	m = m.Translate(0.0, float64(img.Bounds().Size().Y))
	fmt.Fprintf(w, `<image transform="%s" width="%d" height="%d" xlink:href="data:%s;base64,`,
		m.ToSVG(w.height), img.Bounds().Size().X, img.Bounds().Size().Y, mimetype)

	encoder := base64.NewEncoder(base64.StdEncoding, w)
	if mimetype == "image/jpg" {
		if err := jpeg.Encode(encoder, img, nil); err != nil {
			panic(err)
		}
	} else {
		if err := png.Encode(encoder, img); err != nil {
			panic(err)
		}
	}
	if err := encoder.Close(); err != nil {
		panic(err)
	}

	if refMask != "" {
		fmt.Fprintf(w, `" mask="url(#%s)`, refMask)
	}
	fmt.Fprintf(w, `"/>`)
}
