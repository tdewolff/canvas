package font

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/tdewolff/test"
)

func TestWOFF2Error(t *testing.T) {
	var tts = []struct {
		data string
		err  string
	}{
		{"wOF200000000\x00\x00000000\xff\xff\xff\xff000000000000000000000000", "length in header must match file size"},
		{"wOF200000000\x00\x00000000\x00\x00\x00\b00000000000000000000000030000000", "length in header must match file size"},
		{"wOF200000000\x00\x01000000\x00\x00\x000000000000000000000000000Y\xbf\x00\x00Z\x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "length in header must match file size"},
		{"wOF2OTTO\x00\x00\x03\xd4\x00\t\x00\x00\xff\x01\a@" +
			"\x00\x00\x03\x8d\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
			"\x00\x00\x00\x00\x00\x00\x00\x00\x18\x84.\x06`\x00\x81f\x016\x02$" +
			"\x03\x10\x04\x06\x1f\x85\x17\a \x1b\x9a\x06Q\xd4\xc4\xc0\t\x00\xfcL" +
			"\xb0\xc1=tE8\x8e2X\x96\x9aQl\x87Il\xc2\r\xeb\xb1" +
			"\xe6\x89\xfc\x12\x0f_K\xf7\xfe\xee\x1d\xad5\x8dlՁ*\xd5\t" +
			"T4\x83\xd08d\xc6G\xb9\xa8(\xd0l\xaci\x8a\x96\xcf\xc7\x16" +
			"Y\xb6\x9f\"I~F\xe1g\xd4\xe6\xdc\xee\xfd\xaeb\xb2b\xcfB" +
			"Ɓ\xfe\xa2U\xe9W\xb3\x87\x16^\xfa4\"\x98\xf8M\\s\x99" +
			"\x04_ \x03\x8e2\x8d\x0e\xdc\xcd\"\xca\x13l\xd3\x015\xe0l\xad" +
			"\x89c\x98\x89l\x18\xbb\xe8\x01\x1e\x16\x8aBD\f\xdc\xee\x89.\xac" +
			"\xed\x94\xc1aa\xb670\x92\x17\xbb\x9b!S\x14a\xdabLG" +
			"\x9c\xe9J|\x9f\xa1+\xa2e%?D@Jq\tۧ\x02\x8a" +
			"ʄD\xc4\xd0|\xa9\x16z\xa4:\xa4tl,\xa0\xbc\aѭ" +
			"\x00\"\b\xcbZ\x92IH\xb2\xe7\xf6\xafPP0\xe5\xb7\b\xb6\x83" +
			",\x81|\xba\x03\x93\x85S\x14+\xf9\x04\x82G(iz\xa5\xb0Q" +
			"\x00\x830\x01\x93\xe0\x86N\x9a\x9d\xbf\x1b<\x84k\xa9\x0eu*\xbe" +
			"\x8ac\xcb!\x02w\xb1\x958X>酖\x136>!\xfb\x8f" +
			"\xaa\xc20\xcc\xe0\xcec\aA9\xaa`\xc0\xf4\xe0\x1c\a\xb6/\x9f" +
			"\x82$H\xba>E[}<\xd4\xf3n\xdaɡ\a\x8e.7\x83" +
			"\xfd\x94\x97\xc3\xfd\\\x0fl]\x1e!\x89\xf3\xf90ҟ\xc6z\xe1" +
			"[\x12f\xe5\xd6'h\x8f\xf5\xc0\x92\xe5\x8d4B\ruk\x93\xfe" +
			">\x90\"\xd0C\x1fb#a\x9f\xf9h\xae~`l\xb4\xb75\u007f" +
			"\x97\xaf6*9\x04\x19\xda\x1c\x8f\x06\xf38}PIߖ\xef\xa3" +
			"啒\xe9e\xe2v\x13|\xba\xf4zp\xder\xban\xce\xd2\xe5" +
			"\xab\x0e\xec(\xfb\xaa\r\x82\xf7OA\xeaY\xf5U\f\xbc\xa3\x8fr" +
			"G\xab\xf6\xddֺu\xec\xc8ӷW\xecQv\xa9\xderh\xa3" +
			"\xff\u007fa1\x87\xb2Y\x1bkAO&\xf2\x94~>\x12\x83\xb2\xfa" +
			"p\xe5\x04\a\"x\x96`#\xd5!3Y~[\x1c\xab0c\x06" +
			"\xe5\x87\xd4t\x1d\x00\xd0\xfa>\b\xc9\xf2/*J\xd2G\xba\x03\xce" +
			"\xd3\xcbĽv\x1b\x02\xee ?g\xef+c'\xcc8ܷQ" +
			"r\x9d\x19\x10\x10\xbf\xa9\x81\x19\x80\x8c\x8c\t1p!!0!i" +
			"\ah\xd1\"a\x069lV\n9^\xee\xc8v\v\xb5M>\xd1" +
			"\xea$\xff\xc8\x01J>`\x05$\xcanzLp\xf0O\x82G\xc6" +
			"\xe7\xaf%\xe7:\xd4t5\xfe\x005\xc3ѳ/h\xf9J\rd" +
			"\t\x04\xfd\xcf\xd7ڪ\fҷ9\xd2\x17\xd5\xc8cyY\xf6\x87" +
			"\xf5\xd9{\x03\nt\xf9\x16<\xd4I6\x904\xeb\x04\xb2)\xc5\xfe" +
			"\xa2<\xa0t\xa2\r4\xe9\x17%Z\f\xf9\xe9\x1ev\xd1\x1a\x1a\xf3" +
			"\x9b\r$]\xaa@\xb6\xef\x1f(,\xd5\f\x94*m\xf67\x15[" +
			"l\xb1Quơ\x9a\x99\xe8[/\xa3\xa18\x1d\xb5ռJ\xb5" +
			"M\xa5,3y8c\xb3\x999+\xe8\xce{T\xc8\v\xc6\xeb\xbd" +
			"\xc1d\x9e)͋\xe0\xea\x8a\xf8nϷ[\xc3Q\xef\xbfȶ" +
			"\xaa}\xf4ξ\x89\xb8\x1c\xfb\x99\xea\x8ay\xfc\x87\xe1n\x8f\x10t" +
			"\xc1\xb29\xd0\x14!',\x9eK\xf1bi\xa9kjj\x9a\xd8\xd1" +
			"iX\x93:\xa8\x98\x8a\r\xa9\xc1n\x94Yzò\xd8\xc4&\f" +
			"bS\x13\ag\x98\xddX$U\xa0s|\xaa\xb1x\\\x1a\xfbz" +
			"\x066U\xf5\"c\xcb\xdbǩS\x88\xde\xda\xe0\x87kk\x19\x1a" +
			"\xe9A3\x8c\x1b\xc6\xc6\xde\xc1\x16c\xe7\xe2\x8e\xf5Ft\x1a\x87\r" +
			"\xdd\xe9,2\x91\x86\x87\xe6,:\x97!\x01\xfc\xff\xd8\x13\x00\x00\x00", "memory limit exceded"},
	}
	for i, tt := range tts {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			_, err := ParseWOFF2([]byte(tt.data))
			if err != nil {
				test.T(t, err.Error(), tt.err)
			} else {
				test.T(t, "", tt.err)
			}
		})
	}
}

