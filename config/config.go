package configs

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	DBOneJYP  DatabaseConfig
	DBOneMNK  DatabaseConfig
	DBOneTMK  DatabaseConfig
	DBFiveJYP DatabaseConfig
	DBFiveMNK DatabaseConfig
	DBFiveTMK DatabaseConfig

	TelegramToken  string
	TelegramChatID string
	CronSchedule   string

	AuthorizedTelegramIDs []string

	PRTGUrl      string
	PRTGAPITOKEN string

	IPTX_JYP string
	IPTX_MNK string
	IPTX_TMK string

	NIF_JYP string
	NIF_MNK string
	NIF_TMK string

	G1gURL string
	G1kURL string
	G1lURL string

	APIEmail    string
	APIPassword string
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
		log.Println("Peringatan: Tidak dapat menemukan file local.env, akan menggunakan environment variables yang ada.")
	}

	rawIDs := getEnv("AUTHORIZED_TELEGRAM_IDS")
	splitIDs := strings.Split(rawIDs, ",")
	authorizedIDs := make([]string, 0, len(splitIDs))
	for _, id := range splitIDs {
		trimmedID := strings.TrimSpace(id)
		if trimmedID != "" {
			authorizedIDs = append(authorizedIDs, trimmedID)
		}
	}

	cfg := &AppConfig{
		TelegramToken:         getEnv("TELEGRAM_BELLA_TOKEN"),
		TelegramChatID:        getEnv("TELEGRAM_BELLA_GROUP_ID"),
		AuthorizedTelegramIDs: authorizedIDs,

		CronSchedule: getEnv("CRON_SCHEDULE"),

		PRTGUrl:      getEnv("PRTG_URL"),
		PRTGAPITOKEN: getEnv("PRTG_API_TOKEN"),

		IPTX_JYP: getEnv("IPTX_JYP"),
		IPTX_MNK: getEnv("IPTX_MNK"),
		IPTX_TMK: getEnv("IPTX_TMK"),

		NIF_JYP: getEnv("NIF_JYP"),
		NIF_MNK: getEnv("NIF_MNK"),
		NIF_TMK: getEnv("NIF_TMK"),

		G1gURL: getEnv("G1G_URL"),
		G1kURL: getEnv("G1K_URL"),
		G1lURL: getEnv("G1L_URL"),

		APIEmail:    getEnv("API_EMAIL"),
		APIPassword: getEnv("API_PASSWORD"),
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
