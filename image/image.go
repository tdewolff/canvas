package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"

	ccolor "github.com/tdewolff/canvas/color"
)

// ImageEncoding defines whether the embedded image shall be embedded as lossless (typically PNG) or lossy (typically JPG).
type ImageEncoding int

// see ImageEncoding
const (
	Lossless ImageEncoding = iota
	Lossy
)

// Image is a raster image that is loaded lazily. Keeping the original bytes allows the renderer to optimize rendering in some cases.
type Image struct {
	Bytes    []byte
	Mimetype string
	image.Config
	Mask *Image

	decode func(io.Reader) (image.Image, error)
	image  image.Image
}

func (i *Image) Image() (image.Image, error) {
	if i.image != nil {
		return i.image, nil
	}
	var err error
	i.image, err = i.decode(bytes.NewReader(i.Bytes))
	if err != nil {
		i.image = image.NewUniform(color.Black)
	}
	if i.Mask != nil {
		src := i.image
		mask, _ := i.Mask.Image()
		bounds := src.Bounds()
		dst := image.NewNRGBA(bounds)
		if alpha, ok := mask.(interface{ AlphaAt(int, int) color.Alpha }); ok {
			for j := bounds.Min.Y; j < bounds.Max.Y; j++ {
				for i := bounds.Min.X; i < bounds.Max.X; i++ {
					r, g, b, a := src.At(i, j).RGBA()
					m := uint32(alpha.AlphaAt(i, j).A)
					m |= m << 8
					r = (r * m) / 0xffff
					g = (g * m) / 0xffff
					b = (b * m) / 0xffff
					a = (a * m) / 0xffff
					dst.SetNRGBA(i, j, color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
				}
			}
		} else if alpha, ok := mask.(interface{ Alpha16At(int, int) color.Alpha16 }); ok {
			for j := bounds.Min.Y; j < bounds.Max.Y; j++ {
				for i := bounds.Min.X; i < bounds.Max.X; i++ {
					r, g, b, a := src.At(i, j).RGBA()
					m := uint32(alpha.Alpha16At(i, j).A)
					r = (r * m) / 0xffff
					g = (g * m) / 0xffff
					b = (b * m) / 0xffff
					a = (a * m) / 0xffff
					dst.SetNRGBA(i, j, color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
				}
			}
		} else if gray, ok := mask.(interface{ GrayAt(int, int) color.Gray }); ok {
			for j := bounds.Min.Y; j < bounds.Max.Y; j++ {
				for i := bounds.Min.X; i < bounds.Max.X; i++ {
					r, g, b, a := src.At(i, j).RGBA()
					m := uint32(gray.GrayAt(i, j).Y)
					m |= m << 8
					r = (r * m) / 0xffff
					g = (g * m) / 0xffff
					b = (b * m) / 0xffff
					a = (a * m) / 0xffff
					dst.SetNRGBA(i, j, color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
				}
			}
		} else if gray, ok := mask.(interface{ Gray16At(int, int) color.Gray16 }); ok {
			for j := bounds.Min.Y; j < bounds.Max.Y; j++ {
				for i := bounds.Min.X; i < bounds.Max.X; i++ {
					r, g, b, a := src.At(i, j).RGBA()
					m := uint32(gray.Gray16At(i, j).Y)
					r = (r * m) / 0xffff
					g = (g * m) / 0xffff
					b = (b * m) / 0xffff
					a = (a * m) / 0xffff
					dst.SetNRGBA(i, j, color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
				}
			}
		} else {
			for j := bounds.Min.Y; j < bounds.Max.Y; j++ {
				for i := bounds.Min.X; i < bounds.Max.X; i++ {
					r, g, b, a := src.At(i, j).RGBA()
					mr, mg, mb, _ := mask.At(i, j).RGBA()
					m := (19595*mr + 38470*mg + 7471*mb + 1<<15) >> 16
					r = (r * m) / 0xffff
					g = (g * m) / 0xffff
					b = (b * m) / 0xffff
					a = (a * m) / 0xffff
					dst.SetNRGBA(i, j, color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
				}
			}
		}
		i.image = dst
	}
	return i.image, err
}

func (i *Image) ColorModel() color.Model {
	return i.Config.ColorModel
}

func (i *Image) Bounds() image.Rectangle {
	return image.Rectangle{
		Min: image.ZP,
		Max: image.Point{i.Width, i.Height},
	}
}

func (i *Image) At(x, y int) color.Color {
	img, _ := i.Image()
	return img.At(x, y)
}

func (i *Image) Opaque() bool {
	if i.Mask != nil {
		return false
	}
	img, _ := i.Image()
	if opaqueImg, ok := img.(interface{ Opaque() bool }); ok && opaqueImg.Opaque() {
		return true
	} else if i.Width == 0 || i.Height == 0 || ccolor.OpaqueModel(i.Config.ColorModel) {
		return true
	}
	return false
}

// NewJPEGImage parses a JPEG image.
func NewJPEGImage(r io.Reader) (*Image, error) {
	return NewJPEGMaskedImage(r, nil)
}

// NewJPEGMaskedImage parses a JPEG image and a separate mask. This allows for a brightness-only JPEG to act as mask.
func NewJPEGMaskedImage(r io.Reader, mask *Image) (*Image, error) {
	if b, err := io.ReadAll(r); err != nil {
		return nil, err
	} else if config, err := jpeg.DecodeConfig(bytes.NewReader(b)); err != nil {
		return nil, err
	} else if mask != nil && (mask.Width != config.Width || mask.Height != config.Height) {
		return nil, fmt.Errorf("image and mask have different dimensions: %vx%v != %v %v", mask.Width, mask.Height, config.Width, config.Height)
	} else {
		return &Image{
			Bytes:    b,
			Mimetype: "image/jpeg",
			Config:   config,
			Mask:     mask,

			decode: jpeg.Decode,
		}, nil
	}
}

// NewPNGImage parses a PNG image
func NewPNGImage(r io.Reader) (*Image, error) {
	if b, err := io.ReadAll(r); err != nil {
		return nil, err
	} else if config, err := png.DecodeConfig(bytes.NewReader(b)); err != nil {
		return nil, err
	} else {
		return &Image{
			Bytes:    b,
			Mimetype: "image/png",
			Config:   config,

			decode: png.Decode,
		}, nil
	}
}

// RGB is an in-memory image whose At method returns [color.RGBA] values.
type RGB struct {
	// Pix holds the image's pixels, in R, G, B order. The pixel at
	// (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*3].
	Pix []uint8
	// Stride is the Pix stride (in bytes) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
}

func (p *RGB) ColorModel() color.Model { return ccolor.RGBModel }

func (p *RGB) Bounds() image.Rectangle { return p.Rect }

func (p *RGB) At(x, y int) color.Color {
	return p.RGBAt(x, y)
}

func (p *RGB) RGBAt(x, y int) ccolor.RGB {
	if !(image.Point{x, y}.In(p.Rect)) {
		return ccolor.RGB{0, 0, 0}
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857
	return ccolor.RGB{s[0], s[1], s[2]}
}

func (p *RGB) RGBA64At(x, y int) color.RGBA64 {
	if !(image.Point{x, y}.In(p.Rect)) {
		return color.RGBA64{0, 0, 0, 0xffff}
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857
	r, g, b := uint16(s[0]), uint16(s[1]), uint16(s[2])
	return color.RGBA64{r<<8 | r, g<<8 | g, b<<8 | b, 0xffff}
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *RGB) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*3
}

func (p *RGB) Set(x, y int, c color.Color) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	c1 := ccolor.RGBModel.Convert(c).(ccolor.RGB)
	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857
	s[0], s[1], s[2] = c1.R, c1.G, c1.B
}

func (p *RGB) SetRGB(x, y int, c ccolor.RGB) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857
	s[0], s[1], s[2] = c.R, c.G, c.B
}

// SubImage returns an image representing the portion of the image p visible
// through r. The returned value shares pixels with the original image.
func (p *RGB) SubImage(r image.Rectangle) image.Image {
	r = r.Intersect(p.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &RGB{}
	}
	i := p.PixOffset(r.Min.X, r.Min.Y)
	return &RGB{
		Pix:    p.Pix[i:],
		Stride: p.Stride,
		Rect:   r,
	}
}

// Opaque scans the entire image and reports whether it is fully opaque.
func (p *RGB) Opaque() bool {
	return true
}

// NewRGBA returns a new [RGB] image with the given bounds.
func NewRGB(r image.Rectangle) *RGB {
	return &RGB{
		Pix:    make([]uint8, 3*r.Dx()*r.Dy()),
		Stride: 3 * r.Dx(),
		Rect:   r,
	}
}

// RGB48 is an in-memory image whose At method returns [color.RGBA64] values.
type RGB48 struct {
	// Pix holds the image's pixels, in R, G, B order and big-endian format. The pixel at
	// (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*6].
	Pix []uint8
	// Stride is the Pix stride (in bytes) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
}

func (p *RGB48) ColorModel() color.Model { return ccolor.RGB48Model }

func (p *RGB48) Bounds() image.Rectangle { return p.Rect }

func (p *RGB48) At(x, y int) color.Color {
	return p.RGB48At(x, y)
}

func (p *RGB48) RGB48At(x, y int) ccolor.RGB48 {
	if !(image.Point{x, y}.In(p.Rect)) {
		return ccolor.RGB48{0, 0, 0}
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+6 : i+6] // Small cap improves performance, see https://golang.org/issue/27857
	return ccolor.RGB48{
		uint16(s[0])<<8 | uint16(s[1]),
		uint16(s[2])<<8 | uint16(s[3]),
		uint16(s[4])<<8 | uint16(s[5]),
	}
}

func (p *RGB48) RGBA64At(x, y int) color.RGBA64 {
	if !(image.Point{x, y}.In(p.Rect)) {
		return color.RGBA64{0, 0, 0, 0xffff}
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+6 : i+6] // Small cap improves performance, see https://golang.org/issue/27857
	return color.RGBA64{
		uint16(s[0])<<8 | uint16(s[1]),
		uint16(s[2])<<8 | uint16(s[3]),
		uint16(s[4])<<8 | uint16(s[5]),
		0xffff,
	}
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *RGB48) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*6
}

func (p *RGB48) Set(x, y int, c color.Color) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	c1 := ccolor.RGB48Model.Convert(c).(ccolor.RGB48)
	s := p.Pix[i : i+6 : i+6] // Small cap improves performance, see https://golang.org/issue/27857
	s[0] = uint8(c1.R >> 8)
	s[1] = uint8(c1.R)
	s[2] = uint8(c1.G >> 8)
	s[3] = uint8(c1.G)
	s[4] = uint8(c1.B >> 8)
	s[5] = uint8(c1.B)
}

func (p *RGB48) SetRGB48(x, y int, c ccolor.RGB48) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+6 : i+6] // Small cap improves performance, see https://golang.org/issue/27857
	s[0] = uint8(c.R >> 8)
	s[1] = uint8(c.R)
	s[2] = uint8(c.G >> 8)
	s[3] = uint8(c.G)
	s[4] = uint8(c.B >> 8)
	s[5] = uint8(c.B)
}

