package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	config "bella/config"
	"bella/db"
	"bella/internal/notifier"
	"bella/setup"

	"github.com/robfig/cron/v3"
)

func main() {
	log.Println("Memulai Bella Alert System (Agent SDK Mode)...")

	// 1. Inisialisasi komponen dasar
	config := config.LoadConfig()
	allConnections := db.InitializeDatabases(config)
	// PERBAIKAN: Mengirim token dan chat ID secara eksplisit
	telegramNotifier := notifier.NewTelegramNotifier(config.TelegramToken, config.TelegramChatID)
	scheduler := cron.New()

	// 2. Panggil satu fungsi untuk mendaftarkan Agent
	setup.RegisterAgentTasks(allConnections, telegramNotifier, scheduler, config)
	
	// 3. Jalankan Scheduler dan tunggu sinyal shutdown
	if len(scheduler.Entries()) > 0 {
		scheduler.Start()
		log.Printf("Scheduler berjalan dengan %d tugas.", len(scheduler.Entries()))
	} else {
		log.Println("Tidak ada tugas yang dijadwalkan.")
	}
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Menerima sinyal shutdown, menghentikan scheduler...")
	ctx := scheduler.Stop()
	<-ctx.Done()
	log.Println("Aplikasi berhasil dihentikan.")
}
