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

	commands := []tgbotapi.BotCommand{
		{Command: "help", Description: "Tampilkan bantuan"},
		{Command: "myid", Description: "Tampilkan ID Telegram Anda"},
		{Command: "satria1_gateway_all", Description: "Ringkasan status semua Gateway SATRIA-1"},
		{Command: "satria1_gateway_jyp", Description: "Ringkasan status Gateway Jayapura"},
		{Command: "satria1_gateway_mnk", Description: "Ringkasan status Gateway Manokwari"},
		{Command: "satria1_gateway_tmk", Description: "Ringkasan status Gateway Timika"},
		{Command: "satria1_iptx_jyp", Description: "Info IP Transit Gateway Jayapura"},
		{Command: "satria1_iptx_mnk", Description: "Info IP Transit Gateway Manokwari"},
		{Command: "satria1_iptx_tmk", Description: "Info IP Transit Gateway Timika"},
	}
	if _, err := bot.Request(tgbotapi.NewSetMyCommands(commands...)); err != nil {
		slog.Warn("Gagal mendaftarkan perintah bot", "error", err)
	}

	return &BotHandler{
		bot:            bot,
		authorizedIDs:  authMap,
		commandHandler: commandHandler,
	}, nil
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
	if !h.authorizedIDs[userID] {
		h.commandHandler.sendMessage(message.Chat.ID, escape("Maaf, Anda tidak memiliki izin untuk menggunakan bot ini."))
		return
	}

	slog.Info("Menerima perintah", "command", message.Command(), "from", message.From.UserName)

	switch message.Command() {
	case "start", "help":
		h.commandHandler.sendMessage(message.Chat.ID, escape("Selamat datang! Silakan pilih perintah yang tersedia."))
	case "myid":
		h.commandHandler.sendMessage(message.Chat.ID, fmt.Sprintf("ID Telegram Anda adalah: `%d`", userID))

	case "satria1_gateway_jyp":
		go h.commandHandler.HandleGatewaySummary(message.Chat.ID, "Jayapura")
	case "satria1_gateway_mnk":
		go h.commandHandler.HandleGatewaySummary(message.Chat.ID, "Manokwari")
	case "satria1_gateway_tmk":
		go h.commandHandler.HandleGatewaySummary(message.Chat.ID, "Timika")
	case "satria1_gateway_all":
		go h.commandHandler.HandleGatewayAll(message.Chat.ID)

	case "satria1_iptx_jyp":
		go h.commandHandler.HandleIpTransitInfo(message.Chat.ID, "Jayapura")
	case "satria1_iptx_mnk":
		go h.commandHandler.HandleIpTransitInfo(message.Chat.ID, "Manokwari")
	case "satria1_iptx_tmk":
		go h.commandHandler.HandleIpTransitInfo(message.Chat.ID, "Timika")

	default:
		h.commandHandler.sendMessage(message.Chat.ID, escape("Perintah tidak dikenal. Ketik /help untuk melihat daftar perintah."))
	}
}
