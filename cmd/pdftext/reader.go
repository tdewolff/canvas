package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/tdewolff/parse/v2/strconv"
)

var noEncryptRef = pdfRef{0, 0}

type pdfObject struct {
	free, compressed bool
	offset           int
	object           uint32 // compressed or next free object number
}

type pdfReader struct {
	data    []byte
	objects map[pdfRef]pdfObject
	trailer pdfDict
	encrypt pdfEncrypt
	kids    []pdfRef

	startxref int
	eol       []byte
	cache     map[pdfRef]interface{}
}

func NewPDFReader(reader io.Reader, password string) (*pdfReader, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	} else if len(data) < 8 || !bytes.Equal(data[:7], []byte("%PDF-1.")) || data[7] < '0' || '7' < data[7] {
		return nil, fmt.Errorf("invalid PDF file: bad version")
	}

	r := &pdfReader{
		data:    data,
		objects: map[pdfRef]pdfObject{},
		cache:   map[pdfRef]interface{}{},
	}

	// get startxref
	var line []byte
	lrr := newLineReaderReverse(r.data, len(r.data))
	if line = lrr.Next(); !bytes.Equal(line, []byte("%%EOF")) {
		return nil, fmt.Errorf("invalid PDF file")
	}
	if r.data[lrr.Pos()] == '\r' && r.data[lrr.Pos()+1] == '\n' {
		r.eol = r.data[lrr.Pos() : lrr.Pos()+2]
	} else {
		r.eol = r.data[lrr.Pos() : lrr.Pos()+1]
	}
	num, _ := strconv.ParseUint(lrr.Next())
	if num == 0 {
		return nil, fmt.Errorf("invalid PDF file")
	} else if line = lrr.Next(); !bytes.Equal(line, []byte("startxref")) {
		return nil, fmt.Errorf("invalid PDF file")
	}
	startxref := int(num)
	//endtrailer := lrr.Pos()

	if r.trailer, err = r.readTrailer(startxref); err != nil {
		return nil, err
	} else if err := r.readEncrypt([]byte(password)); err != nil {
		return nil, err
	} else if err := r.readKids(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *pdfReader) readCrossReferenceTable() (pdfDict, error) {
	var starttrailer int
	var line []byte

	lr := newLineReader(r.data, r.startxref)
	_ = lr.Next() // xref
	for {
		starttrailer = lr.Pos()
		line = lr.Next()
		if line == nil {
			return pdfDict{}, fmt.Errorf("invalid cross reference table")
		} else if bytes.HasPrefix(line, []byte("trailer")) {
			break
		}

		first, n := strconv.ParseUint(line)
		if n == 0 {
			return pdfDict{}, fmt.Errorf("invalid cross reference table")
		}
		i := moveWhiteSpace(line, n)
		entries, n := strconv.ParseUint(line[i:])
		if n == 0 {
			return pdfDict{}, fmt.Errorf("invalid cross reference table")
		}

		for i := uint32(0); i < uint32(entries); i++ {
			line = lr.Next()
			if len(line) != 18 && len(line) != 19 {
				return pdfDict{}, fmt.Errorf("invalid cross reference table")
			}
			offset, n := strconv.ParseUint(line)
			if n != 10 {
				return pdfDict{}, fmt.Errorf("invalid cross reference table")
			}
			generation, n := strconv.ParseUint(line[11:])
			if n != 5 || line[17] != 'f' && line[17] != 'n' {
				return pdfDict{}, fmt.Errorf("invalid cross reference table")
			}
			free := line[17] == 'f'
			if i == 0 && (!free || generation != 65535) {
				return pdfDict{}, fmt.Errorf("invalid cross reference table")
			}
			ref := pdfRef{uint32(first) + i, uint32(generation)}
			if _, ok := r.objects[ref]; !ok {
				// add object of previous generations only if not over-written by a new version
				if free {
					r.objects[ref] = pdfObject{free: true, object: uint32(offset)}
				} else {
					r.objects[ref] = pdfObject{offset: int(offset)}
				}
			}
		}
	}

	// trailer
	starttrailer = moveWhiteSpace(r.data, starttrailer+7)
	itrailer, _, err := pdfReadVal(r, noEncryptRef, r.data[starttrailer:])
	if err != nil {
		return pdfDict{}, fmt.Errorf("invalid trailer: %w", err)
	} else if _, ok := itrailer.(pdfDict); !ok {
		return pdfDict{}, fmt.Errorf("invalid trailer: must be dictionary")
	}
	trailer := itrailer.(pdfDict)
	return trailer, nil
}

