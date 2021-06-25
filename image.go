package canvas

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

// ImageEncoding defines whether the embedded image shall be embedded as lossless (typically PNG) or lossy (typically JPG).
type ImageEncoding int

// see ImageEncoding
const (
	Lossless ImageEncoding = iota
	Lossy
)

// Image is a raster image. Keeping the original bytes allows the renderer to optimize rendering in some cases.
type Image struct {
	image.Image
	Mimetype string
	Bytes    []byte
}

// NewJPEGImage parses a JPEG image.
func NewJPEGImage(r io.Reader) (Image, error) {
	return newImage("image/jpeg", jpeg.Decode, r)
}

// NewPNGImage parses a PNG image
func NewPNGImage(r io.Reader) (Image, error) {
	return newImage("image/png", png.Decode, r)
}

func newImage(mimetype string, decode func(io.Reader) (image.Image, error), r io.Reader) (Image, error) {
	// TODO: use lazy decoding
	var buffer bytes.Buffer
	r = io.TeeReader(r, &buffer)
	img, err := decode(r)
	return Image{
		Image:    img,
		Bytes:    buffer.Bytes(),
		Mimetype: mimetype,
	}, err
}
