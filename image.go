package canvas

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

// JPEGImage gives access to the raw bytes.
// Should be used with PDF, only for baseline JPEGs
// (progressive might not be displayed properly)
type JPEGImage interface {
	image.Image
	JPEGBytes() []byte
}

type jpegImage struct {
	bufferedImage
}

func (i jpegImage) JPEGBytes() []byte {
	return i.bytes
}

// NewJPEGImage parses a reader to later give access to the JPEG raw bytes.
// Should be used with PDF, only for baseline JPEGs
// (progressive might not be displayed properly)
func NewJPEGImage(r io.Reader) (JPEGImage, error) {
	bi, err := newBufferedImage(jpeg.Decode, r)
	if err != nil {
		return nil, err
	}
	return jpegImage{bi}, nil
}

// PNGImage gives access to the raw bytes
type PNGImage interface {
	image.Image
	PNGBytes() []byte
}

type pngImage struct {
	bufferedImage
}

func (i pngImage) PNGBytes() []byte {
	return i.bytes
}

// NewPNGImage parses a reader to later give access to the PNG raw bytes
func NewPNGImage(r io.Reader) (PNGImage, error) {
	bi, err := newBufferedImage(png.Decode, r)
	if err != nil {
		return nil, err
	}
	return pngImage{bi}, nil
}

// bufferedImage is a generic struct for holding specific decoders
type bufferedImage struct {
	image.Image
	bytes []byte
}

func newBufferedImage(decode func(io.Reader) (image.Image, error), r io.Reader) (bufferedImage, error) {
	var buffer bytes.Buffer
	r = io.TeeReader(r, &buffer)
	img, err := decode(r)
	return bufferedImage{
		Image: img,
		bytes: buffer.Bytes(),
	}, err
}
