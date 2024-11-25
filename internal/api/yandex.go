package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/d1mk9/tgChanPost/internal/models"
)

const (
	yandexAPIURL          = "https://llm.api.cloud.yandex.net/foundationModels/v1/completion"
	yandexArtAPIURL       = "https://llm.api.cloud.yandex.net/foundationModels/v1/imageGenerationAsync"
	yandexArtOperationURL = "https://llm.api.cloud.yandex.net/operations/"
)

// GenerateMessage генерирует сообщение с использованием YandexGPT
func GenerateMessage(apiKey, catalogID, userMessage string) (models.FormattedResponse, error) {
	requestBody, err := json.Marshal(map[string]interface{}{
		"modelUri": fmt.Sprintf("gpt://%s/yandexgpt/latest", catalogID),
		"completionOptions": map[string]interface{}{
			"stream":      false,
			"temperature": 0.6,
			"maxTokens":   2000,
		},
		"messages": []map[string]string{
			{"role": "system", "text": "Ты умный ассистент"},
			{"role": "user", "text": userMessage},
		},
	})
	if err != nil {
		return models.FormattedResponse{}, err
	}

	req, err := http.NewRequest("POST", yandexAPIURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return models.FormattedResponse{}, err
	}

	req.Header.Set("Authorization", "Api-Key "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.FormattedResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return models.FormattedResponse{}, fmt.Errorf("API error: %d %s, response: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(bodyBytes))
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return models.FormattedResponse{}, err
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if alternatives, ok := result["alternatives"].([]interface{}); ok && len(alternatives) > 0 {
			if message, ok := alternatives[0].(map[string]interface{})["message"].(map[string]interface{}); ok {
				if text, ok := message["text"].(string); ok {
					return models.FormattedResponse{
						Response: text,
						Status:   "success",
					}, nil
				}
			}
		}
	}

	return models.FormattedResponse{
		Response: "Не удалось извлечь текст из ответа",
		Status:   "error",
	}, nil
}

// GenerateArtImage генерирует изображение с использованием Yandex Art API
func GenerateArtImage(apiKey, catalogID, prompt string, seed int) (string, error) {
	// Подготовка запроса
	requestBody := map[string]interface{}{
		"modelUri": fmt.Sprintf("art://%s/yandex-art/latest", catalogID),
		"generationOptions": map[string]interface{}{
			"seed": seed,
			"aspectRatio": map[string]string{
				"widthRatio":  "1",
				"heightRatio": "1",
			},
		},
		"messages": []map[string]interface{}{
			{
				"weight": "1",
				"text":   prompt,
			},
		},
	}

	// Преобразование тела запроса в JSON
	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	// Создание нового запроса
	req, err := http.NewRequest("POST", yandexArtAPIURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	// Установка заголовков
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Api-Key "+apiKey)

	// Отправка запроса на создание изображения
	client := &http.Client{Timeout: 40 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("API error: %d %s, response: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(bodyBytes))
	}

	var createResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createResponse); err != nil {
		return "", err
	}

	var operationID string
	if id, exists := createResponse["id"]; exists {
		operationID = fmt.Sprintf("%v", id)
		log.Printf("Operation ID: %s", operationID)
	} else {
		return "", fmt.Errorf("ID field not found in response: %v", createResponse)
	}

	// Ожидание завершения генерации
	for {
		time.Sleep(10 * time.Second)

		log.Printf("Checking status for operation ID: %s", operationID)
		doneResp, err := http.NewRequest("GET", fmt.Sprintf("https://llm.api.cloud.yandex.net:443/operations/%s", operationID), nil)
		if err != nil {
			return "", err
		}
		doneResp.Header.Set("Authorization", "Api-Key "+apiKey) // Установка заголовка авторизации
		resp, err := client.Do(doneResp)                        // Отправка запроса
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			return "", fmt.Errorf("API error: %d %s, response: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(bodyBytes))
		}

		var doneResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&doneResponse); err != nil {
			return "", err
		}

		done, exists := doneResponse["done"].(bool)
		if !exists {
			return "", fmt.Errorf("missing 'done' field in response: %v", doneResponse)
		}

		if done {
			if errMsg, exists := doneResponse["error"]; exists {
				return "", fmt.Errorf("operation failed: %v", errMsg)
			}

			imageData, ok := doneResponse["response"].(map[string]interface{})["image"].(string)
			if !ok {
				return "", fmt.Errorf("failed to get image data from response: %v", doneResponse)
			}

			imageBytes, err := base64.StdEncoding.DecodeString(imageData)
			if err != nil {
				return "", err
			}

			dataFile := time.Now().Format("2006-01-02_15-04-05")
			imageFileName := dataFile + ".jpeg"
			imageFile, err := os.Create(imageFileName)
			if err != nil {
				return "", err
			}
			defer imageFile.Close()

			if _, err := imageFile.Write(imageBytes); err != nil {
				return "", err
			}

			log.Printf("Image saved to %s", imageFileName)
			return imageFileName, nil
		}
	}
}