func (r *pdfReader) readCrossReferenceStream() (pdfDict, error) {
	istream, err := r.readObjectAt(noEncryptRef, r.startxref)
	if err != nil {
		return pdfDict{}, fmt.Errorf("invalid cross reference stream: %w", err)
	}
	stream, ok := istream.(pdfStream)
	if !ok {
		return pdfDict{}, fmt.Errorf("invalid cross reference stream")
	}
	size, err := r.GetInt(stream.dict["Size"])
	if err != nil {
		return pdfDict{}, fmt.Errorf("invalid cross reference stream")
	}
	ws, err := r.GetArray(stream.dict["W"])
	if err != nil || len(ws) != 3 {
		return pdfDict{}, fmt.Errorf("invalid cross reference stream")
	}
	W := [3]int{}
	for i, w := range ws {
		W[i], err = r.GetInt(w)
		if err != nil || 4 < W[i] {
			return pdfDict{}, fmt.Errorf("invalid cross reference stream")
		}
	}
	indices := make([]int, size)
	if _, ok := stream.dict["Index"]; ok {
		is, err := r.GetArray(stream.dict["Index"])
		if err != nil {
			return pdfDict{}, fmt.Errorf("invalid cross reference stream")
		}
		d := 0
		for i := 0; i < len(is); i += 2 {
			first, err1 := r.GetInt(is[i+0])
			n, err2 := r.GetInt(is[i+1])
			if err1 != nil || err2 != nil {
				return pdfDict{}, fmt.Errorf("invalid cross reference stream")
			}
			for j := 0; j < n; j++ {
				indices[d+j] = first + j
			}
			d += n
		}
		indices = indices[:d]
	} else {
		for i := 0; i < size; i++ {
			indices[i] = i
		}
	}
	dW := W[0] + W[1] + W[2]
	if len(stream.data) != dW*size || W[1] == 0 {
		return pdfDict{}, fmt.Errorf("invalid cross reference stream")
	}
	for i := 0; i < size; i++ {
		if len(indices) <= i {
			break
		}

		d := i * dW
		t := uint32(1)
		if W[0] != 0 {
			t = readNumberLE(stream.data[d:], W[0])
		}
		index := indices[i]
		field2 := readNumberLE(stream.data[d+W[0]:], W[1])
		field3 := uint32(0)
		if W[2] != 0 {
			field3 = readNumberLE(stream.data[d+W[0]+W[1]:], W[2])
		}
		if t == 0 {
			// free object
			ref := pdfRef{uint32(index), field3}
			if _, ok := r.objects[ref]; !ok {
				r.objects[ref] = pdfObject{free: true, object: field2}
			}
		} else if t == 1 {
			// used object
			ref := pdfRef{uint32(index), field3}
			if _, ok := r.objects[ref]; !ok {
				// add object of previous generations only if not over-written by a new version
				r.objects[ref] = pdfObject{offset: int(field2)}
			}
		} else if t == 2 {
			// compressed object
			ref := pdfRef{uint32(index), 0}
			if _, ok := r.objects[ref]; !ok {
				// add object of previous generations only if not over-written by a new version
				r.objects[ref] = pdfObject{compressed: true, offset: int(field3), object: field2}
			}
		} else {
			// no-op
		}
	}
	if prev, err := r.GetInt(stream.dict["Prev"]); err == nil {
		if _, err = r.readTrailer(prev); err != nil {
			return pdfDict{}, err
		}
	}

	trailer := stream.dict
	delete(trailer, "Type")
	delete(trailer, "Index")
	delete(trailer, "W")
	delete(trailer, "Length")
	return trailer, nil
}

