package configs

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	DBOneJYP DatabaseConfig
	DBOneMNK DatabaseConfig
	DBOneTMK DatabaseConfig
	DBFiveJYP DatabaseConfig
	DBFiveMNK DatabaseConfig
	DBFiveTMK DatabaseConfig

	TelegramToken  string
	TelegramChatID string
	CronSchedule   string
	OpenAIApiKey   string
}

type DatabaseConfig struct {
	IsConfigured bool
	Host         string
	Port         string
	User         string
	Pass         string
	Name         string
}

func LoadConfig() *AppConfig {
	if err := godotenv.Load("local.env"); err != nil {
		log.Fatalf("Error: Tidak dapat memuat file local.env: %v", err)
	}

	cfg := &AppConfig{
		TelegramToken:  getEnv("TELEGRAM_BELLA_TOKEN"),
		TelegramChatID: getEnv("TELEGRAM_BELLA_GROUP_ID"),
		CronSchedule:   getEnv("CRON_SCHEDULE"),
		OpenAIApiKey:   getEnv("OPENAI_API_KEY"),
	}

	cfg.DBOneJYP = loadDBConfig("DB_ONE_JYP")
	cfg.DBOneMNK = loadDBConfig("DB_ONE_MNK")
	cfg.DBOneTMK = loadDBConfig("DB_ONE_TMK")
	cfg.DBFiveJYP = loadDBConfig("DB_FIVE_JYP")
	cfg.DBFiveMNK = loadDBConfig("DB_FIVE_MNK")
	cfg.DBFiveTMK = loadDBConfig("DB_FIVE_TMK")

	return cfg
}

func loadDBConfig(prefix string) DatabaseConfig {
	user := os.Getenv(prefix + "_USERNAME")
	if user == "" {
		return DatabaseConfig{IsConfigured: false}
	}

	return DatabaseConfig{
		IsConfigured: true,
		Host:         getEnv(prefix + "_HOST"),
		Port:         getEnv(prefix + "_PORT"),
		User:         user,
		Pass:         os.Getenv(prefix + "_PASS"),
		Name:         getEnv(prefix + "_NAME"),
	}
}

func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Error: Environment variable '%s' harus diisi dan tidak boleh kosong.", key)
	}
	return value
}