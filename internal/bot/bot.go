package bot

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/d1mk9/tgChanPost/internal/api"
	"github.com/d1mk9/tgChanPost/internal/models"
	"github.com/d1mk9/tgChanPost/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func StartBot(BotToken, yandexAPIKey, catalogID, imageAPIKey string) {
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Аккаунт %s авторизован", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		userQuery := update.Message.Text
		response, err := api.GenerateMessage(yandexAPIKey, catalogID, userQuery)
		if err != nil {
			log.Printf("Ошибка генерации сообщения %d: %v", update.Message.Chat.ID, err)
			response = models.FormattedResponse{
				Response: "Извините, произошла ошибка при обработке вашего запроса.",
				Status:   "error",
			}
		}

		// Проверка на наличие ответа
		if response.Response == "" {
			log.Printf("Проверка наличия ответа %d", update.Message.Chat.ID)
			continue
		}

		quote, author, err := utils.ExtractQuoteAndAuthor(response.Response)
		if err != nil {
			log.Printf("Ошибка формата ответа %d: %s. Ошибка: %v", update.Message.Chat.ID, response.Response, err)
			continue
		}

		// Генерация изображения на основе цитаты
		seed := time.Now().Nanosecond() // Используем текущее время в качестве сид
		imageFileName, err := api.GenerateArtImage(imageAPIKey, catalogID, quote, seed)
		if err != nil {
			log.Printf("Ошибка генерации изображения %d: %v", update.Message.Chat.ID, err)
			continue
		}

		// Проверка существования файла
		if _, err := os.Stat(imageFileName); os.IsNotExist(err) {
			log.Printf("Изображение не существует: %s", imageFileName)
			continue
		}

		// Форматируем цитату для отправки
		formattedQuote := formatQuote(quote, author)

		// Открываем файл для отправки
		file, err := os.Open(imageFileName)
		if err != nil {
			log.Printf("Ошибка открытия изображения: %v", err)
			continue
		}
		defer file.Close()

		// Отправка изображения с подписью
		photoMsg := tgbotapi.NewPhoto(update.Message.Chat.ID, tgbotapi.FilePath(imageFileName))
		photoMsg.ParseMode = "Markdown"
		photoMsg.Caption = formattedQuote // Устанавливаем отформатированную цитату в качестве подписи

		if _, err := bot.Send(photoMsg); err != nil {
			log.Printf("Ошибка отправки изображения: %v", err)
		}

		// Сохранение интеракции
		interaction := models.PromtReq{
			ChatID:    update.Message.Chat.ID,
			UserQuery: userQuery,
			Quote:     quote,
			Author:    author,
			Timestamp: time.Now(),
		}

		if err := utils.SaveInteractionToFile(interaction); err != nil {
			log.Printf("Ошибка сохранения файла интеракции: %v", err)
		}
	}
}

func formatQuote(quote, author string) string {
	return fmt.Sprintf("«%s»\n\n_%s_\n\n[%s](https://t.me/offthepages)", quote, author, "Мысли, сошедшие со страниц")
}