func (r *pdfReader) readTrailer(startxref int) (pdfDict, error) {
	r.startxref = startxref

	var trailer pdfDict
	var err error
	if bytes.HasPrefix(r.data[startxref:], []byte("xref")) {
		trailer, err = r.readCrossReferenceTable()
	} else {
		trailer, err = r.readCrossReferenceStream()
	}
	if err != nil {
		return pdfDict{}, err
	}
	if prev, err := r.GetInt(trailer["Prev"]); err == nil {
		if _, err = r.readTrailer(prev); err != nil {
			return pdfDict{}, err
		}
	}
	return trailer, nil
}

func (r *pdfReader) readKids() error {
	// kids
	root, err := r.GetDict(r.trailer["Root"])
	if err != nil || root["Type"] != pdfName("Catalog") {
		return fmt.Errorf("bad /Catalog object")
	}
	pages, err := r.GetDict(root["Pages"])
	if err != nil || pages["Type"] != pdfName("Pages") {
		return fmt.Errorf("bad /Pages object")
	} else if err = r.addKids(pages); err != nil {
		return err
	}
	return nil
}

func (r *pdfReader) addKids(pages pdfDict) error {
	kids, err := r.GetArray(pages["Kids"])
	if err != nil {
		return fmt.Errorf("missing or invalid Kids entry in Pages object")
	}
	for _, kid := range kids {
		obj, err := r.GetDict(kid)
		if err != nil || (obj["Type"] != pdfName("Pages") && obj["Type"] != pdfName("Page")) {
			return fmt.Errorf("bad Kids entry")
		}
		if obj["Type"] == pdfName("Page") {
			r.kids = append(r.kids, kid.(pdfRef))
		} else {
			r.addKids(obj)
		}
	}
	return nil
}

func (r *pdfReader) get(val interface{}) (interface{}, error) {
	for {
		ref, ok := val.(pdfRef)
		if !ok {
			break
		}
		var err error
		if val, err = r.readObject(ref); err != nil {
			return nil, err
		}
	}
	return val, nil
}

func (r *pdfReader) GetName(val interface{}) (pdfName, error) {
	val, err := r.get(val)
	if err != nil {
		return "", err
	}
	name, ok := val.(pdfName)
	if !ok {
		return "", fmt.Errorf("not a name or missing")
	}
	return name, nil
}

func (r *pdfReader) GetInt(val interface{}) (int, error) {
	val, err := r.get(val)
	if err != nil {
		return 0, err
	}
	i, ok := val.(int)
	if !ok {
		return 0, fmt.Errorf("not an integer or missing")
	}
	return i, nil
}

func (r *pdfReader) GetString(val interface{}) ([]byte, error) {
	val, err := r.get(val)
	if err != nil {
		return nil, err
	}
	i, ok := val.([]byte)
	if !ok {
		return nil, fmt.Errorf("not a string or missing")
	}
	return i, nil
}

func (r *pdfReader) GetArray(val interface{}) (pdfArray, error) {
	val, err := r.get(val)
	if err != nil {
		return pdfArray{}, err
	}
	array, ok := val.(pdfArray)
	if !ok {
		return pdfArray{}, fmt.Errorf("not an array or missing")
	}
	return array, nil
}

func (r *pdfReader) GetDict(val interface{}) (pdfDict, error) {
	val, err := r.get(val)
	if err != nil {
		return pdfDict{}, err
	}
	dict, ok := val.(pdfDict)
	if !ok {
		return pdfDict{}, fmt.Errorf("not a dictionary or missing")
	}
	return dict, nil
}

func (r *pdfReader) GetStream(val interface{}) (pdfStream, error) {
	val, err := r.get(val)
	if err != nil {
		return pdfStream{}, err
	}
	stream, ok := val.(pdfStream)
	if !ok {
		return pdfStream{}, fmt.Errorf("not a stream or missing")
	}
	return stream, nil
}

