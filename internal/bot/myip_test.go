package bot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"testing"
	"time"
)

func sign(tok, authDate string) string {
	user := `{"id":42,"language_code":"ru"}`
	check := "auth_date=" + authDate + "\nuser=" + user
	k := hmac.New(sha256.New, []byte("WebAppData"))
	k.Write([]byte(tok))
	m := hmac.New(sha256.New, k.Sum(nil))
	m.Write([]byte(check))
	return "auth_date=" + authDate + "&user=" + url.QueryEscape(user) + "&hash=" + hex.EncodeToString(m.Sum(nil))
}

func TestValidInitData(t *testing.T) {
	now := fmt.Sprint(time.Now().Unix())
	if id, lang, ok := validInitData(sign("tok", now), "tok"); !ok || id != 42 || lang != "ru" {
		t.Fatal("valid data rejected")
	}
	if _, _, ok := validInitData(sign("tok", now), "other"); ok {
		t.Error("wrong token accepted")
	}
	if _, _, ok := validInitData(sign("tok", "1000"), "tok"); ok {
		t.Error("stale data accepted")
	}
}
