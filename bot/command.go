// ============== File: Bella/bot/command.go ==============

package bot

import (
	"bella/api"
	config "bella/config"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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