func (r *pdfReader) GetInfo() pdfInfo {
	v := pdfInfo{}
	info, err := r.GetDict(r.trailer["Info"])
	if err != nil {
		return v
	}
	if title, err := r.GetString(info["Title"]); err == nil {
		v.Title = parseTextString(title)
	}
	if author, err := r.GetString(info["Author"]); err == nil {
		v.Author = parseTextString(author)
	}
	if subject, err := r.GetString(info["Subject"]); err == nil {
		v.Subject = parseTextString(subject)
	}
	if keywords, err := r.GetString(info["Keywords"]); err == nil {
		v.Keywords = parseTextString(keywords)
	}
	if creator, err := r.GetString(info["Creator"]); err == nil {
		v.Creator = parseTextString(creator)
	}
	if producer, err := r.GetString(info["Producer"]); err == nil {
		v.Producer = parseTextString(producer)
	}
	if creationDate, err := r.GetString(info["CreationDate"]); err == nil {
		v.CreationDate = parseDate(creationDate)
	}
	if modDate, err := r.GetString(info["ModDate"]); err == nil {
		v.ModDate = parseDate(modDate)
	}
	return v
}

func (r *pdfReader) GetPages() []pdfRef {
	return r.kids
}

func (r *pdfReader) GetPage(index int) (pdfDict, []byte, error) {
	if len(r.kids) <= index {
		return nil, nil, fmt.Errorf("unknown page %d", index)
	}
	dict, err := r.GetDict(r.kids[index])
	if err != nil {
		return nil, nil, fmt.Errorf("bad page %d: %w", index, err)
	}

	if dict["Contents"] == nil {
		return dict, []byte{}, nil
	} else if array, err := r.GetArray(dict["Contents"]); err == nil {
		b := []byte{}
		for _, item := range array {
			contents, err := r.GetStream(item)
			if err != nil {
				return nil, nil, fmt.Errorf("bad page %d: %w", index, err)
			}
			b = append(b, contents.data...)
		}
		return dict, b, nil
	}

	contents, err := r.GetStream(dict["Contents"])
	if err != nil {
		return nil, nil, fmt.Errorf("bad page %d: %w", index, err)
	}
	return dict, contents.data, nil
}

func (r *pdfReader) readObject(ref pdfRef) (interface{}, error) {
	if val, ok := r.cache[ref]; ok {
		return val, nil
	}
	obj, ok := r.objects[ref]
	if !ok {
		return nil, fmt.Errorf("unknown object %v", ref)
	} else if obj.free {
		return nil, fmt.Errorf("bad object %v is free", ref)
	} else if obj.compressed {
		iobjectStream, err := r.readObject(pdfRef{obj.object, 0})
		if err != nil {
			return nil, err
		}
		objectStream, ok := iobjectStream.(pdfStream)
		if !ok {
			return nil, fmt.Errorf("compressed object %v must refer to stream", ref)
		}

		b, i := objectStream.data, 0
		object, offset := 0, 0
		for index := 0; index <= obj.offset; index++ {
			val, n, err := pdfReadContentVal(b[i:])
			if err != nil {
				return nil, fmt.Errorf("invalid stream: %w", err)
			}
			object, _ = val.(int)
			i = moveWhiteSpace(b, i+n)

			val, n, err = pdfReadContentVal(b[i:])
			if err != nil {
				return nil, fmt.Errorf("invalid stream: %w", err)
			}
			offset, _ = val.(int)
			i = moveWhiteSpace(b, i+n)
		}
		if uint32(object) != ref[0] {
			return nil, fmt.Errorf("bad object %v: invalid index in compressed object", ref)
		}

		first, _ := r.GetInt(objectStream.dict["First"])
		offset += first
		if len(b) <= offset {
			return nil, fmt.Errorf("bad object %v: invalid offset in compressed object", ref)
		}

		val, _, err := pdfReadVal(r, ref, b[offset:])
		if err != nil {
			return nil, fmt.Errorf("bad object %v: %w", ref, err)
		}
		r.cache[ref] = val
		return val, nil
	}
	val, err := r.readObjectAt(ref, obj.offset)
	r.cache[ref] = val
	return val, err
}

