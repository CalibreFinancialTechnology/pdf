package pdf

import (
	"bytes"
	"strings"
	"testing"
)

func withHeader(data []byte, header string) []byte {
	return append([]byte(header+"\n"), bytes.SplitN(data, []byte("\n"), 2)[1]...)
}

func TestAPDF20HeaderIsRead(t *testing.T) {
	data := withHeader(emptyStreamPDF(), "%PDF-2.0")

	r, err := NewReader(bytes.NewReader(data), int64(len(data)))

	if err != nil {
		t.Fatalf("open: %v", err)
	}
	text, err := r.Page(1).GetPlainText(nil)
	if err != nil || !strings.Contains(text, "an empty stream does not hide this line") {
		t.Fatalf("page text %q, err %v", text, err)
	}
}

func TestAnUnknownHeaderVersionStaysRefused(t *testing.T) {
	for _, header := range []string{"%PDF-3.0", "%PDF-2.1", "<!DOCTYPE html>"} {
		data := withHeader(emptyStreamPDF(), header)

		_, err := NewReader(bytes.NewReader(data), int64(len(data)))

		if err == nil || err.Error() != "not a PDF file: invalid header" {
			t.Fatalf("%s: err %v, want the invalid-header refusal", header, err)
		}
	}
}
