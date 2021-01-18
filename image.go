package canvas

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

type ImageMimetype int

const (
	ImageUnknown ImageMimetype = iota
	ImageJPEG
	ImagePNG
)

func (m ImageMimetype) String() string {
	switch m {
	case ImageJPEG:
		return "image/jpeg"
	case ImagePNG:
		return "image/png"
	}
	return "image/unknown"
}

// Image allows the renderer to optimize specific cases
type Image struct {
	image.Image
	Bytes    []byte
	Mimetype ImageMimetype
}

// NewJPEGImage parses a reader to later give access to the JPEG raw bytes.
func NewJPEGImage(r io.Reader) (Image, error) {
	return newImage(ImageJPEG, jpeg.Decode, r)
}

// NewPNGImage parses a reader to later give access to the PNG raw bytes
func NewPNGImage(r io.Reader) (Image, error) {
	return newImage(ImagePNG, png.Decode, r)
}

func newImage(mimetype ImageMimetype, decode func(io.Reader) (image.Image, error), r io.Reader) (Image, error) {
	var buffer bytes.Buffer
	r = io.TeeReader(r, &buffer)
	img, err := decode(r)
	return Image{
		Image:    img,
		Bytes:    buffer.Bytes(),
		Mimetype: mimetype,
	}, err
}
