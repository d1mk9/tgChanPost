package configs

import (
	"log"
	"os"
)

// Config содержит все глобальные конфигурационные параметры
type Config struct {
	BotToken     string
	YandexAPIKey string
	CatalogID    string
	ImageAPIKey  string
}

// GlobalConfig - глобальная переменная для хранения конфигурации
var GlobalConfig Config

func LoadConfig() {
	/*err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки файла .env")
	}*/

	GlobalConfig.BotToken = os.Getenv("TELEGRAM_APITOKEN2")
	if GlobalConfig.BotToken == "" {
		log.Fatal("Переменная окружения TELEGRAM_APITOKEN2 не установлена")
	}

	GlobalConfig.YandexAPIKey = os.Getenv("YANDEX_API_KEY")
	if GlobalConfig.YandexAPIKey == "" {
		log.Fatal("Переменная окружения YANDEX_API_KEY не установлена")
	}

	GlobalConfig.CatalogID = os.Getenv("YANDEX_CATALOG_ID")
	if GlobalConfig.CatalogID == "" {
		log.Fatal("Переменная окружения YANDEX_CATALOG_ID не установлена")
	}

	GlobalConfig.ImageAPIKey = os.Getenv("YANDEX_API_ART_KEY")
	if GlobalConfig.ImageAPIKey == "" {
		log.Fatal("Переменная окружения YANDEX_API_ART_KEY не установлена")
	}

}
