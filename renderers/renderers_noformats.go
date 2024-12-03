//go:build !formats

package renderers

import (
	"fmt"

	"github.com/Seanld/canvas"
)

// WebP returns a Webp writer that uses libwebp and accepts the following options: canvas.Resolution, canvas.Colorspace, github.com/kolesa-team/go-webp/encoder.*Options
func WebP(opts ...interface{}) canvas.Writer {
	return errorWriter(fmt.Errorf("unsupported WebP: CGO must be enabled"))
}

// AVIF returns a AVIF writer that uses libaom and accepts the following options: canvas.Resolution, canvas.Colorspace, github.com/Kagami/go-avif.*Options
func AVIF(opts ...interface{}) canvas.Writer {
	return errorWriter(fmt.Errorf("unsupported AVIF: CGO must be enabled"))
}
