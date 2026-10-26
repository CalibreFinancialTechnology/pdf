package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// headerLinePDF is a one-page text PDF whose first line is header, with every
// cross-reference offset counted after it.
func headerLinePDF(header string) []byte {
	content := "BT /F1 12 Tf 72 720 Td (text after the version does not hide this line) Tj ET"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var b bytes.Buffer
	b.WriteString(header)
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

// 137 of the 161 files publib's library still refused on 2026-10-27 carry a
// space, or a space and a scanner's name, between the version and the line end.
func TestTextAfterAKnownHeaderVersionIsRead(t *testing.T) {
	for _, header := range []string{"%PDF-1.1 \n", "%PDF-1.7 \r\n", "%PDF-1.4\t\n", "%PDF-1.4 Sharp Scanned ImagePDF\n", "%PDF-2.0 x\n"} {
		data := headerLinePDF(header)

		r, err := NewReader(bytes.NewReader(data), int64(len(data)))

		if err != nil {
			t.Fatalf("%q: open: %v", header, err)
		}
		text, err := r.Page(1).GetPlainText(nil)
		if err != nil || !strings.Contains(text, "text after the version does not hide this line") {
			t.Fatalf("%q: page text %q, err %v", header, text, err)
		}
	}
}

func TestAVersionThatContinuesPastAKnownOneStaysRefused(t *testing.T) {
	for _, header := range []string{"%PDF-1.70\n", "%PDF-1.7.1\n", "%PDF-2.7\n", "%PDF-1.8\n", "%PDF-1.4x\n"} {
		data := headerLinePDF(header)

		_, err := NewReader(bytes.NewReader(data), int64(len(data)))

		if err == nil || err.Error() != "not a PDF file: invalid header" {
			t.Fatalf("%q: err %v, want the invalid-header refusal", header, err)
		}
	}
}
