package canvas

import (
	"bytes"
	"testing"
)

func TestEPS(t *testing.T) {
	w := &bytes.Buffer{}
	eps := NewEPS(w, 100, 80)
	eps.setColor(Red)
	//test.String(t, string(w.Bytes()), "")
}
