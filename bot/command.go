package bot

import (
	"bella/api"
	config "bella/config"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	logFilePath      = "logs/bella.log"
	activeAlertsFile = "logs/active_alerts.json"
	maxLogLines      = 20
	maxFilteredLines = 15
	telegramMaxMsgLen   = 4096
)

type CommandHandler struct {
	bot       *tgbotapi.BotAPI
	config    *config.AppConfig
	apiClient *api.APIClient
}

type GatewayData struct {
	IpcnStatus       *api.IpcnStatusResponse
	IptxTraffic      *api.LnmIptxTrafficResponse
	OnlineUT         *api.ToaRangeIntervalResponse
	IpcnSensors      *api.IpcnSensorStatusResponse
	DeviceProps      *api.DevicePropertiesStatusResponse
	CnBeacon         *api.CnBeaconResponse
	BeamStatus       *api.TerminalBeamStatusResponse
	IntegratedStatus *api.TerminalStatusTotalIntegratedResponse
}

func NewCommandHandler(bot *tgbotapi.BotAPI, config *config.AppConfig, apiClient *api.APIClient) *CommandHandler {
	return &CommandHandler{
		bot:       bot,
		config:    config,
		apiClient: apiClient,
	}
}

func (ch *CommandHandler) sendMessage(chatID int64, text string) {
	if text == "" {
		return
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "MarkdownV2"
	if _, err := ch.bot.Send(msg); err != nil {
		slog.Warn("Gagal mengirim pesan", "error", err, "chat_id", chatID)
	}
}

// getPublicCommands mendefinisikan perintah untuk pengguna biasa.
func getPublicCommands() []tgbotapi.BotCommand {
	return []tgbotapi.BotCommand{
		{Command: "start", Description: "Memulai interaksi dengan bot"},
		{Command: "help", Description: "Menampilkan pesan bantuan ini"},
		{Command: "myid", Description: "Menampilkan ID Telegram Anda"},
	}
}

// getAdminCommands mendefinisikan semua perintah untuk pengguna admin.
func getAdminCommands() []tgbotapi.BotCommand {
	return []tgbotapi.BotCommand{
		{Command: "help", Description: "Tampilkan bantuan"},
		{Command: "myid", Description: "Tampilkan ID Telegram Anda"},
		{Command: "satria1_gateway_all", Description: "Ringkasan status semua Gateway"},
		{Command: "satria1_gateway_jyp", Description: "Ringkasan status Gateway Jayapura"},
		{Command: "satria1_gateway_mnk", Description: "Ringkasan status Gateway Manokwari"},
		{Command: "satria1_gateway_tmk", Description: "Ringkasan status Gateway Timika"},
		{Command: "satria1_iptx_jyp", Description: "Info IP Transit Gateway Jayapura"},
		{Command: "satria1_iptx_mnk", Description: "Info IP Transit Gateway Manokwari"},
		{Command: "satria1_iptx_tmk", Description: "Info IP Transit Gateway Timika"},
		{Command: "log_error", Description: "Tampilkan log error terakhir"},
		{Command: "log_notif", Description: "Tampilkan log notifikasi terakhir"},
		{Command: "log_alerts_active", Description: "Tampilkan alert yang sedang aktif"},
		{Command: "log_all", Description: "Tampilkan semua log terakhir"},
	}
}

func (ch *CommandHandler) sendFile(chatID int64, title, content, fileName string) {
	// Buat file sementara
	tmpFile, err := os.Create(fileName)
	if err != nil {
		slog.Error("Gagal membuat file log sementara", "error", err)
		ch.sendMessage(chatID, escape("Gagal membuat file log untuk dikirim."))
		return
	}
	defer os.Remove(tmpFile.Name()) // Pastikan file dihapus setelah selesai

	_, err = tmpFile.WriteString(content)
	if err != nil {
		slog.Error("Gagal menulis ke file log sementara", "error", err)
		tmpFile.Close()
		return
	}
	tmpFile.Close()

	// Kirim file sebagai dokumen
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(tmpFile.Name()))
	doc.Caption = escape(fmt.Sprintf("Berikut adalah %s", title))

	if _, err := ch.bot.Send(doc); err != nil {
		slog.Warn("Gagal mengirim file log", "error", err, "chat_id", chatID)
	}
}

