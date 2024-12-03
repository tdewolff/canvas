package ps

import (
	"bytes"
	"testing"

	"github.com/Seanld/canvas"
)

func TestPS(t *testing.T) {
	w := &bytes.Buffer{}
	ps := New(w, 100, 80, nil)
	ps.setPaint(canvas.Paint{Color: canvas.Red})
	//test.String(t, string(w.Bytes()), "")
}