func TestWOFF2ValidationDecoder(t *testing.T) {
	filenames := []string{
		"validation-checksum-001",
		"validation-checksum-002",
		"validation-loca-format-001",
		"validation-loca-format-002",
	}
	for i := 1; i < 156; i++ {
		filenames = append(filenames, fmt.Sprintf("validation-off-%03d", i))
	}
	for _, filename := range filenames {
		t.Run(filename, func(t *testing.T) {
			b, err := ioutil.ReadFile("./testdata/woff2_decoder/" + filename + ".woff2")
			test.Error(t, err)
			_, err = ParseWOFF2(b)
			test.Error(t, err)
		})
	}
}

func TestWOFF2ValidationDecoderRoundtrip(t *testing.T) {
	filenames := []string{
		//"roundtrip-collection-dsig-001",
		//"roundtrip-collection-order-001",
		//"roundtrip-hmtx-lsb-001", // the woff2 test file seems to be broken, advanceWidth is in reverse order
		//"roundtrip-offset-tables-001",
	}
	for _, filename := range filenames {
		t.Run(filename, func(t *testing.T) {
			a, err := ioutil.ReadFile("./testdata/woff2_decoder/" + filename + ".ttf")
			test.Error(t, err)
			b, err := ioutil.ReadFile("./testdata/woff2_decoder/" + filename + ".woff2")
			test.Error(t, err)
			b, err = ParseWOFF2(b)
			test.Error(t, err)
			if !bytes.Equal(a, b) {
				test.Fail(t, "decoded WOFF2 unequal to TTF")
			}
		})
	}
}