// HandleLogs menangani semua perintah terkait log.
func (ch *CommandHandler) HandleLogs(chatID int64, command string) {
	slog.Info("Menangani perintah log", "command", command)

	var title, rawContent, fileName string

	switch command {
	case "log_error":
		title = "Log Error Terakhir"
		fileName = "log_error.txt"
		rawContent = ch.readLogFile(maxFilteredLines, "ERROR", "Gagal", "❌")
	case "log_notif":
		title = "Log Notifikasi Terakhir"
		fileName = "log_notif.txt"
		rawContent = ch.readLogFile(maxFilteredLines, "[NOTIFIER]")
	case "log_all":
		title = "Log Mentah Terakhir"
		fileName = "log_all.txt"
		rawContent = ch.readLogFile(maxLogLines)
	case "log_alerts_active":
		title = "Alert yang Sedang Aktif"
		fileName = "active_alerts.json"
		rawContent = ch.readActiveAlerts()
	default:
		ch.sendMessage(chatID, escape("Perintah log tidak dikenal."))
		return
	}

	// Format pesan untuk pengecekan panjang
	fullMessage := FormatLogMessage(title, rawContent)

	if len(fullMessage) > telegramMaxMsgLen {
		// Jika pesan terlalu panjang, kirim sebagai file
		slog.Info("Pesan log terlalu panjang, mengirim sebagai file", "length", len(fullMessage))
		ch.sendFile(chatID, title, rawContent, fileName)
	} else {
		// Jika cukup pendek, kirim sebagai teks biasa
		ch.sendMessage(chatID, fullMessage)
	}
}

// readLogFile sekarang hanya mengembalikan konten mentah.
func (ch *CommandHandler) readLogFile(maxLines int, filters ...string) string {
	file, err := os.Open(logFilePath)
	if err != nil {
		slog.Error("Gagal membuka file log", "path", logFilePath, "error", err)
		return "Error: tidak dapat membuka file log."
	}
	defer file.Close()

	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		slog.Error("Gagal membaca file log", "path", logFilePath, "error", err)
		return "Error: tidak dapat membaca file log."
	}

	var resultLines []string
	if len(filters) > 0 {
		for i := len(allLines) - 1; i >= 0; i-- {
			line := allLines[i]
			for _, filter := range filters {
				if strings.Contains(line, filter) {
					resultLines = append([]string{line}, resultLines...)
					break
				}
			}
			if len(resultLines) >= maxLines {
				break
			}
		}
	} else {
		if len(allLines) > maxLines {
			resultLines = allLines[len(allLines)-maxLines:]
		} else {
			resultLines = allLines
		}
	}

	return strings.Join(resultLines, "\n")
}

// readActiveAlerts sekarang hanya mengembalikan konten mentah.
func (ch *CommandHandler) readActiveAlerts() string {
	content, err := os.ReadFile(activeAlertsFile)
	if err != nil {
		slog.Error("Gagal membaca file alert aktif", "path", activeAlertsFile, "error", err)
		return "Error: tidak dapat membaca file alert."
	}

	if len(content) == 0 || string(content) == "{}" || string(content) == "[]" {
		return "Tidak ada alert yang sedang aktif."
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, content, "", "  "); err != nil {
		return string(content) // Kembalikan mentah jika gagal di-indent
	}

	return prettyJSON.String()
}

func (ch *CommandHandler) fetchGatewayData(gwName string) GatewayData {
	var wg sync.WaitGroup
	var data GatewayData

	var g1gURL, g1kURL, g1lURL, sensorURL string
	switch gwName {
	case "Jayapura":
		g1gURL = ch.config.G1gURL
		g1kURL = ch.config.G1kURL
		sensorURL = g1gURL
	case "Manokwari":
		g1kURL = ch.config.G1kURL
		sensorURL = g1kURL
	case "Timika":
		g1lURL = ch.config.G1lURL
		g1kURL = ch.config.G1kURL
		sensorURL = g1lURL
	}

	ipcnStatusURL := sensorURL

	logApiError := func(taskName string, err error) {
		if err != nil {
			slog.Error("API call failed", "task", taskName, "gateway", gwName, "error", err)
		}
	}

	tasks := []func(){
		func() {
			defer wg.Done()
			var err error
			data.IpcnStatus, err = api.GetIpcnStatus(ch.apiClient, ipcnStatusURL)
			logApiError("IpcnStatus", err)
		},
		func() {
			defer wg.Done()
			var err error
			sdate := time.Now().Add(-5 * time.Minute).Format("2006-01-02-15-04-05")
			edate := time.Now().Format("2006-01-02-15-04-05")
			data.IptxTraffic, err = api.GetIptxTraffic(ch.apiClient, ipcnStatusURL, sdate, edate, "300", strings.ToLower(gwName))
			logApiError("IptxTraffic", err)
		},
		func() {
			defer wg.Done()
			var err error
			data.OnlineUT, err = api.GetOnlineUT(ch.apiClient, g1kURL)
			logApiError("OnlineUT", err)
		},
		func() {
			defer wg.Done()
			var err error
			data.IpcnSensors, err = api.GetIpcnSensorStatus(ch.apiClient, sensorURL, "")
			logApiError("IpcnSensorStatus", err)
		},
		func() {
			defer wg.Done()
			var err error
			data.DeviceProps, err = api.GetDevicePropertiesStatus(ch.apiClient, g1kURL)
			logApiError("DevicePropertiesStatus", err)
		},
		func() {
			defer wg.Done()
			var err error
			data.CnBeacon, err = api.GetCnBeacon(ch.apiClient, g1kURL)
			logApiError("CnBeacon", err)
		},
		func() {
			defer wg.Done()
			var err error
			data.BeamStatus, err = api.GetBeamTerminalStatus(ch.apiClient, g1kURL)
			logApiError("BeamTerminalStatus", err)
		},
		func() {
			defer wg.Done()
			var err error
			data.IntegratedStatus, err = api.GetTerminalStatusTotalIntegrated(ch.apiClient, g1kURL)
			logApiError("TerminalStatusTotalIntegrated", err)
		},
	}

	wg.Add(len(tasks))
	for _, task := range tasks {
		go task()
	}
	wg.Wait()
	return data
}

