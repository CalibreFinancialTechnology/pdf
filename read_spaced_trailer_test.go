package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// spacedTrailerPDF builds a one-page PDF whose last lines are written as
// trailer, each keyword line closed by pad and a newline. The Commonwealth
// Bank's Medallion Trust reports end "startxref \n57344 \n%%EOF \n".
func spacedTrailerPDF(pad, keyword string) []byte {
	content := "BT /F1 12 Tf 72 720 Td (a trailer line may end in spaces) Tj ET"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.3\n")
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
	fmt.Fprintf(&b, "trailer%s\n<< /Size %d /Root 1 0 R >>%s\n%s%s\n%d%s\n%%%%EOF%s\n",
		pad, len(objs)+1, pad, keyword, pad, xref, pad, pad)
	return b.Bytes()
}

func openAndRead(t *testing.T, data []byte) string {
	t.Helper()
	r, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	text, err := r.Page(1).GetPlainText(nil)
	if err != nil {
		t.Fatalf("page text: %v", err)
	}
	return text
}

func TestATrailerWhoseLinesEndInASpaceOpens(t *testing.T) {
	text := openAndRead(t, spacedTrailerPDF(" ", "startxref"))
	if !strings.Contains(text, "a trailer line may end in spaces") {
		t.Fatalf("page text %q does not carry the shown string", text)
	}
}

func TestATrailerWhoseLinesEndInSpacesAndTabsOpens(t *testing.T) {
	text := openAndRead(t, spacedTrailerPDF(" \t ", "startxref"))
	if !strings.Contains(text, "a trailer line may end in spaces") {
		t.Fatalf("page text %q does not carry the shown string", text)
	}
}

func TestATrailerWithoutPaddingStillOpens(t *testing.T) {
	text := openAndRead(t, spacedTrailerPDF("", "startxref"))
	if !strings.Contains(text, "a trailer line may end in spaces") {
		t.Fatalf("page text %q does not carry the shown string", text)
	}
}

// A word that only begins with the keyword is not the keyword's line.
func TestAWordThatBeginsWithStartxrefIsNotTheTrailer(t *testing.T) {
	data := spacedTrailerPDF(" ", "startxrefs")
	_, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err == nil || !strings.Contains(err.Error(), "missing final startxref") {
		t.Fatalf("open: got %v, want missing final startxref", err)
	}
}
