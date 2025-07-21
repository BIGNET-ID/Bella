package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"bella/config"
	"bella/db"
	"bella/internal/notifier"
	"bella/setup"

	"github.com/robfig/cron/v3"
)

func main() {
	log.Println("Memulai Bella Alert System (Struktur Final)...")

	config := configs.LoadConfig()
	allConnections := db.InitializeDatabases(config)
	telegramNotifier := notifier.NewTelegramNotifier(config.TelegramToken, config.TelegramChatID)
	scheduler := cron.New()

	setup.RegisterServicesAndTasks(allConnections, telegramNotifier, scheduler, config)
	
	if len(scheduler.Entries()) > 0 {
		scheduler.Start()
		log.Printf("Scheduler berjalan dengan %d tugas.", len(scheduler.Entries()))
	} else {
		log.Println("Tidak ada koneksi database yang aktif. Tidak ada tugas yang dijadwalkan.")
	}
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Menerima sinyal shutdown, menghentikan scheduler...")
	ctx := scheduler.Stop()
	<-ctx.Done()
	log.Println("Aplikasi berhasil dihentikan.")
}