func (ch *CommandHandler) HandleGatewaySummary(chatID int64, gwName string) {
	slog.Info("Menangani perintah ringkasan gateway", "gateway", gwName)
	ch.sendMessage(chatID, escape(fmt.Sprintf("Mengambil data untuk Gateway %s, mohon tunggu...", gwName)))

	data := ch.fetchGatewayData(gwName)

	jsonBytes, _ := json.MarshalIndent(data.IpcnSensors, "", "  ")
	slog.Debug("Data Sensor IPCN yang diterima dari API", "gateway", gwName, "data", string(jsonBytes))

	response := FormatGatewaySummary(gwName, data)
	ch.sendMessage(chatID, response)
}

func (ch *CommandHandler) HandleGatewayAll(chatID int64) {
	slog.Info("Menangani perintah ringkasan semua gateway")
	ch.sendMessage(chatID, escape("Mengambil data untuk semua gateway, ini mungkin memakan waktu beberapa saat..."))

	var wg sync.WaitGroup
	allData := make(map[string]GatewayData)
	mu := &sync.Mutex{}
	gateways := []string{"Jayapura", "Manokwari", "Timika"}

	wg.Add(len(gateways))
	for _, gw := range gateways {
		go func(gwName string) {
			defer wg.Done()
			data := ch.fetchGatewayData(gwName)
			mu.Lock()
			allData[gwName] = data
			mu.Unlock()
		}(gw)
	}
	wg.Wait()

	var finalReport strings.Builder
	for i, gwName := range gateways {
		data, ok := allData[gwName]
		if !ok {
			finalReport.WriteString(fmt.Sprintf("*Gateway %s*\n_Gagal mengambil data\\._\n\n", escape(gwName)))
			continue
		}

		finalReport.WriteString(FormatGatewayHeader(gwName))
		finalReport.WriteString(formatSystemStatus(data))
		finalReport.WriteString(formatTrafficInfo(data))
		finalReport.WriteString(formatIpcnDeviceSummary(data, gwName))
		finalReport.WriteString(formatModDemod(data))
		finalReport.WriteString(formatSatBeamInfo(data))
		finalReport.WriteString(formatRtgsAiStatus(data))

		if i < len(gateways)-1 {
			finalReport.WriteString("\n" + escape("====================") + "\n\n")
		}
	}

	ch.sendMessage(chatID, finalReport.String())
}

func (ch *CommandHandler) HandleIpTransitInfo(chatID int64, gwName string) {
	slog.Info("Menangani perintah info IP Transit", "gateway", gwName)
	ch.sendMessage(chatID, escape(fmt.Sprintf("Mengambil data IP Transit untuk Gateway %s...", gwName)))

	var g1gURL, g1kURL, g1lURL, ipcnURL string
	switch gwName {
	case "Jayapura":
		g1gURL = ch.config.G1gURL
		g1kURL = ch.config.G1kURL
		ipcnURL = g1gURL
	case "Manokwari":
		g1kURL = ch.config.G1kURL
		ipcnURL = g1kURL
	case "Timika":
		g1lURL = ch.config.G1lURL
		g1kURL = ch.config.G1kURL
		ipcnURL = g1lURL
	}

	var status *api.IpcnStatusResponse
	var traffic *api.LnmIptxTrafficResponse
	var onlineUT *api.ToaRangeIntervalResponse
	var wg sync.WaitGroup

	logApiError := func(taskName string, err error) {
		if err != nil {
			slog.Error("API call failed", "task", taskName, "gateway", gwName, "error", err)
		}
	}

	wg.Add(3)
	go func() {
		defer wg.Done()
		var err error
		status, err = api.GetIpcnStatus(ch.apiClient, ipcnURL)
		logApiError("IpcnStatus (IP Transit)", err)
	}()
	go func() {
		defer wg.Done()
		var err error
		sdate := time.Now().Add(-5 * time.Minute).Format("2006-01-02-15-04-05")
		edate := time.Now().Format("2006-01-02-15-04-05")
		traffic, err = api.GetIptxTraffic(ch.apiClient, ipcnURL, sdate, edate, "300", strings.ToLower(gwName))
		logApiError("IptxTraffic (IP Transit)", err)
	}()
	go func() {
		defer wg.Done()
		var err error
		onlineUT, err = api.GetOnlineUT(ch.apiClient, g1kURL)
		logApiError("OnlineUT (IP Transit)", err)
	}()
	wg.Wait()

	response := FormatIpTransitInfo(gwName, status, traffic, onlineUT)
	ch.sendMessage(chatID, response)
}