// SubImage returns an image representing the portion of the image p visible
// through r. The returned value shares pixels with the original image.
func (p *RGB48) SubImage(r image.Rectangle) image.Image {
	r = r.Intersect(p.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &RGB48{}
	}
	i := p.PixOffset(r.Min.X, r.Min.Y)
	return &RGB48{
		Pix:    p.Pix[i:],
		Stride: p.Stride,
		Rect:   r,
	}
}

// Opaque scans the entire image and reports whether it is fully opaque.
func (p *RGB48) Opaque() bool {
	return true
}

// NewRGB48 returns a new [RGB48] image with the given bounds.
func NewRGB48(r image.Rectangle) *RGB48 {
	return &RGB48{
		Pix:    make([]uint8, 6*r.Dx()*r.Dy()),
		Stride: 6 * r.Dx(),
		Rect:   r,
	}
}

// GrayA is an in-memory image whose At method returns [color.GrayA] values.
type GrayA struct {
	// Pix holds the image's pixels, in Gray, A order. The pixel at
	// (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*2].
	Pix []uint8
	// Stride is the Pix stride (in bytes) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
}

func (p *GrayA) ColorModel() color.Model { return ccolor.GrayAModel }

func (p *GrayA) Bounds() image.Rectangle { return p.Rect }

