package bot

import (
	"bytes"
	"compress/zlib"
	"testing"
)

func TestTraceWidgetFits(t *testing.T) {
	var b bytes.Buffer
	w, _ := zlib.NewWriterLevel(&b, zlib.BestCompression)
	w.Write([]byte(traceWidget))
	w.Close()
	if b.Len() > 4096 {
		t.Fatalf("widget is %d bytes compressed, limit 4096", b.Len())
	}
}

func TestTraceDataFits(t *testing.T) {
	rows := make([]traceRow, 20)
	for i := range rows {
		rows[i] = traceRow{"DE", 123}
	}
	if n := len(traceData("a-very-long-hostname.example.com", rows, true)); n > 1024 {
		t.Fatalf("chat_data is %d bytes, limit 1024", n)
	}
}
