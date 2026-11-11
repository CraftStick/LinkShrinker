package shorten

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
)

const (
	Domain   = "https://tewfsd.xyz"
	alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var store = struct {
	sync.RWMutex
	urls map[string]string
}{urls: map[string]string{}}

func generateID(length int) string {
	b := make([]byte, length)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		b[i] = alphabet[num.Int64()]
	}
	return string(b)
}

func HasScheme(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func Shorten(raw string) string {
	if !HasScheme(raw) {
		raw = "http://" + raw
	}
	id := generateID(6)
	store.Lock()
	store.urls[id] = raw
	store.Unlock()
	return fmt.Sprintf("%s/%s", Domain, id)
}

func handleShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"short_url": Shorten(req.URL)})
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" || id == "shorten" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	store.RLock()
	originalURL, exists := store.urls[id]
	store.RUnlock()
	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}

func Register(mux *http.ServeMux) {
	mux.HandleFunc("/shorten", handleShorten)
	mux.HandleFunc("/", handleRedirect)
}
