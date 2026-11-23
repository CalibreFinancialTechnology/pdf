package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

// pageText reads page 1's plain text, failing the test if it does not return
// within 5 s.
func pageText(t *testing.T, data []byte) (string, error) {
	t.Helper()
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
		return got.text, got.err
	case <-time.After(5 * time.Second):
		t.Fatal("reading the page did not return within 5 s")
		return "", nil
	}
}

// Before content pieces were read in turn, a string left open at the end of one
// piece ended with that piece. Only an array carries into the next piece.
func TestAStringLeftOpenAtAPieceEndDoesNotSwallowTheNextPiece(t *testing.T) {
	data := twoPieceContentsPDF("BT /F1 12 Tf 72 720 Td (first) Tj ET (oops", "BT /F1 12 Tf 72 700 Td (second) Tj ET")

	text, err := pageText(t, data)

	if err != nil {
		t.Fatalf("page text: %v", err)
	}
	if !strings.Contains(text, "first") || !strings.Contains(text, "second") {
		t.Fatalf("page text %q does not carry both pieces", text)
	}
}

func TestADictionaryLeftOpenAtAPieceEndDoesNotSwallowTheNextPiece(t *testing.T) {
	data := twoPieceContentsPDF("BT /F1 12 Tf 72 720 Td (first) Tj ET /Span <</MCID 0", "BT /F1 12 Tf 72 700 Td (second) Tj ET")

	text, err := pageText(t, data)

	if err != nil {
		t.Fatalf("page text: %v", err)
	}
	if !strings.Contains(text, "first") || !strings.Contains(text, "second") {
		t.Fatalf("page text %q does not carry both pieces", text)
	}
}

// readHexString treated the end of a stream as whitespace and looped for ever.
func TestAHexStringCutByTheEndOfAStreamReturns(t *testing.T) {
	data := twoPieceContentsPDF("BT /F1 12 Tf 72 720 Td (before the hex string) Tj <4142", "")

	text, err := pageText(t, data)

	if err != nil {
		t.Fatalf("page text: %v", err)
	}
	if !strings.Contains(text, "before the hex string") {
		t.Fatalf("page text %q does not carry the text before the hex string", text)
	}
}

// A piece is opened only when the reader reaches it, so the operators of the
// pieces before one that cannot be opened are still interpreted.
func TestAPieceIsOpenedOnlyWhenReached(t *testing.T) {
	first := "BT /F1 12 Tf 72 720 Td (first) Tj ET"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 6 0 R >> >> /Contents [4 0 R 5 0 R] >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(first), first),
		"<< /Length 4 /Filter /LZWDecode >>\nstream\nabcd\nendstream",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	data := buildPDF(objs)
	r, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	var ops []string

	func() {
		defer func() { _ = recover() }()
		Interpret(r.Page(1).V.Key("Contents"), func(stk *Stack, op string) {
			ops = append(ops, op)
			for stk.Len() > 0 {
				stk.Pop()
			}
		})
	}()

	if strings.Join(ops, " ") != "BT Tf Td Tj ET" {
		t.Fatalf("operators before the unopenable piece = %v, want BT Tf Td Tj ET", ops)
	}
}

func buildPDF(objs []string) []byte {
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

// An odd number of hex digits leaves readHexString waiting for the second digit
// of a pair when the stream ends.
func TestAHexStringCutAfterAnOddDigitReturns(t *testing.T) {
	data := twoPieceContentsPDF("BT /F1 12 Tf 72 720 Td (before the odd hex string) Tj <414", "")

	text, err := pageText(t, data)

	if err != nil {
		t.Fatalf("page text: %v", err)
	}
	if !strings.Contains(text, "before the odd hex string") {
		t.Fatalf("page text %q does not carry the text before the hex string", text)
	}
}