func (r *pdfReader) readObjectAt(ref pdfRef, i int) (interface{}, error) {
	b := r.data
	val, n, err := pdfReadContentVal(b[i:])
	if _, ok := val.(int); !ok || err != nil {
		return nil, fmt.Errorf("bad object %v", ref)
	}
	i = moveWhiteSpace(b, i+n)
	val, n, err = pdfReadContentVal(b[i:])
	if _, ok := val.(int); !ok || err != nil {
		return nil, fmt.Errorf("bad object %v", ref)
	}
	i = moveWhiteSpace(b, i+n)
	if len(b) <= i+3 || !bytes.Equal(b[i:i+3], []byte("obj")) {
		return nil, fmt.Errorf("bad object %v", ref)
	}
	i = moveWhiteSpace(b, i+3)

	if encryptRef, ok := r.trailer["Encrypt"].(pdfRef); ok && ref == encryptRef {
		ref = noEncryptRef
	}
	val, n, err = pdfReadVal(r, ref, b[i:])
	if err != nil {
		return nil, fmt.Errorf("bad object %v: %w", ref, err)
	}
	i = moveWhiteSpace(b, i+n)

	if len(b) <= i+6 || !bytes.Equal(b[i:i+6], []byte("endobj")) {
		return nil, fmt.Errorf("bad object %v", ref)
	}
	return val, nil
}

func pdfReadContentVal(b []byte) (interface{}, int, error) {
	return pdfReadVal(nil, pdfRef{}, b)
}

