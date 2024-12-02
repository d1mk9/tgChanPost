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

var photoMsg tgbotapi.PhotoConfig
var formattedQuote string                  // Объявляем переменную для хранения отформатированной цитаты
var waitingForQuery = make(map[int64]bool) // Хранит состояние ожидания для каждого чата

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
		if update.Message != nil {
			if err := handleMessage(bot, update.Message); err != nil {
				log.Printf("Ошибка при обработке сообщения: %v", err)
			}
		} else if update.CallbackQuery != nil {
			if err := handleCallback(bot, update.CallbackQuery); err != nil {
				log.Printf("Ошибка при обработке callback: %v", err)
			}
		}
	}
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) error {
	log.Printf("[%s] %s", message.From.UserName, message.Text)

	// Проверяем, ожидаем ли мы новый запрос от пользователя
	if waitingForQuery[message.Chat.ID] {
		// Если мы ожидаем новый запрос, генерируем новый ответ
		userQuery := message.Text
		response, err := generateResponse(userQuery)
		if err != nil {
			log.Printf("Ошибка генерации сообщения: %v", err)
			return err
		}

		if response.Response == "" {
			log.Printf("Проверка наличия ответа %d", message.Chat.ID)
			return nil
		}

		quote, author, err := utils.ExtractQuoteAndAuthor(response.Response)
		if err != nil {
			log.Printf("Ошибка формата ответа %d: %s. Ошибка: %v", message.Chat.ID, response.Response, err)
			return err
		}

		imageFileName, err := generateImage(quote)
		if err != nil {
			log.Printf("Ошибка генерации изображения %d: %v", message.Chat.ID, err)
			return err
		}

		if err := sendPost(bot, message.Chat.ID, imageFileName, quote, author); err != nil {
			log.Printf("Ошибка отправки изображения: %v", err)
		}

		// Сброс состояния ожидания
		delete(waitingForQuery, message.Chat.ID)
		return nil
	}

	// Если не ожидаем новый запрос, просто обрабатываем обычное сообщение
	userQuery := message.Text
	response, err := generateResponse(userQuery)
	if err != nil {
		log.Printf("Ошибка генерации сообщения: %v", err)
		return err
	}

	if response.Response == "" {
		log.Printf("Проверка наличия ответа %d", message.Chat.ID)
		return nil
	}

	quote, author, err := utils.ExtractQuoteAndAuthor(response.Response)
	if err != nil {
		log.Printf("Ошибка формата ответа %d: %s. Ошибка: %v", message.Chat.ID, response.Response, err)
		return err
	}

	imageFileName, err := generateImage(quote)
	if err != nil {
		log.Printf("Ошибка генерации изображения %d: %v", message.Chat.ID, err)
		return err
	}

	if err := sendPost(bot, message.Chat.ID, imageFileName, quote, author); err != nil {
		log.Printf("Ошибка отправки изображения: %v", err)
	}

	// Сохранение интеракции
	interaction := models.PromtReq{
		ChatID:    message.Chat.ID,
		UserQuery: userQuery,
		Quote:     quote,
		Author:    author,
		Timestamp: time.Now(),
	}

	if err := utils.SaveInteractionToFile(interaction); err != nil {
		log.Printf(" Ошибка сохранения файла интеракции: %v", err)
	}

	return nil
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
	formattedQuote = fmt.Sprintf("«%s»\n\n_%s_\n\n[%s](https://t.me/offthepages)", quote, author, "Мысли, сошедшие со страниц")
	// Открываем файл для отправки
	file, err := os.Open(imageFileName)
	if err != nil {
		return fmt.Errorf("ошибка открытия изображения: %v", err)
	}
	defer file.Close()

	keyboardAfterGenerate := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сгенерировать еще", "genAgain"),
			tgbotapi.NewInlineKeyboardButtonData("Отправить в канал", "sendCh"),
		),
	)

	// Отправка изображения с подписью
	photoMsg = tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(imageFileName))
	photoMsg.ParseMode = "Markdown"
	photoMsg.Caption = formattedQuote            // Устанавливаем отформатированную цитату в качестве подписи
	photoMsg.ReplyMarkup = keyboardAfterGenerate // Добавляем кнопки

	if _, err := bot.Send(photoMsg); err != nil {
		return fmt.Errorf("ошибка отправки изображения: %v", err)
	}

	return nil
}

func handleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) error {
	cb := callback.Data

	log.Printf("Получен Callback: %s", cb)

	switch cb {
	case "genAgain":
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "Пожалуйста, введите запрос для цитаты:")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
			return err
		}
		// Устанавливаем состояние ожидания для текущего чата
		waitingForQuery[callback.Message.Chat.ID] = true
	case "sendCh":

		msgtoch := tgbotapi.NewPhotoToChannel("@offthepages", photoMsg.File)
		msgtoch.ParseMode = "Markdown"
		msgtoch.Caption = formattedQuote // Устанавливаем отформатированную цитату в качестве подписи
		if _, err := bot.Send(msgtoch); err != nil {
			log.Printf("Ошибка отправки изображения в канал: %v", err)
			return err
		}
	}

	// Ответ на callback_query
	answer := tgbotapi.CallbackConfig{
		CallbackQueryID: callback.ID,
		Text:            "Обработка завершена",
		ShowAlert:       false,
	}

	if _, err := bot.Request(answer); err != nil {
		log.Printf("Ошибка ответа на callback_query: %v", err)
		return err
	}

	return nil
}
