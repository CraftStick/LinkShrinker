package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var botToken = os.Getenv("BOT_TOKEN")

const (
	domain = "https://tewfsd.xyz"
)

var allowedUsers = map[int64]bool{
	6196125778: true,
}

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type URLStore struct {
	sync.RWMutex
	urls map[string]string
}

var store = &URLStore{
	urls: make(map[string]string),
}

func generateID(length int) string {
	b := make([]byte, length)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		b[i] = alphabet[num.Int64()]
	}
	return string(b)
}

func startBot() {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Ошибка подключения бота: %v", err)
	}

	bot.Debug = false
	log.Printf("Бот авторизован как %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	setupCommands(bot)

	for update := range updates {
		if update.InlineQuery != nil {
			go handleInline(bot, update.InlineQuery)
			continue
		}
		m := update.Message
		if m == nil || m.Text == "" || m.From == nil {
			continue
		}
		if m.IsCommand() {
			switch m.Command() {
			case "ip", "info":
				go handleIP(bot, m)
			case "start":
				go handleStart(bot, m)
			case "help":
				go handleHelp(bot, m)
			}
			continue
		}

		if !allowedUsers[m.From.ID] || m.Chat.Type != "private" {
			continue
		}

		originalURL := strings.TrimSpace(m.Text)
		if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
			originalURL = "http://" + originalURL
		}

		id := generateID(6)
		store.Lock()
		store.urls[id] = originalURL
		store.Unlock()

		shortURL := fmt.Sprintf("%s/%s", domain, id)
		msg := tgbotapi.NewMessage(m.Chat.ID, "🔗 Короткая ссылка:\n"+shortURL)
		msg.DisableWebPagePreview = true
		bot.Send(msg)
	}
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

	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		req.URL = "http://" + req.URL
	}

	id := generateID(6)
	store.Lock()
	store.urls[id] = req.URL
	store.Unlock()

	shortURL := fmt.Sprintf("%s/%s", domain, id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"short_url": shortURL})
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

func main() {
	go startBot()

	mux := http.NewServeMux()
	mux.HandleFunc("/shorten", handleShorten)
	mux.HandleFunc("/", handleRedirect)

	port := ":8080"
	fmt.Printf("HTTP сервер запущен на порту %s...\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