func TestWOFF2ValidationFormat(t *testing.T) {
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
		//{"blocks-extraneous-data-001", "err"},
		//{"blocks-extraneous-data-002", "err"},
		//{"blocks-extraneous-data-003", "err"},
		//{"blocks-extraneous-data-004", "err"},
		//{"blocks-extraneous-data-005", "err"},
		//{"blocks-extraneous-data-006", "err"},
		//{"blocks-extraneous-data-007", "err"},
		//{"blocks-metadata-absent-002", "err"},
		//{"blocks-metadata-padding-001", "err"},
		//{"blocks-metadata-padding-002", "err"},
		//{"blocks-metadata-padding-003", "err"},
		//{"blocks-metadata-padding-004", "err"},
		//{"blocks-ordering-003", "err"},
		//{"blocks-ordering-004", "err"},
		//{"blocks-private-001", "err"},
		//{"blocks-private-002", "err"},
		{"directory-table-order-001", ""},
		{"directory-table-order-002", "loca: must come after glyf table"},
		{"tabledata-extraneous-data-001", "sum of table lengths must match decompressed font data size"},
		{"tabledata-brotli-001", "brotli: corrupted input"},
		{"tabledata-decompressed-length-001", "sum of table lengths must match decompressed font data size"},
		{"tabledata-decompressed-length-002", "sum of table lengths must match decompressed font data size"},
		{"tabledata-decompressed-length-003", "sum of table lengths must match decompressed font data size"},
		{"tabledata-decompressed-length-004", "sum of table lengths must match decompressed font data size"},
		{"tabledata-transform-length-001", "loca: transformLength must be zero"},
		{"tabledata-transform-length-002", "glyf: table defined more than once"}, // not the right error, but impossible to know if transformLength is set or not
		{"tabledata-loca-size-001", ""},
		{"tabledata-loca-size-002", ""},
		{"tabledata-loca-size-003", ""},
		{"tabledata-hmtx-transform-001", ""},
		{"tabledata-hmtx-transform-002", "hmtx: must reconstruct at least one left side bearing array"},
		{"tabledata-hmtx-transform-003", "hmtx: reserved bits in flags must not be set"},
		{"tabledata-transform-glyf-loca-001", "glyf and loca tables must be both present and either be both transformed or untransformed"},
		{"tabledata-transform-glyf-loca-002", "glyf and loca tables must be both present and either be both transformed or untransformed"},
		{"tabledata-glyf-composite-bbox-001", ""},
		//{"metadata-padding-001", "err"},
		//{"metadata-compression-001", "err"},
		//{"metadata-compression-002", "err"},
		//{"metadata-metaOrigLength-001", "err"},
		//{"metadata-metaOrigLength-002", "err"},
		//{"metadata-well-formed-001", "err"},
		//{"metadata-well-formed-002", "err"},
		//{"metadata-well-formed-003", "err"},
		//{"metadata-well-formed-004", "err"},
		//{"metadata-well-formed-005", "err"},
		//{"metadata-well-formed-006", "err"},
		//{"metadata-well-formed-007", "err"},
		//{"metadata-encoding-001", ""},
		//{"metadata-encoding-002", "err"},
		//{"metadata-encoding-003", "err"},
		//{"metadata-encoding-004", ""},
		//{"metadata-encoding-005", ""},
		//{"metadata-encoding-006", "err"},
		//{"metadata-schema-metadata-001", ""},
		//{"metadata-schema-metadata-002", "err"},
		//{"metadata-schema-metadata-003", "err"},
		//{"metadata-schema-metadata-004", "err"},
		//{"metadata-schema-metadata-005", "err"},
		//{"metadata-schema-metadata-006", "err"},
		//{"metadata-schema-uniqueid-001", ""},
		//{"metadata-schema-uniqueid-002", ""},
		//{"metadata-schema-uniqueid-003", "err"},
		//{"metadata-schema-uniqueid-004", "err"},
		//{"metadata-schema-uniqueid-005", "err"},
		//{"metadata-schema-uniqueid-006", "err"},
		//{"metadata-schema-uniqueid-007", "err"},
		//{"metadata-schema-vendor-001", ""},
		//{"metadata-schema-vendor-002", ""},
		//{"metadata-schema-vendor-003", ""},
		//{"metadata-schema-vendor-004", "err"},
		//{"metadata-schema-vendor-005", "err"},
		//{"metadata-schema-vendor-006", ""},
		//{"metadata-schema-vendor-007", ""},
		//{"metadata-schema-vendor-008", "err"},
		//{"metadata-schema-vendor-009", ""},
		//{"metadata-schema-vendor-010", "err"},
		//{"metadata-schema-vendor-011", "err"},
		//{"metadata-schema-vendor-012", "err"},
		//{"metadata-schema-credits-001", ""},
		//{"metadata-schema-credits-002", ""},
		//{"metadata-schema-credits-003", "err"},
		//{"metadata-schema-credits-004", "err"},
		//{"metadata-schema-credits-005", "err"},
		//{"metadata-schema-credits-006", "err"},
		//{"metadata-schema-credits-007", "err"},
		//{"metadata-schema-credit-001", ""},
		//{"metadata-schema-credit-002", ""},
		//{"metadata-schema-credit-003", ""},
		//{"metadata-schema-credit-004", "err"},
		//{"metadata-schema-credit-005", ""},
		//{"metadata-schema-credit-006", ""},
		//{"metadata-schema-credit-007", "err"},
		//{"metadata-schema-credit-008", ""},
		//{"metadata-schema-credit-009", "err"},
		//{"metadata-schema-credit-010", "err"},
		//{"metadata-schema-credit-011", "err"},
		//{"metadata-schema-description-001", ""},
		//{"metadata-schema-description-002", ""},
		//{"metadata-schema-description-003", ""},
		//{"metadata-schema-description-004", ""},
		//{"metadata-schema-description-005", ""},
		//{"metadata-schema-description-006", ""},
		//{"metadata-schema-description-007", ""},
		//{"metadata-schema-description-008", "err"},
		//{"metadata-schema-description-009", "err"},
		//{"metadata-schema-description-010", "err"},
		//{"metadata-schema-description-011", "err"},
		//{"metadata-schema-description-012", "err"},
		//{"metadata-schema-description-013", ""},
		//{"metadata-schema-description-014", ""},
		//{"metadata-schema-description-015", "err"},
		//{"metadata-schema-description-016", ""},
		//{"metadata-schema-description-017", "err"},
		//{"metadata-schema-description-018", "err"},
		//{"metadata-schema-description-019", ""},
		//{"metadata-schema-description-020", ""},
		//{"metadata-schema-description-021", ""},
		//{"metadata-schema-description-022", ""},
		//{"metadata-schema-description-023", ""},
		//{"metadata-schema-description-024", "err"},
		//{"metadata-schema-description-025", ""},
		//{"metadata-schema-description-026", ""},
		//{"metadata-schema-description-027", ""},
		//{"metadata-schema-description-028", ""},
		//{"metadata-schema-description-029", ""},
		//{"metadata-schema-description-030", ""},
		//{"metadata-schema-description-031", "err"},
		//{"metadata-schema-description-032", ""},
		//{"metadata-schema-license-001", ""},
		//{"metadata-schema-license-002", ""},
		//{"metadata-schema-license-003", ""},
		//{"metadata-schema-license-004", ""},
		//{"metadata-schema-license-005", ""},
		//{"metadata-schema-license-006", ""},
		//{"metadata-schema-license-007", ""},
		//{"metadata-schema-license-008", ""},
		//{"metadata-schema-license-009", "err"},
		//{"metadata-schema-license-010", ""},
		//{"metadata-schema-license-011", "err"},
		//{"metadata-schema-license-012", "err"},
		//{"metadata-schema-license-013", "err"},
		//{"metadata-schema-license-014", ""},
		//{"metadata-schema-license-015", ""},
		//{"metadata-schema-license-016", "err"},
		//{"metadata-schema-license-017", ""},
		//{"metadata-schema-license-018", "err"},
		//{"metadata-schema-license-019", "err"},
		//{"metadata-schema-license-020", ""},
		//{"metadata-schema-license-021", ""},
		//{"metadata-schema-license-022", ""},
		//{"metadata-schema-license-023", ""},
		//{"metadata-schema-license-024", ""},
		//{"metadata-schema-license-025", "err"},
		//{"metadata-schema-license-026", ""},
		//{"metadata-schema-license-027", ""},
		//{"metadata-schema-license-028", ""},
		//{"metadata-schema-license-029", ""},
		//{"metadata-schema-license-030", ""},
		//{"metadata-schema-license-031", ""},
		//{"metadata-schema-license-032", "err"},
		//{"metadata-schema-license-033", ""},
		//{"metadata-schema-copyright-001", ""},
		//{"metadata-schema-copyright-002", ""},
		//{"metadata-schema-copyright-003", ""},
		//{"metadata-schema-copyright-004", ""},
		//{"metadata-schema-copyright-005", ""},
		//{"metadata-schema-copyright-006", "err"},
		//{"metadata-schema-copyright-007", "err"},
		//{"metadata-schema-copyright-008", "err"},
		//{"metadata-schema-copyright-009", "err"},
		//{"metadata-schema-copyright-010", "err"},
		//{"metadata-schema-copyright-011", ""},
		//{"metadata-schema-copyright-012", ""},
		//{"metadata-schema-copyright-013", "err"},
		//{"metadata-schema-copyright-014", ""},
		//{"metadata-schema-copyright-015", "err"},
		//{"metadata-schema-copyright-016", "err"},
		//{"metadata-schema-copyright-017", ""},
		//{"metadata-schema-copyright-018", ""},
		//{"metadata-schema-copyright-019", ""},
		//{"metadata-schema-copyright-020", ""},
		//{"metadata-schema-copyright-021", ""},
		//{"metadata-schema-copyright-022", "err"},
		//{"metadata-schema-copyright-023", ""},
		//{"metadata-schema-copyright-024", ""},
		//{"metadata-schema-copyright-025", ""},
		//{"metadata-schema-copyright-026", ""},
		//{"metadata-schema-copyright-027", ""},
		//{"metadata-schema-copyright-028", ""},
		//{"metadata-schema-copyright-029", "err"},
		//{"metadata-schema-copyright-030", ""},
		//{"metadata-schema-trademark-001", ""},
		//{"metadata-schema-trademark-002", ""},
		//{"metadata-schema-trademark-003", ""},
		//{"metadata-schema-trademark-004", ""},
		//{"metadata-schema-trademark-005", ""},
		//{"metadata-schema-trademark-006", "err"},
		//{"metadata-schema-trademark-007", "err"},
		//{"metadata-schema-trademark-008", "err"},
		//{"metadata-schema-trademark-009", "err"},
		//{"metadata-schema-trademark-010", "err"},
		//{"metadata-schema-trademark-011", ""},
		//{"metadata-schema-trademark-012", ""},
		//{"metadata-schema-trademark-013", "err"},
		//{"metadata-schema-trademark-014", ""},
		//{"metadata-schema-trademark-015", "err"},
		//{"metadata-schema-trademark-016", "err"},
		//{"metadata-schema-trademark-017", ""},
		//{"metadata-schema-trademark-018", ""},
		//{"metadata-schema-trademark-019", ""},
		//{"metadata-schema-trademark-020", ""},
		//{"metadata-schema-trademark-021", ""},
		//{"metadata-schema-trademark-022", "err"},
		//{"metadata-schema-trademark-023", ""},
		//{"metadata-schema-trademark-024", ""},
		//{"metadata-schema-trademark-025", ""},
		//{"metadata-schema-trademark-026", ""},
		//{"metadata-schema-trademark-027", ""},
		//{"metadata-schema-trademark-028", ""},
		//{"metadata-schema-trademark-029", "err"},
		//{"metadata-schema-trademark-030", ""},
		//{"metadata-schema-licensee-001", ""},
		//{"metadata-schema-licensee-002", "err"},
		//{"metadata-schema-licensee-003", "err"},
		//{"metadata-schema-licensee-004", ""},
		//{"metadata-schema-licensee-005", ""},
		//{"metadata-schema-licensee-006", "err"},
		//{"metadata-schema-licensee-007", ""},
		//{"metadata-schema-licensee-008", "err"},
		//{"metadata-schema-licensee-009", "err"},
		//{"metadata-schema-licensee-010", "err"},
		//{"metadata-schema-extension-001", ""},
		//{"metadata-schema-extension-002", ""},
		//{"metadata-schema-extension-003", ""},
		//{"metadata-schema-extension-004", ""},
		//{"metadata-schema-extension-005", ""},
		//{"metadata-schema-extension-006", ""},
		//{"metadata-schema-extension-007", ""},
		//{"metadata-schema-extension-008", "err"},
		//{"metadata-schema-extension-009", "err"},
		//{"metadata-schema-extension-010", "err"},
		//{"metadata-schema-extension-011", "err"},
		//{"metadata-schema-extension-012", ""},
		//{"metadata-schema-extension-013", ""},
		//{"metadata-schema-extension-014", ""},
		//{"metadata-schema-extension-015", ""},
		//{"metadata-schema-extension-016", ""},
		//{"metadata-schema-extension-017", "err"},
		//{"metadata-schema-extension-018", ""},
		//{"metadata-schema-extension-019", "err"},
		//{"metadata-schema-extension-020", "err"},
		//{"metadata-schema-extension-021", ""},
		//{"metadata-schema-extension-022", ""},
		//{"metadata-schema-extension-023", ""},
		//{"metadata-schema-extension-024", ""},
		//{"metadata-schema-extension-025", ""},
		//{"metadata-schema-extension-026", ""},
		//{"metadata-schema-extension-027", ""},
		//{"metadata-schema-extension-028", "err"},
		//{"metadata-schema-extension-029", "err"},
		//{"metadata-schema-extension-030", "err"},
		//{"metadata-schema-extension-031", "err"},
		//{"metadata-schema-extension-032", "err"},
		//{"metadata-schema-extension-033", ""},
		//{"metadata-schema-extension-034", ""},
		//{"metadata-schema-extension-035", ""},
		//{"metadata-schema-extension-036", ""},
		//{"metadata-schema-extension-037", ""},
		//{"metadata-schema-extension-038", "err"},
		//{"metadata-schema-extension-039", ""},
		//{"metadata-schema-extension-040", "err"},
		//{"metadata-schema-extension-041", "err"},
		//{"metadata-schema-extension-042", ""},
		//{"metadata-schema-extension-043", ""},
		//{"metadata-schema-extension-044", ""},
		//{"metadata-schema-extension-045", ""},
		//{"metadata-schema-extension-046", ""},
		//{"metadata-schema-extension-047", "err"},
		//{"metadata-schema-extension-048", ""},
		//{"metadata-schema-extension-049", "err"},
		//{"metadata-schema-extension-050", "err"},
	}
	for _, tt := range tts {
		t.Run(tt.filename, func(t *testing.T) {
			b, err := ioutil.ReadFile("./testdata/woff2_format/" + tt.filename + ".woff2")
			test.Error(t, err)
			_, err = ParseWOFF2(b)
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

func TestWOFF2ValidationUserAgent(t *testing.T) {
	var tts = []struct {
		filename string
		err      string
	}{
		//{"available-002", ""},
		//{"blocks-extraneous-data-001", "err"}, // not sure how to test this
		//{"blocks-overlap-001", ""},
		//{"blocks-overlap-002", ""},
		//{"blocks-overlap-003", ""},
		//{"datatypes-alt-255uint16-001", ""}, // bad test, has wrong length of hmtx table
		{"datatypes-invalid-base128-001", "readUintBase128: must not start with leading zeros"},
		{"datatypes-invalid-base128-002", "readUintBase128: overflow"},
		{"datatypes-invalid-base128-003", "readUintBase128: exceeds 5 bytes"},
		{"directory-knowntags-001", ""},
		//{"directory-mismatched-tables-001", "err"},
		{"header-totalsfntsize-001", ""},
		{"header-totalsfntsize-002", ""},
		{"tabledata-bad-origlength-loca-001", "loca: origLength must match numGlyphs+1 entries"},
		{"tabledata-bad-origlength-loca-002", "loca: origLength must match numGlyphs+1 entries"},
		{"tabledata-glyf-bbox-002", "glyf: composite glyph must have bbox definition"},
		{"tabledata-glyf-bbox-003", "glyf: empty glyph cannot have bbox definition"},
		{"tabledata-glyf-origlength-001", ""},
		{"tabledata-glyf-origlength-002", ""},
		{"tabledata-glyf-origlength-003", ""},
		{"tabledata-non-zero-loca-001", "loca: transformLength must be zero"},
		//{"tabledata-transform-bad-flag-001", "head: invalid transformation"}, // TODO: test fixed with CFF support
		//{"tabledata-transform-bad-flag-002", "glyf: invalid transformation"}, // TODO: test fixed with CFF support
	}
	for _, tt := range tts {
		t.Run(tt.filename, func(t *testing.T) {
			b, err := ioutil.ReadFile("./testdata/woff2_useragent/" + tt.filename + ".woff2")
			test.Error(t, err)
			_, err = ParseWOFF2(b)
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
