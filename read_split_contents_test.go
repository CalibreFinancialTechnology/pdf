package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

// splitContentsPDF builds a one-page PDF whose /Contents is two streams, split
// inside a TJ array: the first ends with "[(split", the second begins with
// "-250 (across)] TJ". A page's content streams are one stream cut at token
// boundaries (ISO 32000-1, 7.8.2), and Microsoft Word's PDF export cuts them
// like this.
func splitContentsPDF() []byte {
	return twoPieceContentsPDF("BT /F1 12 Tf 72 720 Td [(split)", "-250 (across)] TJ ET")
}

// twoPieceContentsPDF builds a one-page PDF whose /Contents is the two given
// streams, in order.
func twoPieceContentsPDF(first, second string) []byte {
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 6 0 R >> >> /Contents [4 0 R 5 0 R] >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(first), first),
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(second), second),
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

func TestAnArraySplitAcrossTwoContentStreamsIsRead(t *testing.T) {
	data := splitContentsPDF()
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
		if !strings.Contains(got.text, "split") || !strings.Contains(got.text, "across") {
			t.Fatalf("page text %q does not carry both halves of the array", got.text)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reading an array split across two content streams did not return within 5 s")
	}
}

// A cut may fall between two keywords with no space either side: "Tj" ends one
// piece and "ET" begins the next. Joined without a separator they read as one
// unknown keyword "TjET" and the first piece's text is never shown.
func TestTwoKeywordsEitherSideOfACutStayTwoKeywords(t *testing.T) {
	data := twoPieceContentsPDF("BT /F1 12 Tf 72 720 Td (the first piece is shown) Tj", "ET BT /F1 12 Tf 72 700 Td (the second piece is shown) Tj ET")
	r, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	text, err := r.Page(1).GetPlainText(nil)

	if err != nil {
		t.Fatalf("page text: %v", err)
	}
	if !strings.Contains(text, "the first piece is shown") || !strings.Contains(text, "the second piece is shown") {
		t.Fatalf("page text %q does not carry both pieces", text)
	}
}
