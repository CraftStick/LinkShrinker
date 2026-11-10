package game

import (
	"bytes"
	"compress/zlib"
	"testing"
)

func TestWidgetLimits(t *testing.T) {
	h := HTML()
	if len(h) > 32768 {
		t.Fatalf("html is %d bytes, limit 32768", len(h))
	}
	var b bytes.Buffer
	w, _ := zlib.NewWriterLevel(&b, zlib.BestCompression)
	w.Write([]byte(h))
	w.Close()
	if b.Len() > 4096 {
		t.Fatalf("compressed html is %d bytes, limit 4096", b.Len())
	}
}
