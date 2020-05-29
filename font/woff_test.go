package font

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/tdewolff/test"
)

func TestWOFFError(t *testing.T) {
	var tts = []struct {
		data string
		err  string
	}{
		{"wOFF00000000\x00\x01\x00\x0000000000000000000000i00000000000\xff\xff\xff\xfc\x00\x00\x0000000000000000000", "length in header must match file size"},
		{"wOFF\x01bwOFF u\x00\x01\x00\x00de\x80\x00orma\x10\x00wOFF\x01b dunicF u\x00r\xbd\xbf\xef^\x00\x00\x00\x00 \x00\x00\x00 :pur  oes ?ite:\t", "length in header must match file size"},
	}
	for i, tt := range tts {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			_, err := ParseWOFF([]byte(tt.data))
			test.T(t, err.Error(), tt.err)
		})
	}
}

func TestWOFFValidationFormat(t *testing.T) {
	var tts = []struct {
		filename string
		err      string
	}{
		{"valid-001", ""},
		{"valid-002", ""},
		{"valid-003", ""},
		{"valid-004", ""},
		{"valid-005", ""},
		{"valid-006", ""},
		{"valid-007", ""},
		{"valid-008", ""},
		{"header-signature-001", "bad signature"},
		//{"header-flavor-001", "err"},
		//{"header-flavor-002", "err"},
		{"header-length-001", "length in header must match file size"},
		{"header-length-002", "length in header must match file size"},
		{"header-numTables-001", "numTables in header must not be zero"},
		{"header-reserved-001", "reserved in header must be zero"},
		{"header-totalSfntSize-001", "totalSfntSize is incorrect"},
		{"header-totalSfntSize-002", "totalSfntSize is incorrect"},
		{"header-totalSfntSize-003", "totalSfntSize is incorrect"},
		//{"blocks-extraneous-data-001", "err"}, // bad test?
		//{"blocks-extraneous-data-002", "err"}, // bad test?
		//{"blocks-extraneous-data-003", "err"},
		//{"blocks-extraneous-data-004", "err"},
		//{"blocks-extraneous-data-005", "err"},
		//{"blocks-extraneous-data-006", "err"},
		//{"blocks-extraneous-data-007", "err"},
		//{"blocks-ordering-001", "metadata must follow table data block"},
		//{"blocks-ordering-002", "private data without metadata must follow table data block"},
		//{"blocks-ordering-003", "metadata must follow table data block"},
		//{"blocks-ordering-004", "metadata must follow table data block"},
		//{"blocks-overlap-001", "tables can not overlap"},
		//{"blocks-overlap-002", "tables can not overlap"},
		//{"blocks-overlap-003", "tables can not overlap"},
		{"directory-4-byte-001", "totalSfntSize is incorrect"},
		//{"directory-4-byte-002", "err"}, // bad test?
		//{"directory-4-byte-003", "err"}, // we do not test this
		{"directory-ascending-001", "tables are not sorted alphabetically"},
		{"directory-compLength-001", "compressed table size is larger than decompressed size"},
		//{"directory-extraneous-data-001", "err"}, // bad test?
		{"directory-origCheckSum-001", "CFF : bad checksum"},
		//{"directory-origCheckSum-002", "err"}, // I don't understand why this should be an error
		{"directory-origLength-001", "decompressed table length must be equal to origLength"},
		{"directory-origLength-002", "decompressed table length must be equal to origLength"},
		{"directory-overlaps-001", "table extends beyond file size"},
		{"directory-overlaps-002", "table extends beyond file size"},
		//{"directory-overlaps-003", "tables can not overlap"},
		//{"directory-overlaps-004", "tables can not overlap"},
		{"directory-overlaps-005", "tables can not overlap"},
		{"tabledata-compression-001", ""},
		{"tabledata-compression-002", ""},
		{"tabledata-compression-003", ""},
		{"tabledata-compression-004", ""},
		{"tabledata-zlib-001", "name: zlib: invalid header"},
	}
	for _, tt := range tts {
		t.Run(tt.filename, func(t *testing.T) {
			b, err := ioutil.ReadFile("testdata/woff_format/" + tt.filename + ".woff")
			test.Error(t, err)
			_, err = ParseWOFF(b)
			if tt.err == "" {
				test.Error(t, err)
			} else if err == nil {
				test.Fail(t, "must give error")
			} else {
				test.T(t, err.Error(), tt.err)
			}
		})
	}
}
