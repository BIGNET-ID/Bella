package main

import (
	configs "bella/config"
	"bella/db"
	"bella/internal/bot"
	"bella/internal/logger"
	"bella/internal/notifier"
	"bella/internal/prtgn"
	"bella/internal/state"
	"bella/setup"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
)

func main() {
	logger.InitSlog()

	slog.Info("Memulai Bella Alert System & Bot ....")

	config := configs.LoadConfig()
	allConnections := db.InitializeDatabases(config)
	defer allConnections.CloseAll()

	stateManager := state.NewManager("logs/active_alerts.json")
	telegramNotifier := notifier.NewTelegramNotifier(config.TelegramToken, config.TelegramChatID)

	satnetServiceMap := setup.RegisterServices(allConnections, telegramNotifier, stateManager)
	prtgAPI := prtgn.NewPRTGAPI(config, telegramNotifier, stateManager)

	scheduler := cron.New()
	setup.RegisterCronJobs(scheduler, config, satnetServiceMap, prtgAPI, allConnections, telegramNotifier, stateManager)
	
	if len(scheduler.Entries()) > 0 {
		scheduler.Start()
		slog.Info("Scheduler berhasil dimulai", "tasks_count", len(scheduler.Entries()))
	} else {
		slog.Warn("Tidak ada tugas cron yang didaftarkan.")
	}

	botHandler, err := bot.NewBotHandler(config)
	if err != nil {
		slog.Error("Gagal membuat bot handler", "error", err)
		os.Exit(1)
	}
	go botHandler.StartPolling()

	slog.Info("Aplikasi berjalan. Tekan Ctrl+C untuk berhenti.")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Menerima sinyal shutdown, menghentikan scheduler...")
	ctx := scheduler.Stop()
	<-ctx.Done()
	slog.Info("Aplikasi berhasil dihentikan.")
}
