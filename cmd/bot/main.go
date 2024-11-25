package main

import (
	"log"
	"os"

	"github.com/d1mk9/tgChanPost/internal/bot"
)

func main() {
	BotToken := os.Getenv("TELEGRAM_APITOKEN2")
	if BotToken == "" {
		log.Fatal("Переменная окружения TELEGRAM_APITOKEN2 не установлена")
	}

	yandexAPIKey := os.Getenv("YANDEX_API_KEY")
	if yandexAPIKey == "" {
		log.Fatal("Переменная окружения YANDEX_API_KEY не установлена")
	}

	catalogID := os.Getenv("YANDEX_CATALOG_ID")
	if catalogID == "" {
		log.Fatal("Переменная окружения YANDEX_CATALOG_ID не установлена")
	}

	imageAPIKey := os.Getenv("YANDEX_API_ART_KEY")
	if imageAPIKey == "" {
		log.Fatal("Переменная окружения YANDEX_API_ART_KEY не установлена")
	}

	bot.StartBot(BotToken, yandexAPIKey, catalogID, imageAPIKey)
}
