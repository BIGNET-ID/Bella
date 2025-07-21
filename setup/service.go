package setup

import (
	"log"
	"strings" // <-- Import paket strings
	"bella/config"
	"bella/db"
	"bella/internal/notifier"
	"bella/internal/satnet"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// RegisterServicesAndTasks menginisialisasi semua service dan mendaftarkan tugas terjadwal.
func RegisterServicesAndTasks(
	allConnections *db.Connections,
	notifier notifier.Notifier,
	scheduler *cron.Cron,
	config *configs.AppConfig,
) {
	log.Println("Mendaftarkan service dan tugas terjadwal...")

	// Helper function untuk mendaftarkan service
	register := func(dbConn *gorm.DB, name string) {
		if dbConn == nil {
			return // Lewati jika koneksi tidak aktif
		}

		if strings.HasPrefix(name, "DB_ONE") {
			// Buat service untuk fitur satnet
			satnetSvc := satnet.NewService(dbConn, notifier, name)
			scheduler.AddFunc(config.CronSchedule, satnetSvc.CheckAndAlert)
			log.Printf("Service Satnet untuk '%s' berhasil didaftarkan.", name)
		}

		log.Printf("Inisialisasi dasar untuk '%s' selesai.", name)
	}

	// Panggil helper untuk setiap koneksi
	register(allConnections.DBOneJYP, "DB_ONE_JYP")
	register(allConnections.DBOneMNK, "DB_ONE_MNK")
	register(allConnections.DBOneTMK, "DB_ONE_TMK")
	register(allConnections.DBFiveJYP, "DB_FIVE_JYP")
	register(allConnections.DBFiveMNK, "DB_FIVE_MNK")
	register(allConnections.DBFiveTMK, "DB_FIVE_TMK")
}