func (p *GrayA) At(x, y int) color.Color {
	return p.GrayAAt(x, y)
}

func (p *GrayA) GrayAAt(x, y int) ccolor.GrayA {
	if !(image.Point{x, y}.In(p.Rect)) {
		return ccolor.GrayA{0, 0}
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+2 : i+2] // Small cap improves performance, see https://golang.org/issue/27857
	return ccolor.GrayA{s[0], s[1]}
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *GrayA) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*2
}

func (p *GrayA) Set(x, y int, c color.Color) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	c1 := ccolor.GrayAModel.Convert(c).(ccolor.GrayA)
	s := p.Pix[i : i+2 : i+2] // Small cap improves performance, see https://golang.org/issue/27857
	s[0] = c1.Y
	s[1] = c1.A
}

func (p *GrayA) SetGrayA(x, y int, c ccolor.GrayA) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+2 : i+2] // Small cap improves performance, see https://golang.org/issue/27857
	s[0] = c.Y
	s[1] = c.A
}

// SubImage returns an image representing the portion of the image p visible
// through r. The returned value shares pixels with the original image.
func (p *GrayA) SubImage(r image.Rectangle) image.Image {
	r = r.Intersect(p.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &GrayA{}
	}
	i := p.PixOffset(r.Min.X, r.Min.Y)
	return &GrayA{
		Pix:    p.Pix[i:],
		Stride: p.Stride,
		Rect:   r,
	}
}

