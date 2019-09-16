package canvas

import (
	"strings"
	"testing"

	"github.com/tdewolff/test"
)

func TestPDF(t *testing.T) {
	c := New(10, 10)
	c.DrawPath(0, 0, MustParseSVG("L10 0"))

	PDFCompress = false
	sb := strings.Builder{}
	c.WritePDF(&sb)
	test.T(t, sb.String(), `%PDF-1.7
1 0 obj
<< /Length 38 >> stream
0.00000 0.00000 m 10.00000 0.00000 l f
endstream
endobj
2 0 obj
<< /Type /Page /Contents 1 0 R /Group << /Type /Group /CS /DeviceRGB /I true /S /Transparency >> /MediaBox [0.00000 0.00000 10.00000 10.00000] /Parent 2 0 R /Resources << >> >>
endobj
3 0 obj
<< /Type /Pages /Count 1 /Kids [2 0 R] >>
endobj
4 0 obj
<< /Type /Catalog /Pages 3 0 R >>
endobj
xref
0 5
0000000000 65535 f
0000000009 00000 n
0000000097 00000 n
0000000289 00000 n
0000000346 00000 n
trailer
<< /Root 4 0 R /Size 4 >>
starxref
395
%%EOF`)
}
