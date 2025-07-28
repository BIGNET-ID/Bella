package prtgn

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	configs "bella/config"
	"bella/internal/notifier"
	"bella/internal/state"
	"bella/internal/types"
)

type PrtgResponse struct {
	SensorData SensorData `json:"sensordata"`
}
type SensorData struct {
	Name             string `json:"name"`
	ParentDeviceName string `json:"parentdevicename"`
	LastValue        string `json:"lastvalue"`
	StatusText       string `json:"statustext"`
	LastCheck        string `json:"lastcheck"`
	LastMessage      string `json:"lastmessage"`
	LastUp           string `json:"lastup"`
	LastDown         string `json:"lastdown"`
}
type PRTGAPIInterface interface {
	RunPeriodicChecks()
}
type PRTGAPI struct {
	BaseURL     string
	APIToken    string
	Notifier    notifier.Notifier
	State       *state.Manager
	NifSensors  map[string]string
	IptxSensors map[string]string
}
func NewPRTGAPI(config *configs.AppConfig, notifier notifier.Notifier, stateMgr *state.Manager) PRTGAPIInterface {
	if config.PRTGUrl == "" || config.PRTGAPITOKEN == "" {
		slog.Warn("Konfigurasi PRTG (URL/Token) hilang. Service PRTG tidak akan berjalan.")
		return nil
	}
	return &PRTGAPI{
		BaseURL:  config.PRTGUrl,
		APIToken: config.PRTGAPITOKEN,
		Notifier: notifier,
		State:    stateMgr,
		NifSensors: map[string]string{
			"JAYAPURA": config.NIF_JYP, "MANOKWARI": config.NIF_MNK, "TIMIKA": config.NIF_TMK,
		},
		IptxSensors: map[string]string{
			"JAYAPURA": config.IPTX_JYP, "MANOKWARI": config.IPTX_MNK, "TIMIKA": config.IPTX_TMK,
		},
	}
}


func (p *PRTGAPI) RunPeriodicChecks() {
	slog.Info("Memulai pengecekan periodik PRTG untuk semua sensor...")

	previousAlerts := p.State.GetActiveAlerts()
	
	allSensors := map[string]map[string]string{
		"NIF":  p.NifSensors,
		"IPTX": p.IptxSensors,
	}

	for sensorType, sensorsMap := range allSensors {
		for location, id := range sensorsMap {
			p.checkSensorAndNotify(location, id, sensorType, previousAlerts)
		}
	}
}

func (p *PRTGAPI) checkSensorAndNotify(location, id, sensorType string, previousAlerts map[string]state.ActiveAlert) {
	const thresholdKbps = 1000.0
	alertKey := fmt.Sprintf("prtg_%s_%s", sensorType, location)

	url := fmt.Sprintf("%s/api/getsensordetails.json?id=%s&apitoken=%s", p.BaseURL, id, p.APIToken)
	resp, err := http.Get(url)
	if err != nil {
		slog.Error("Gagal request ke PRTG", "sensor", location, "error", err)
		return
	}
	defer resp.Body.Close()

	var prtgResp PrtgResponse
	if err := json.NewDecoder(resp.Body).Decode(&prtgResp); err != nil {
		slog.Error("Gagal parsing JSON dari PRTG", "sensor", location, "error", err)
		return
	}

	sensorData := prtgResp.SensorData
	isCurrentlyDown := false
	alertValue := sensorData.LastValue

	if strings.EqualFold(sensorData.StatusText, "Down") {
		isCurrentlyDown = true
	} else {
		valueKbps, err := p.parseAndConvertValue(sensorData.LastValue)
		if err == nil && valueKbps < thresholdKbps {
			isCurrentlyDown = true
			alertValue = fmt.Sprintf("%.2f Kbit/s", valueKbps)
		}
	}

	_, wasPreviouslyDown := previousAlerts[alertKey]

	if isCurrentlyDown && !wasPreviouslyDown {
		slog.Warn("Alert PRTG BARU terdeteksi", "key", alertKey)
		alertData := p.createDownAlert(location, sensorType, sensorData, alertValue)
		p.sendDownAlert(alertData)
		p.State.AddAlert(alertKey, state.ActiveAlert{
			Type: "prtg", Gateway: location, Details: alertData,
		})
	} else if !isCurrentlyDown && wasPreviouslyDown {
		slog.Info("Sensor PRTG terdeteksi PULIH", "key", alertKey)
		upAlert := types.PRTGUpAlert{
			Location:       p.Notifier.DetermineFriendlyGatewayName(location),
			SensorFullName: sensorData.Name,
			DeviceName:     sensorData.ParentDeviceName,
			SensorType:     sensorType,
			RecoveryTime:   time.Now(),
		}
		if err := p.Notifier.SendPrtgUpAlert(upAlert); err != nil {
			slog.Error("Gagal mengirim notifikasi pemulihan PRTG", "key", alertKey, "error", err)
		}
		p.State.RemoveAlertByKey(alertKey)
	}
}

func (p *PRTGAPI) createDownAlert(location, sensorType string, sensorData SensorData, value string) types.PRTGDownAlert {
	return types.PRTGDownAlert{
		Location:       p.Notifier.DetermineFriendlyGatewayName(location),
		SensorFullName: sensorData.Name,
		DeviceName:     sensorData.ParentDeviceName,
		SensorType:     sensorType,
		Status:         sensorData.StatusText,
		Value:          value,
		LastMessage:    sensorData.LastMessage,
		LastCheck:      p.extractTimeAgo(sensorData.LastCheck),
		LastUp:         p.extractTimeAgo(sensorData.LastUp),
		LastDown:       p.extractTimeAgo(sensorData.LastDown),
	}
}
func (p *PRTGAPI) sendDownAlert(alertData types.PRTGDownAlert) {
	var err error
	switch alertData.SensorType {
	case "NIF":
		err = p.Notifier.SendPrtgNIFDownAlert(alertData)
	case "IPTX":
		err = p.Notifier.SendPrtgTrafficDownAlert(alertData)
	}
	if err != nil {
		slog.Error("Gagal mengirim notifikasi PRTG", "sensor_type", alertData.SensorType, "error", err)
	}
}
func (p *PRTGAPI) parseAndConvertValue(valueStr string) (float64, error) {
	re := regexp.MustCompile(`[0-9]+(?:\.[0-9]+)?`)
	numberPart := re.FindString(valueStr)
	if numberPart == "" {
		return 0, fmt.Errorf("tidak ada bagian numerik yang ditemukan di '%s'", valueStr)
	}
	value, err := strconv.ParseFloat(numberPart, 64)
	if err != nil {
		return 0, fmt.Errorf("gagal konversi nilai numerik '%s': %w", numberPart, err)
	}
	unitStrLower := strings.ToLower(valueStr)
	switch {
	case strings.Contains(unitStrLower, "mbit/s"):
		return value * 1000, nil
	case strings.Contains(unitStrLower, "kbit/s"):
		return value, nil
	case strings.Contains(unitStrLower, "bit/s"):
		return value / 1000, nil
	default:
		return value, nil
	}
}
func (p *PRTGAPI) extractTimeAgo(fullString string) string {
	re := regexp.MustCompile(`\[(.*?)\]`)
	matches := re.FindStringSubmatch(fullString)
	if len(matches) > 1 {
		return matches[1]
	}
	if fullString != "-" {
		return fullString
	}
	return ""
}
