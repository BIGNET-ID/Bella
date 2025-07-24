package main

import (
	configs "bella/config"
	"bella/db"
	"bella/internal/notifier"
	"bella/setup"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/robfig/cron/v3"
)

func main() {
	log.Println("Memulai Bella Alert System ....")

	config := configs.LoadConfig()

	allConnections := db.InitializeDatabases(config)

	telegramNotifier := notifier.NewTelegramNotifier(config.TelegramToken, config.TelegramChatID)

	scheduler := cron.New()

	setup.RegisterServicesAndTasks(allConnections, telegramNotifier, scheduler, config)

	if len(scheduler.Entries()) > 0 {
		scheduler.Start()
		log.Printf("Scheduler berhasil dimulai dengan %d tugas.", len(scheduler.Entries()))
	} else {
		log.Println("Tidak ada tugas yang berhasil didaftarkan di scheduler. Aplikasi akan berhenti karena tidak ada pekerjaan yang harus dilakukan.")
		return
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Menerima sinyal shutdown, menghentikan scheduler...")
	ctx := scheduler.Stop()
	<-ctx.Done()
	log.Println("Aplikasi berhasil dihentikan.")
}
