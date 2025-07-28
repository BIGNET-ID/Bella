package bot

import (
	config "bella/config"
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotHandler struct {
	bot           *tgbotapi.BotAPI
	authorizedIDs map[int64]bool
}

func NewBotHandler(config *config.AppConfig) (*BotHandler, error) {
	bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		return nil, fmt.Errorf("gagal menginisialisasi bot Telegram: %w", err)
	}
	bot.Debug = false
	slog.Info("Berhasil terhubung sebagai bot", "username", bot.Self.UserName)

	authMap := make(map[int64]bool)
	for _, idStr := range config.AuthorizedTelegramIDs {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err == nil {
			authMap[id] = true
		}
	}

	commands := []tgbotapi.BotCommand{
		{Command: "log", Description: "Tampilkan menu untuk melihat log"},
		{Command: "myid", Description: "Tampilkan ID Telegram Anda"},
		{Command: "help", Description: "Tampilkan pesan bantuan ini"},
	}
	if _, err := bot.Request(tgbotapi.NewSetMyCommands(commands...)); err != nil {
		slog.Warn("Gagal mendaftarkan perintah bot", "error", err)
	} else {
		slog.Info("Perintah bot berhasil didaftarkan ke Telegram.")
	}

	return &BotHandler{bot: bot, authorizedIDs: authMap}, nil
}

func (h *BotHandler) StartPolling() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := h.bot.GetUpdatesChan(u)

	slog.Info("Bot mulai mendengarkan perintah dan interaksi...")
	for update := range updates {
		if update.Message != nil && update.Message.IsCommand() {
			go h.handleCommand(update.Message)
			continue
		}
		if update.CallbackQuery != nil {
			go h.handleCallbackQuery(update.CallbackQuery)
			continue
		}
	}
}

func (h *BotHandler) handleCommand(message *tgbotapi.Message) {
	userID := message.From.ID
	isAuthorized := h.authorizedIDs[userID]

	slog.Info("Menerima perintah", "command", message.Command(), "from_user", message.From.UserName)

	switch message.Command() {
	case "myid":
		h.sendMessage(userID, fmt.Sprintf("ID Telegram Anda adalah: `%d`", userID))

	case "log":
		if !isAuthorized {
			h.sendMessage(userID, "Maaf, Anda tidak memiliki izin untuk menggunakan perintah ini.")
			return
		}
		h.sendLogMenu(userID)

	case "help":
		h.sendMessage(userID, "Perintah yang tersedia:\n/log - Tampilkan menu log.\n/myid - Tampilkan ID Telegram Anda.")

	default:
		if isAuthorized {
			h.sendMessage(userID, "Perintah tidak dikenal. Ketik /help untuk bantuan.")
		}
	}
}

func (h *BotHandler) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	isAuthorized := h.authorizedIDs[userID]

	if !isAuthorized {
		h.answerCallback(callback, "Akses ditolak.")
		return
	}

	filter := callback.Data

	deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
	h.bot.Request(deleteMsg)

	h.handleLogCommand(userID, filter)
	h.answerCallback(callback, fmt.Sprintf("Memuat log dengan filter: %s", filter))
}

func (h *BotHandler) sendLogMenu(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Pilih kategori log yang ingin Anda lihat:")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Semua Log", "all"),
			tgbotapi.NewInlineKeyboardButtonData("Hanya Error", "error"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Gateway JYP", "JYP"),
			tgbotapi.NewInlineKeyboardButtonData("Gateway MNK", "MNK"),
			tgbotapi.NewInlineKeyboardButtonData("Gateway TMK", "TMK"),
		),
	)
	msg.ReplyMarkup = keyboard

	if _, err := h.bot.Send(msg); err != nil {
		slog.Warn("Gagal mengirim menu log", "error", err)
	}
}

func (h *BotHandler) handleLogCommand(chatID int64, filter string) {
	const maxLines = 20
	file, err := os.Open(filepath.Join("logs", "bella.log"))
	if err != nil {
		slog.Error("Gagal membuka file log", "error", err)
		h.sendMessage(chatID, "Error: Tidak dapat membaca file log.")
		return
	}
	defer file.Close()

	var filteredLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry struct {
			Time    time.Time `json:"time"`
			Level   string    `json:"level"`
			Msg     string    `json:"msg"`
			Gateway string    `json:"gateway,omitempty"`
		}
		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if filter != "all" {
			filterLower := strings.ToLower(filter)
			levelLower := strings.ToLower(entry.Level)
			gatewayLower := strings.ToLower(entry.Gateway)
			if !strings.Contains(levelLower, filterLower) && !strings.Contains(gatewayLower, filterLower) {
				continue
			}
		}

		formattedLine := fmt.Sprintf("[%s] %s - %s", entry.Time.Format("15:04:05"), entry.Level, entry.Msg)
		if entry.Gateway != "" {
			formattedLine += fmt.Sprintf(" (%s)", entry.Gateway)
		}
		filteredLines = append(filteredLines, formattedLine)
	}

	if len(filteredLines) > maxLines {
		filteredLines = filteredLines[len(filteredLines)-maxLines:]
	}

	if len(filteredLines) == 0 {
		h.sendMessage(chatID, fmt.Sprintf("Tidak ada log yang cocok dengan filter '%s'.", filter))
		return
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString("```\n")
	responseBuilder.WriteString(fmt.Sprintf("--- Menampilkan %d log terakhir (filter: %s) ---\n", len(filteredLines), filter))
	responseBuilder.WriteString(strings.Join(filteredLines, "\n"))
	responseBuilder.WriteString("\n```")

	h.sendMessageWithMarkdown(chatID, responseBuilder.String())
}

func (h *BotHandler) answerCallback(callback *tgbotapi.CallbackQuery, text string) {
	callbackConfig := tgbotapi.NewCallback(callback.ID, text)
	h.bot.Request(callbackConfig)
}

func (h *BotHandler) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if _, err := h.bot.Send(msg); err != nil {
		slog.Warn("Gagal mengirim pesan balasan", "error", err)
	}
}

func (h *BotHandler) sendMessageWithMarkdown(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if _, err := h.bot.Send(msg); err != nil {
		slog.Warn("Gagal mengirim pesan markdown balasan", "error", err)
	}
}
