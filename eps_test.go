package canvas

import (
	"bytes"
	"testing"
)

func TestEPS(t *testing.T) {
	w := &bytes.Buffer{}
	eps := newEPSWriter(w, 100, 80)
	eps.SetColor(Red)
	//test.String(t, string(w.Bytes()), "")
}
