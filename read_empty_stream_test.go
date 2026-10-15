package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// emptyStreamPDF builds a one-page PDF whose /Contents is an array of two
// streams: a real, unfiltered text stream, then a zero-length stream that
// declares /FlateDecode. MYOB PDF Manager 1.5 writes the second stream on
// every page.
func emptyStreamPDF() []byte {
	content := "BT /F1 12 Tf 72 720 Td (an empty stream does not hide this line) Tj ET"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 6 0 R >> >> /Contents [4 0 R 5 0 R] >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Length 0 /Filter /FlateDecode >>\nstream\n\nendstream",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return b.Bytes()
}

func TestAZeroLengthFlateStreamReadsAsEmpty(t *testing.T) {
	data := emptyStreamPDF()
	r, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	text, err := r.Page(1).GetPlainText(nil)

	if err != nil {
		t.Fatalf("page text: %v", err)
	}
	if !strings.Contains(text, "an empty stream does not hide this line") {
		t.Fatalf("page text %q does not carry the line", text)
	}
}
