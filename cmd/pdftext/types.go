package main

import (
	"bytes"
	"compress/zlib"
	"encoding/ascii85"
	"fmt"
	"io"
	"strings"
	"time"
)

type pdfRef [2]uint32
type pdfName string
type pdfArray []interface{}
type pdfDict map[string]interface{}
type pdfStream struct {
	dict    pdfDict
	filters []pdfName
	data    []byte
}

func (v pdfRef) String() string {
	return fmt.Sprintf("%d %d R", v[0], v[1])
}

const (
	pdfFilterASCII85 pdfName = "ASCII85Decode"
	pdfFilterFlate   pdfName = "FlateDecode"
)

func (v pdfStream) SetFilters(filters []pdfName) {
	v.filters = filters
}

func (v pdfStream) Decompress() (pdfStream, error) {
	if len(v.filters) == 0 {
		delete(v.dict, "Filter")
		return v, nil
	}

	b := v.data
	for i := len(v.filters) - 1; 0 <= i; i-- {
		filter := v.filters[i]
		switch filter {
		case pdfFilterASCII85:
			var err error
			b = b[:len(b)-2] // remove ~> characters
			r := ascii85.NewDecoder(bytes.NewReader(b))
			b, err = io.ReadAll(r)
			if err != nil {
				return pdfStream{}, err
			}
		case pdfFilterFlate:
			r, err := zlib.NewReader(bytes.NewReader(b))
			if err != nil {
				return pdfStream{}, err
			}
			b, err = io.ReadAll(r)
			if err != nil {
				return pdfStream{}, err
			}
			r.Close()
		default:
			return pdfStream{}, fmt.Errorf("unsupported filter: %v", filter)
		}
	}
	delete(v.dict, "Filter")
	v.dict["Length"] = len(b)
	v.data = b
	return v, nil
}

func (v pdfStream) Compress() (pdfStream, error) {
	if len(v.filters) == 0 {
		return v, nil
	}

	b := v.data
	filterArray := make(pdfArray, len(v.filters))
	for i, filter := range v.filters {
		var b2 bytes.Buffer
		switch filter {
		case pdfFilterASCII85:
			w := ascii85.NewEncoder(&b2)
			w.Write(b)
			w.Write([]byte("~>"))
			w.Close()
		case pdfFilterFlate:
			w := zlib.NewWriter(&b2)
			w.Write(b)
			w.Close()
		default:
			return pdfStream{}, fmt.Errorf("unsupported filter: %v", filter)
		}
		b = b2.Bytes()
		filterArray[len(filterArray)-i-1] = filter
	}
	v.dict["Filter"] = filterArray
	v.dict["Length"] = len(b)
	v.data = b
	return v, nil
}

type pdfInfo struct {
	Title        string
	Author       string
	Subject      string
	Keywords     string
	Creator      string
	Producer     string
	CreationDate time.Time
	ModDate      time.Time
}

var dateFormat = "2006-01-02 15:04:05 UTC-0700"

func (d pdfInfo) String() string {
	sb := strings.Builder{}
	if d.Title != "" {
		sb.WriteString("Title: " + d.Title + "\n")
	}
	if d.Author != "" {
		sb.WriteString("Author: " + d.Author + "\n")
	}
	if d.Subject != "" {
		sb.WriteString("Subject: " + d.Subject + "\n")
	}
	if d.Keywords != "" {
		sb.WriteString("Keywords: " + d.Keywords + "\n")
	}
	if d.Creator != "" {
		sb.WriteString("Creator: " + d.Creator + "\n")
	}
	if d.Producer != "" {
		sb.WriteString("Producer: " + d.Producer + "\n")
	}
	if !d.CreationDate.IsZero() {
		sb.WriteString("CreationDate: " + d.CreationDate.Format(dateFormat) + "\n")
	}
	if !d.ModDate.IsZero() {
		sb.WriteString("ModDate: " + d.ModDate.Format(dateFormat) + "\n")
	}
	s := sb.String()
	if 0 < len(s) {
		s = s[:len(s)-1]
	}
	return s
}
