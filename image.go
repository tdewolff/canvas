package canvas

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

// Image allows the renderer to optimize specific cases
type Image struct {
	image.Image
	Bytes    []byte
	Mimetype string // image/png or image/jpeg for instance
}

// NewJPEGImage parses a reader to later give access to the JPEG raw bytes.
func NewJPEGImage(r io.Reader) (Image, error) {
	return newImage("image/jpeg", jpeg.Decode, r)
}

// NewPNGImage parses a reader to later give access to the PNG raw bytes
func NewPNGImage(r io.Reader) (Image, error) {
	return newImage("image/png", png.Decode, r)
}

func newImage(mimetype string, decode func(io.Reader) (image.Image, error), r io.Reader) (Image, error) {
	var buffer bytes.Buffer
	r = io.TeeReader(r, &buffer)
	img, err := decode(r)
	return Image{
		Image:    img,
		Bytes:    buffer.Bytes(),
		Mimetype: mimetype,
	}, err
}
