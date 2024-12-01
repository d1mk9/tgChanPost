package bot

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/d1mk9/tgChanPost/configs"
	"github.com/d1mk9/tgChanPost/internal/api"
	"github.com/d1mk9/tgChanPost/internal/models"
	"github.com/d1mk9/tgChanPost/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func StartBot() {
	bot, err := tgbotapi.NewBotAPI(configs.GlobalConfig.BotToken)
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
		response, err := generateResponse(userQuery)
		if err != nil {
			log.Printf("Ошибка генерации сообщения: %v", err)
			continue
		}

		if response.Response == "" {
			log.Printf("Проверка наличия ответа %d", update.Message.Chat.ID)
			continue
		}

		quote, author, err := utils.ExtractQuoteAndAuthor(response.Response)
		if err != nil {
			log.Printf("Ошибка формата ответа %d: %s. Ошибка: %v", update.Message.Chat.ID, response.Response, err)
			continue
		}

		imageFileName, err := generateImage(quote)
		if err != nil {
			log.Printf("Ошибка генерации изображения %d: %v", update.Message.Chat.ID, err)
			continue
		}

		if err := sendPost(bot, update.Message.Chat.ID, imageFileName, quote, author); err != nil {
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

func generateResponse(userQuery string) (models.FormattedResponse, error) {
	response, err := api.GenerateMessage(configs.GlobalConfig.YandexAPIKey, configs.GlobalConfig.CatalogID, userQuery)
	if err != nil {
		return models.FormattedResponse{}, err
	}
	return response, nil
}

func generateImage(quote string) (string, error) {
	// Генерация изображения на основе цитаты
	seed := time.Now().Nanosecond() // Используем текущее время в качестве сид
	imageFileName, err := api.GenerateArtImage(configs.GlobalConfig.ImageAPIKey, configs.GlobalConfig.CatalogID, quote, seed)
	if err != nil {
		return "", err
	}

	// Проверка существования файла
	if _, err := os.Stat(imageFileName); os.IsNotExist(err) {
		return "", fmt.Errorf("изображение не существует: %s", imageFileName)
	}

	return imageFileName, nil
}

func sendPost(bot *tgbotapi.BotAPI, chatID int64, imageFileName, quote, author string) error {
	// Форматируем цитату для отправки
	formattedQuote := fmt.Sprintf("«%s»\n\n_%s_\n\n[%s](https://t.me/offthepages)", quote, author, "Мысли, сошедшие со страниц")
	// Открываем файл для отправки
	file, err := os.Open(imageFileName)
	if err != nil {
		return fmt.Errorf("ошибка открытия изображения: %v", err)
	}
	defer file.Close()

	// Отправка изображения с подписью
	photoMsg := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(imageFileName))
	photoMsg.ParseMode = "Markdown"
	photoMsg.Caption = formattedQuote // Устанавливаем отформатированную цитату в качестве подписи

	if _, err := bot.Send(photoMsg); err != nil {
		return fmt.Errorf("ошибка отправки изображения: %v", err)
	}

	msgtoch := tgbotapi.NewPhotoToChannel("@offthepages", photoMsg.File)
	msgtoch.ParseMode = "Markdown"
	msgtoch.Caption = formattedQuote // Устанавливаем отформатированную цитату в качестве подписи
	bot.Send(msgtoch)
	return nil
}
