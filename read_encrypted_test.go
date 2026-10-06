package pdf

import (
	"os"
	"strings"
	"testing"
)

// Both fixtures are the same one-line page encrypted by pdfcpu with RC4 and an
// empty user password; only the key length differs. PDF 32000-1 Algorithm 1
// truncates the per-object key to min(n/8+5, 16) bytes, which is 16 for a
// 128-bit key and 10 for a 40-bit one.
func TestReadsRC4WithEmptyUserPassword(t *testing.T) {
	for _, bits := range []string{"40", "128"} {
		t.Run(bits+"-bit", func(t *testing.T) {
			f, err := os.Open("testdata/rc4-" + bits + ".pdf")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			st, _ := f.Stat()

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("reader panicked: %v", r)
				}
			}()
			r, err := NewReaderEncrypted(f, st.Size(), func() string { return "" })
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			text, err := r.Page(1).GetPlainText(nil)
			if err != nil {
				t.Fatalf("page text: %v", err)
			}
			if !strings.Contains(text, "forty bit rc4 reads this line") {
				t.Fatalf("page text %q does not carry the line", text)
			}
		})
	}
}
