package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"
)

type crossReference struct {
	offset     int
	generation uint32
	inuse      bool
}

type objectOffsetSorter struct {
	refs    []pdfRef
	offsets []int
}

func sortObjectOffsets(objects map[pdfRef]int) ([]pdfRef, []int) {
	refs := make([]pdfRef, 0, len(objects))
	offsets := make([]int, 0, len(objects))
	for ref, offset := range objects {
		refs = append(refs, ref)
		offsets = append(offsets, offset)
	}
	sorter := &objectOffsetSorter{refs, offsets}
	sort.Sort(sorter)
	return sorter.refs, sorter.offsets
}

func (s *objectOffsetSorter) Len() int {
	return len(s.refs)
}

func (s *objectOffsetSorter) Swap(i, j int) {
	s.refs[i], s.refs[j] = s.refs[j], s.refs[i]
	s.offsets[i], s.offsets[j] = s.offsets[j], s.offsets[i]
}

func (s *objectOffsetSorter) Less(i, j int) bool {
	return s.offsets[i] < s.offsets[j]
}

type pdfWriter struct {
	w   io.Writer
	pos int
	err error

	r       *pdfReader
	objects map[pdfRef][]byte
}

func NewPDFWriter(w io.Writer, r *pdfReader) *pdfWriter {
	return &pdfWriter{
		w:       w,
		r:       r,
		objects: map[pdfRef][]byte{},
	}
}

func (w *pdfWriter) writeBytes(b []byte) {
	if w.err != nil {
		return
	}
	n, err := w.w.Write(b)
	w.pos += n
	w.err = err
}

func (w *pdfWriter) write(s string, v ...interface{}) {
	if w.err != nil {
		return
	}
	n, err := fmt.Fprintf(w.w, s, v...)
	w.pos += n
	w.err = err
}

func (w *pdfWriter) SetObjectData(ref pdfRef, data []byte) {
	w.objects[ref] = data
}

func (w *pdfWriter) SetObject(ref pdfRef, val interface{}) error {
	buf := &bytes.Buffer{}
	if err := pdfWriteVal(buf, w.r, ref, val); err != nil {
		return err
	}
	w.SetObjectData(ref, buf.Bytes())
	return nil
}

func (w *pdfWriter) Close() error {
	if len(w.r.objects) == 0 {
		w.writeBytes(w.r.data)
		return w.err
	}

	refs, offsets := sortObjectOffsets(w.r.objects)
	offsets = append(offsets, w.r.startxref)

	// header
	w.writeBytes(w.r.data[:offsets[0]])

	// objects
	xrefs := make([]crossReference, 0, len(w.r.objects))
	for i := 0; i < len(refs); i++ {
		ref, offset := refs[i], offsets[i]
		xrefs = append(xrefs, crossReference{w.pos, ref[1], true})
		if data, ok := w.objects[ref]; ok {
			// replaced
			w.write("%v %v obj\n", ref[0], ref[1])
			w.writeBytes(data)
			w.write("\nendobj\n")
		} else {
			w.writeBytes(w.r.data[offset:offsets[i+1]])
		}
	}

	// new objects
	for ref, data := range w.objects {
		if _, ok := w.r.objects[ref]; !ok {
			xrefs = append(xrefs, crossReference{w.pos, ref[1], true})
			w.write("%v %v obj\n", ref[0], ref[1])
			w.writeBytes(data)
			w.write("\nendobj\n")
		}
	}

	// xref
	startxref := w.pos
	w.write("xref\n0 %d\n0000000000 65535 f \n", len(xrefs)+1)
	for _, xref := range xrefs {
		inuse := 'n'
		if !xref.inuse {
			inuse = 'f'
		}
		w.write("%010d %05d %c \n", xref.offset, xref.generation, inuse)
	}
	w.write("trailer\n")
	pdfWriteVal(w.w, nil, pdfRef{}, w.r.trailer)
	w.write("\nstartxref\n%v\n%%%%EOF\n", startxref)
	return w.err
}

func pdfWriteVal(w io.Writer, r *pdfReader, ref pdfRef, i interface{}) error {
	switch v := i.(type) {
	case bool:
		if v {
			fmt.Fprintf(w, "true")
		} else {
			fmt.Fprintf(w, "false")
		}
	case int:
		fmt.Fprintf(w, "%d", v)
	case float64:
		fmt.Fprintf(w, "%v", dec(v))
	case string:
		v = strings.Replace(v, `\`, `\\`, -1)
		v = strings.Replace(v, `(`, `\(`, -1)
		v = strings.Replace(v, `)`, `\)`, -1)
		fmt.Fprintf(w, "(%v)", v)
	case []byte:
		w.Write([]byte("<"))
		hex.NewEncoder(w).Write(v)
		w.Write([]byte(">"))
	case pdfRef:
		fmt.Fprintf(w, "%v", v)
	case pdfName:
		fmt.Fprintf(w, "/%v", v)
	case pdfArray:
		fmt.Fprintf(w, "[")
		for j, val := range v {
			if j != 0 {
				fmt.Fprintf(w, " ")
			}
			pdfWriteVal(w, r, ref, val)
		}
		fmt.Fprintf(w, "]")
	case pdfDict:
		fmt.Fprintf(w, "<< ")
		if val, ok := v["Type"]; ok {
			fmt.Fprintf(w, "/Type ")
			pdfWriteVal(w, r, ref, val)
			fmt.Fprintf(w, " ")
		}
		if val, ok := v["Subtype"]; ok {
			fmt.Fprintf(w, "/Subtype ")
			pdfWriteVal(w, r, ref, val)
			fmt.Fprintf(w, " ")
		}
		keys := []string{}
		for key := range v {
			if key != "Type" && key != "Subtype" {
				keys = append(keys, string(key))
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			pdfWriteVal(w, nil, pdfRef{}, pdfName(key))
			fmt.Fprintf(w, " ")
			pdfWriteVal(w, r, ref, v[key])
			fmt.Fprintf(w, " ")
		}
		fmt.Fprintf(w, ">>")
	case pdfStream:
		if v.dict == nil {
			v.dict = pdfDict{}
		}
		var err error
		if v, err = v.Compress(); err != nil {
			return err
		}
		if r.encrypt.isEncrypted {
			// TODO: apply to all strings (possibly nested) and streams
			v.data = r.encrypt.Encrypt(ref, v.data)
		}
		pdfWriteVal(w, r, ref, v.dict)
		fmt.Fprintf(w, " stream\n")
		w.Write(v.data)
		fmt.Fprintf(w, "\nendstream")
	default:
		return fmt.Errorf("unknown PDF type %T", i)
	}
	return nil
}
