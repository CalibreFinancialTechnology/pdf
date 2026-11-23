package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

// arrayEOFPDF builds a one-page PDF whose content stream ends inside a TJ
// array: the closing bracket and the operator never come. Real files in the
// library carry this; the array loop used to append io.EOF until memory ran out.
func arrayEOFPDF() []byte {
	content := "BT /F1 12 Tf 72 720 Td (the line before the cut array is read) Tj [(cut) -250 (short"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
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

func TestAContentStreamEndingInsideAnArrayIsRead(t *testing.T) {
	data := arrayEOFPDF()
	r, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	type result struct {
		text string
		err  error
	}
	done := make(chan result, 1)

	go func() {
		text, err := r.Page(1).GetPlainText(nil)
		done <- result{text, err}
	}()

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("page text: %v", got.err)
		}
		if !strings.Contains(got.text, "the line before the cut array is read") {
			t.Fatalf("page text %q does not carry the line", got.text)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reading a content stream that ends inside an array did not return within 5 s")
	}
}