func pdfReadVal(r *pdfReader, ref pdfRef, b []byte) (interface{}, int, error) {
	if len(b) == 0 {
		return nil, 0, fmt.Errorf("bad value")
	}
	if '0' <= b[0] && b[0] <= '9' || b[0] == '+' || b[0] == '-' || b[0] == '.' {
		isFloat := b[0] == '.'
		i := 1
		for i < len(b) && ('0' <= b[i] && b[i] <= '9' || b[i] == '.') {
			if b[i] == '.' {
				isFloat = true
			}
			i++
		}
		if i == 1 && (b[0] == '+' || b[0] == '-' || b[0] == '.') {
			return nil, 0, fmt.Errorf("bad number")
		} else if isFloat {
			num, _ := strconv.ParseFloat(b[:i])
			return num, i, nil
		}
		num, _ := strconv.ParseInt(b[:i])
		return int(num), i, nil
	} else if b[0] == '/' {
		name, n, err := parseName(b[1:])
		if err != nil {
			return nil, 0, err
		}
		return pdfName(name), n + 1, nil
	} else if b[0] == '[' {
		i := moveWhiteSpace(b, 1)
		array := pdfArray{}
		for {
			if len(b) <= i {
				return nil, 0, fmt.Errorf("bad array")
			} else if b[i] == ']' {
				i++
				break
			} else if val, n, err := pdfReadVal(r, ref, b[i:]); err != nil {
				return nil, 0, err
			} else {
				i = moveWhiteSpace(b, i+n)
				if object, ok := val.(int); ok && r != nil {
					mark := i
					val2, n, err := pdfReadContentVal(b[i:])
					if generation, ok := val2.(int); ok && err == nil && 0 <= generation {
						i = moveWhiteSpace(b, i+n)
						if i < len(b) && b[i] == 'R' {
							val = pdfRef{uint32(object), uint32(generation)}
							i = moveWhiteSpace(b, i+1)
						} else {
							i = mark
						}
					}
				}
				array = append(array, val)
			}
		}
		return array, i, nil
	} else if b[0] == '<' && 0 < len(b) && b[1] == '<' {
		i := moveWhiteSpace(b, 2)
		dict := pdfDict{}
		for {
			if len(b) <= i {
				return nil, 0, fmt.Errorf("bad dict")
			} else if i+1 < len(b) && b[i] == '>' && b[i+1] == '>' {
				i += 2
				break
			}

			val, n, err := pdfReadContentVal(b[i:])
			key, ok := val.(pdfName)
			if err != nil {
				return nil, 0, err
			} else if !ok {
				return nil, 0, fmt.Errorf("bad dict")
			}
			i = moveWhiteSpace(b, i+n)

			val, n, err = pdfReadVal(r, ref, b[i:])
			if err != nil {
				return nil, 0, err
			}
			i = moveWhiteSpace(b, i+n)
			if object, ok := val.(int); ok && r != nil {
				mark := i
				val2, n, err := pdfReadContentVal(b[i:])
				if generation, ok := val2.(int); ok && err == nil && 0 <= generation {
					i = moveWhiteSpace(b, i+n)
					if i < len(b) && b[i] == 'R' {
						val = pdfRef{uint32(object), uint32(generation)}
						i = moveWhiteSpace(b, i+1)
					} else {
						i = mark
					}
				}
			}

			dict[string(key)] = val
		}
		i = moveWhiteSpace(b, i)
		if r != nil && i+7 < len(b) && bytes.Equal(b[i:i+6], []byte("stream")) {
			i += 6
			if b[i] == '\n' || b[i] == '\r' {
				if b[i] == '\r' && i+1 < len(b) && b[i+1] == '\n' {
					i++
				}
				i++
			} else {
				return nil, 0, fmt.Errorf("bad stream")
			}

			length, err := r.GetInt(dict["Length"])
			if err != nil {
				return nil, 0, fmt.Errorf("bad stream length: %w", err)
			} else if len(b) <= i+length {
				return nil, 0, fmt.Errorf("bad stream")
			}

			var filters []pdfName
			if _, ok := dict["Filter"]; ok {
				var fs pdfArray
				if f, err := r.GetName(dict["Filter"]); err == nil {
					fs = pdfArray{f}
				} else if fs, err = r.GetArray(dict["Filter"]); err != nil {
					return nil, 0, fmt.Errorf("bad stream filters")
				}
				filters = make([]pdfName, len(fs))
				for i, f := range fs {
					filters[len(filters)-i-1] = f.(pdfName)
				}
			}

			var params []pdfDict
			if _, ok := dict["DecodeParms"]; ok {
				if p, err := r.GetDict(dict["DecodeParms"]); err == nil && len(filters) == 1 {
					params = []pdfDict{p}
				} else if ps, err := r.GetArray(dict["DecodeParms"]); err == nil && len(ps) == len(filters) {
					for _, ip := range ps {
						if p, err := r.GetDict(ip); err == nil {
							params = append(params, p)
						} else if ip == nil {
							params = append(params, pdfDict{})
						} else {
							return nil, 0, fmt.Errorf("bad stream decode parameters")
						}
					}
				} else {
					return nil, 0, fmt.Errorf("bad stream decode parameters")
				}
			} else {
				for i := 0; i < len(filters); i++ {
					params = append(params, pdfDict{})
				}
			}
			// dereference pdf references
			for _, ps := range params {
				for key, val := range ps {
					if _, ok := val.(pdfRef); ok {
						ps[key], err = r.get(val)
						if err != nil {
							return nil, 0, fmt.Errorf("bad stream decode parameters: %w", err)
						}
					}
				}
			}

			stream := pdfStream{dict, filters, params, b[i : i+length]}
			if r.encrypt.isEncrypted && ref != noEncryptRef {
				stream.data = r.encrypt.Decrypt(ref, stream.data)
			}
			if stream, err = stream.Decompress(); err != nil {
				return nil, 0, err
			}

			i = moveWhiteSpace(b, i+length)
			if len(b) <= i+9 || !bytes.Equal(b[i:i+9], []byte("endstream")) {
				return nil, 0, fmt.Errorf("bad stream")
			}
			i = moveWhiteSpace(b, i+9)
			return stream, i, nil
		}
		return dict, i, nil
	} else if b[0] == '(' {
		var s []byte
		j := 1 // start in b
		i := 1
		level := 0
		for i < len(b) {
			if b[i] == '(' {
				level++
			} else if b[i] == ')' {
				if level == 0 {
					break
				}
				level--
			} else if i+1 < len(b) && b[i] == '\\' {
				s = append(s, b[j:i]...)
				if b[i+1] == 'n' {
					s = append(s, '\n')
					i++
				} else if b[i+1] == 'r' {
					s = append(s, '\r')
					i++
				} else if b[i+1] == 't' {
					s = append(s, '\t')
					i++
				} else if b[i+1] == 'b' {
					s = append(s, '\b')
					i++
				} else if b[i+1] == 'f' {
					s = append(s, '\f')
					i++
				} else if b[i+1] == '(' {
					s = append(s, '(')
					i++
				} else if b[i+1] == ')' {
					s = append(s, ')')
					i++
				} else if b[i+1] == '\\' {
					s = append(s, '\\')
					i++
				} else if '0' <= b[i+1] && b[i+1] <= '7' {
					num := int(b[i+1] - '0')
					i++
					k := 1
					for k < 3 && i+1 < len(b) && '0' <= b[i+1] && b[i+1] <= '7' {
						num = (num * 8) + int(b[i+1]-'0')
						i++
						k++
					}
					if 0 <= num && num < 256 {
						s = append(s, byte(num))
					}
				}
				j = i + 1 // +1 for backslash
			}
			i++
		}
		if i == len(b) || b[i] != ')' {
			return nil, 0, fmt.Errorf("bad string")
		}
		s = append(s, b[j:i]...)
		i++
		if r != nil && r.encrypt.isEncrypted && ref != noEncryptRef {
			s = r.encrypt.Decrypt(ref, s)
		}
		return s, i, nil
	} else if b[0] == '<' {
		i := 1
		for i < len(b) && ('0' <= b[i] && b[i] <= '9' || 'a' <= b[i] && b[i] <= 'f' || 'A' <= b[i] && b[i] <= 'F') {
			i++
		}
		if i == len(b) || b[i] != '>' {
			return nil, 0, fmt.Errorf("bad string")
		}
		s := b[1:i:i]
		i++
		if r != nil && r.encrypt.isEncrypted && ref != noEncryptRef {
			s = r.encrypt.Decrypt(ref, s)
		}
		if len(s)%2 == 1 {
			s = append(s, '0') // allocates new slice
		}
		var err error
		s, err = hex.DecodeString(string(s))
		return s, i, err
	} else if 3 < len(b) && b[0] == 't' && b[1] == 'r' && b[2] == 'u' && b[3] == 'e' {
		return true, 4, nil
	} else if 4 < len(b) && b[0] == 'f' && b[1] == 'a' && b[2] == 'l' && b[3] == 's' && b[4] == 'e' {
		return false, 5, nil
	} else if 3 < len(b) && b[0] == 'n' && b[1] == 'u' && b[2] == 'l' && b[3] == 'l' {
		return nil, 4, nil
	}
	return nil, 0, fmt.Errorf("bad value")
}

type pdfStreamReader struct {
	b []byte
	i int
}

func newPDFStreamReader(b []byte) *pdfStreamReader {
	return &pdfStreamReader{b, 0}
}

func (r *pdfStreamReader) Pos() int {
	return r.i
}

func (r *pdfStreamReader) Next() (string, []interface{}, error) {
	var vals []interface{}
	r.i = moveWhiteSpace(r.b, r.i)
	for r.i < len(r.b) {
		if 'a' <= r.b[r.i] && r.b[r.i] <= 'z' || 'A' <= r.b[r.i] && r.b[r.i] <= 'Z' || r.b[r.i] == '\'' || r.b[r.i] == '"' {
			name, n, err := parseName(r.b[r.i:])
			if err != nil {
				return "", nil, err
			}
			r.i += n
			return string(name), vals, nil
		}

		val, n, err := pdfReadContentVal(r.b[r.i:])
		if err != nil {
			return "", nil, fmt.Errorf("invalid stream: %w", err)
		}
		vals = append(vals, val)
		r.i = moveWhiteSpace(r.b, r.i+n)
	}
	return "", nil, io.EOF
}
