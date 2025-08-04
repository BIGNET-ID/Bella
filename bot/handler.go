package bot

import (
	"bella/api"
	config "bella/config"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotHandler struct {
	bot            *tgbotapi.BotAPI
	authorizedIDs  map[int64]bool
	commandHandler *CommandHandler
}

func NewBotHandler(config *config.AppConfig, apiClient *api.APIClient) (*BotHandler, error) {
	bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		return nil, fmt.Errorf("gagal menginisialisasi bot Telegram: %w", err)
	}
	bot.Debug = false
	slog.Info("Berhasil terhubung sebagai bot", "username", bot.Self.UserName)

	authMap := make(map[int64]bool)
	for _, idStr := range config.AuthorizedTelegramIDs {
		id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
		if err == nil {
			authMap[id] = true
		}
	}

	commandHandler := NewCommandHandler(bot, config, apiClient)

	return &BotHandler{
		bot:            bot,
		authorizedIDs:  authMap,
		commandHandler: commandHandler,
	}, nil
}


// setCommandsForUser mengatur daftar perintah yang terlihat oleh pengguna spesifik.
func (h *BotHandler) setCommandsForUser(chatID int64, isAuthorized bool) {
	var commands []tgbotapi.BotCommand
	if isAuthorized {
		commands = getAdminCommands()
	} else {
		commands = getPublicCommands()
	}

	// Menggunakan scope untuk menargetkan chat spesifik (pengguna)
	scope := tgbotapi.NewBotCommandScopeChat(chatID)
	config := tgbotapi.NewSetMyCommandsWithScope(scope, commands...)

	if _, err := h.bot.Request(config); err != nil {
		slog.Warn("Gagal mengatur perintah untuk pengguna", "chat_id", chatID, "error", err)
	} else {
		slog.Info("Berhasil mengatur perintah untuk pengguna", "chat_id", chatID, "is_admin", isAuthorized)
	}
}

func (h *BotHandler) StartPolling() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := h.bot.GetUpdatesChan(u)

	slog.Info("Bot mulai mendengarkan perintah...")
	for update := range updates {
		if update.Message != nil && update.Message.IsCommand() {
			go h.handleCommand(update.Message)
		}
	}
}


func (h *BotHandler) handleCommand(message *tgbotapi.Message) {
	userID := message.From.ID
	isAuthorized := h.authorizedIDs[userID]
	command := message.Command()

	slog.Info("Menerima perintah", "command", command, "from", message.From.UserName, "user_id", userID)

	// Saat pengguna berinteraksi dengan /start atau /help,
	// perbarui daftar perintah yang mereka lihat.
	if command == "start" || command == "help" {
		h.setCommandsForUser(message.Chat.ID, isAuthorized)
	}

	// Daftar lengkap perintah admin untuk pengecekan akses
	adminCommands := map[string]bool{
		"satria1_gateway_all":   true,
		"satria1_gateway_jyp":   true,
		"satria1_gateway_mnk":   true,
		"satria1_gateway_tmk":   true,
		"satria1_iptx_jyp":      true,
		"satria1_iptx_mnk":      true,
		"satria1_iptx_tmk":      true,
		"log_error":             true,
		"log_notif":             true,
		"log_alerts_active":     true,
		"log_all":               true,
	}

	// Cek otorisasi HANYA untuk perintah yang terdaftar sebagai admin
	if adminCommands[command] && !isAuthorized {
		h.commandHandler.sendMessage(message.Chat.ID, escape("❌ *Akses Ditolak*! Anda tidak memiliki izin untuk menggunakan perintah ini."))
		return
	}

	// Routing perintah
	switch command {
	case "start", "help":
		helpMessage := GetHelpMessage(isAuthorized)
		h.commandHandler.sendMessage(message.Chat.ID, helpMessage)
	case "myid":
		h.commandHandler.sendMessage(message.Chat.ID, FormatMyIDMessage(userID))

	// Perintah Gateway (sudah dipastikan terotorisasi)
	case "satria1_gateway_jyp":
		go h.commandHandler.HandleGatewaySummary(message.Chat.ID, "Jayapura")
	case "satria1_gateway_mnk":
		go h.commandHandler.HandleGatewaySummary(message.Chat.ID, "Manokwari")
	case "satria1_gateway_tmk":
		go h.commandHandler.HandleGatewaySummary(message.Chat.ID, "Timika")
	case "satria1_gateway_all":
		go h.commandHandler.HandleGatewayAll(message.Chat.ID)

	// Perintah IP Transit (sudah dipastikan terotorisasi)
	case "satria1_iptx_jyp":
		go h.commandHandler.HandleIpTransitInfo(message.Chat.ID, "Jayapura")
	case "satria1_iptx_mnk":
		go h.commandHandler.HandleIpTransitInfo(message.Chat.ID, "Manokwari")
	case "satria1_iptx_tmk":
		go h.commandHandler.HandleIpTransitInfo(message.Chat.ID, "Timika")

	// Perintah Log (sudah dipastikan terotorisasi)
	case "log_error", "log_notif", "log_alerts_active", "log_all":
		go h.commandHandler.HandleLogs(message.Chat.ID, command)

	default:
		// Jangan kirim "perintah tidak dikenal" jika itu adalah perintah admin oleh non-admin
		if !adminCommands[command] {
			h.commandHandler.sendMessage(message.Chat.ID, escape("Perintah tidak dikenal. Ketik /help untuk melihat daftar perintah."))
		}
	}
}