package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/d1mk9/tgChanPost/internal/models"
)

// SaveInteractionToFile сохраняет интеракцию в файл в формате JSON
func SaveInteractionToFile(interaction models.PromtReq) error {
	// Определяем путь к файлу
	filePath := "promtreq.json"

	// Открываем файл для чтения и записи
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	// Считываем существующие данные
	var interactions []models.PromtReq
	if err := json.NewDecoder(file).Decode(&interactions); err != nil && err != io.EOF {
		return fmt.Errorf("ошибка декодирования данных из файла: %w", err)
	}

	// Добавляем новую интеракцию
	interactions = append(interactions, interaction)

	// Перемещаем указатель файла в начало
	file.Seek(0, 0)

	// Очищаем файл
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("ошибка очистки файла: %w", err)
	}

	// Записываем обновленный массив интеракций в файл
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Для более читабельного формата
	if err := encoder.Encode(interactions); err != nil {
		return fmt.Errorf("ошибка записи данных в файл: %w", err)
	}

	return nil
}

// CleanQuote cleans the quote from duplicate quotes
func CleanQuote(quote string) string {
	re := regexp.MustCompile(`«{2,}|»{2,}`)
	quote = re.ReplaceAllString(quote, "«")
	quote = strings.TrimSpace(quote)
	quote = strings.Trim(quote, "\"")
	quote = strings.ReplaceAll(quote, "«\"", "«")
	quote = strings.ReplaceAll(quote, "\"»", "»")
	quote = strings.Trim(quote, "«»")
	return quote
}

// ExtractQuoteAndAuthor извлекает цитату и автора из строки
func ExtractQuoteAndAuthor(response string) (string, string, error) {
	log.Printf("Response to parse: %s", response)

	// Удаляем пробелы в начале и конце строки
	response = strings.TrimSpace(response)

	// Регулярное выражение для извлечения цитаты
	reQuote := regexp.MustCompile(`^«(.*?)»\s*[-—]?\s*`)
	quoteMatches := reQuote.FindStringSubmatch(response)

	// Проверяем, что мы получили цитату
	if len(quoteMatches) < 2 {
		return "", "без автора", fmt.Errorf("недостаточно данных для извлечения цитаты")
	}

	quote := strings.TrimSpace(quoteMatches[1])
	author := "без автора" // Значение по умолчанию для автора

	// Оставшаяся часть строки после извлечения цитаты
	remaining := strings.TrimSpace(response[len(quoteMatches[0]):])

	// Проверяем, указан ли автор
	if remaining != "" {
		// Регулярное выражение для извлечения автора
		reAuthor := regexp.MustCompile(`[-—]?\s*(.+)$`)
		authorMatches := reAuthor.FindStringSubmatch(remaining)

		if len(authorMatches) > 1 {
			author = strings.TrimSpace(authorMatches[1])
		}
	}

	return quote, author, nil
}
