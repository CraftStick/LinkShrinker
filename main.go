package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"shortener/internal/bot"
	"shortener/internal/shorten"
)

func main() {
	go bot.Start(os.Getenv("BOT_TOKEN"))

	mux := http.NewServeMux()
	shorten.Register(mux)

	port := ":8080"
	fmt.Printf("HTTP сервер запущен на порту %s...\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