// Opaque scans the entire image and reports whether it is fully opaque.
func (p *GrayA) Opaque() bool {
	if p.Rect.Empty() {
		return true
	}
	i0, i1 := 1, p.Rect.Dx()*2
	for y := p.Rect.Min.Y; y < p.Rect.Max.Y; y++ {
		for i := i0; i < i1; i += 2 {
			if p.Pix[i] != 0xff {
				return false
			}
		}
		i0 += p.Stride
		i1 += p.Stride
	}
	return true
}

// NewGrayA returns a new [GrayA] image with the given bounds.
func NewGrayA(r image.Rectangle) *GrayA {
	return &GrayA{
		Pix:    make([]uint8, 2*r.Dx()*r.Dy()),
		Stride: 2 * r.Dx(),
		Rect:   r,
	}
}

// GrayA32 is an in-memory image whose At method returns [color.RGBA64] values.
type GrayA32 struct {
	// Pix holds the image's pixels, in Gray, A order and big-endian format. The pixel at
	// (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*4].
	Pix []uint8
	// Stride is the Pix stride (in bytes) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
}

func (p *GrayA32) ColorModel() color.Model { return ccolor.GrayA32Model }

func (p *GrayA32) Bounds() image.Rectangle { return p.Rect }

func (p *GrayA32) At(x, y int) color.Color {
	return p.GrayA32At(x, y)
}

func (p *GrayA32) GrayA32At(x, y int) ccolor.GrayA32 {
	if !(image.Point{x, y}.In(p.Rect)) {
		return ccolor.GrayA32{0, 0}
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+4 : i+4] // Small cap improves performance, see https://golang.org/issue/27857
	return ccolor.GrayA32{
		uint16(s[0])<<8 | uint16(s[1]),
		uint16(s[2])<<8 | uint16(s[3]),
	}
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *GrayA32) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*4
}

func (p *GrayA32) Set(x, y int, c color.Color) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	c1 := ccolor.GrayA32Model.Convert(c).(ccolor.GrayA32)
	s := p.Pix[i : i+4 : i+4] // Small cap improves performance, see https://golang.org/issue/27857
	s[0] = uint8(c1.Y >> 8)
	s[1] = uint8(c1.Y)
	s[2] = uint8(c1.A >> 8)
	s[3] = uint8(c1.A)
}

func (p *GrayA32) SetGrayA32(x, y int, c ccolor.GrayA32) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+4 : i+4] // Small cap improves performance, see https://golang.org/issue/27857
	s[0] = uint8(c.Y >> 8)
	s[1] = uint8(c.Y)
	s[2] = uint8(c.A >> 8)
	s[3] = uint8(c.A)
}

// SubImage returns an image representing the portion of the image p visible
// through r. The returned value shares pixels with the original image.
func (p *GrayA32) SubImage(r image.Rectangle) image.Image {
	r = r.Intersect(p.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &GrayA32{}
	}
	i := p.PixOffset(r.Min.X, r.Min.Y)
	return &GrayA32{
		Pix:    p.Pix[i:],
		Stride: p.Stride,
		Rect:   r,
	}
}

// Opaque scans the entire image and reports whether it is fully opaque.
func (p *GrayA32) Opaque() bool {
	return true
}

// NewGrayA32 returns a new [GrayA32] image with the given bounds.
func NewGrayA32(r image.Rectangle) *GrayA32 {
	return &GrayA32{
		Pix:    make([]uint8, 4*r.Dx()*r.Dy()),
		Stride: 4 * r.Dx(),
		Rect:   r,
	}
}
