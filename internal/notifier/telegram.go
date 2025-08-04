package notifier

import (
	"bella/internal/types"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Notifier interface {
	SendSatnetAlert(report types.GatewayReport) error
	SendSatnetUpAlert(alerts []types.SatnetUpAlert) error
	DetermineFriendlyGatewayName(gatewayName string) string
	SendPrtgTrafficDownAlert(traffic types.PRTGDownAlert) error
	SendPrtgNIFDownAlert(traffic types.PRTGDownAlert) error
	SendPrtgUpAlert(alert types.PRTGUpAlert) error
	SendModemDownAlert(alerts []types.ModemDownAlert, deviceType string) error
	SendModemUpAlert(alerts []types.ModemUpAlert, deviceType string) error
}

type telegramNotifier struct {
	botToken string
	chatID   string
}

func NewTelegramNotifier(token, chatID string) Notifier {
	return &telegramNotifier{botToken: token, chatID: chatID}
}

func (t *telegramNotifier) sendMessage(text string) error {
	payload := map[string]string{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "MarkdownV2",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling payload: %w", err)
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var body bytes.Buffer
		body.ReadFrom(resp.Body)
		log.Printf("❌ [NOTIFIER] Gagal mengirim ke Telegram! Status: %d, Pesan: %s", resp.StatusCode, body.String())
		return fmt.Errorf("telegram API Error: %s (status: %d)", body.String(), resp.StatusCode)
	}
	log.Println("✅ [NOTIFIER] Pesan berhasil dikirim ke Telegram.")
	return nil
}

func escapeMarkdownV2(text string) string {
	replacer := strings.NewReplacer("_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-", "=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!")
	return replacer.Replace(text)
}

func (t *telegramNotifier) DetermineFriendlyGatewayName(gatewayName string) string {
	upperName := strings.ToUpper(gatewayName)
	if strings.Contains(upperName, "JYP") {
		return "JAYAPURA"
	}
	if strings.Contains(upperName, "MNK") {
		return "MANOKWARI"
	}
	if strings.Contains(upperName, "TMK") {
		return "TIMIKA"
	}
	return gatewayName
}

func AlarmStateToEmoji(state string) string {
    switch strings.ToLower(state) {
    case "major":
        return "🟨"
    case "critical":
        return "🟥"
    case "timeout":
        return "🟥"
    default:
        return state
    }
}
func (t *telegramNotifier) SendSatnetAlert(report types.GatewayReport) error {
	if len(report.Satnets) == 0 {
		return nil
	}

	var messageBuilder strings.Builder
	friendlyGatewayName := t.DetermineFriendlyGatewayName(report.FriendlyName)

	alertTitle := "🔴 *CRITICAL ALERT*"
	eventLine := fmt.Sprintf("🍎 *EVENT:* %d SATNETS DOWN", len(report.Satnets))
	gatewayLine := fmt.Sprintf("🔰 *GATEWAY:* %s", escapeMarkdownV2(friendlyGatewayName))
	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n",
		alertTitle,
		eventLine,
		gatewayLine,
		escapeMarkdownV2("───────────────"),
	)
	messageBuilder.WriteString(header)

	for _, satnet := range report.Satnets {
		onlineStr := "0"
		if satnet.OnlineCount != nil {
			onlineStr = fmt.Sprintf("%d", *satnet.OnlineCount)
		}

		offlineStr := "0"
		if satnet.OfflineCount != nil {
			offlineStr = fmt.Sprintf("%d", *satnet.OfflineCount)
		}

		startIssueStr := "N/A"
		if satnet.StartIssue != nil {
			startIssueStr = escapeMarkdownV2(satnet.StartIssue.Format("2006-01-02 15:04:05 WIB"))
		}

		fwdStr := escapeMarkdownV2(fmt.Sprintf("%.2f", satnet.FwdTp))
		rtnStr := escapeMarkdownV2(fmt.Sprintf("%.2f", satnet.RtnTp))

		satnetInfo := fmt.Sprintf(
			"  🟥 *SATNET:* %s\n"+
				"   ├─ *FWD :* %s kbps `(LOW)`\n"+
				"   ├─ *RTN :* %s kbps\n"+
				"   ├─ *Online UT :* %s\n"+
				"   ├─ *Offline UT :* %s\n"+
				"   └─ *Start :* %s\n\n",
			escapeMarkdownV2(satnet.Name),
			fwdStr,
			rtnStr,
			onlineStr,
			offlineStr,
			startIssueStr,
		)
		messageBuilder.WriteString(satnetInfo)
	}

	footer := "*ACTION:* Immediate investigation required\\."
	messageBuilder.WriteString(footer)

	return t.sendMessage(messageBuilder.String())
}

func (t *telegramNotifier) SendSatnetUpAlert(alerts []types.SatnetUpAlert) error {
	if len(alerts) == 0 {
		return nil
	}

	var messageBuilder strings.Builder
	friendlyGatewayName := t.DetermineFriendlyGatewayName(alerts[0].GatewayName)

	title := "🟢 *RECOVERY INFO*"
	eventLine := fmt.Sprintf("🍏 *EVENT:* %d SATNETS UP", len(alerts))
	gatewayLine := fmt.Sprintf("🔰 *GATEWAY:* %s", escapeMarkdownV2(friendlyGatewayName))
	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n",
		title,
		eventLine,
		gatewayLine,
		escapeMarkdownV2("───────────────"),
	)
	messageBuilder.WriteString(header)

	for _, alert := range alerts {
		timestamp := alert.RecoveryTime.Format("2006-01-02 15:04:05 WIB")
		line := fmt.Sprintf("  🟩 *SATNET:* %s\n   └─ *RECOVERED AT:* %s\n\n",
			escapeMarkdownV2(alert.SatnetName),
			escapeMarkdownV2(timestamp),
		)
		messageBuilder.WriteString(line)
	}

	return t.sendMessage(messageBuilder.String())
}

func (t *telegramNotifier) SendPrtgTrafficDownAlert(traffic types.PRTGDownAlert) error {
	var messageBuilder strings.Builder

	alertTitle := "🔴 *CRITICAL ALERT*"
	eventLine := "🍎 *EVENT:* IPTX TRAFFIC LOW"
	gatewayLine := fmt.Sprintf("🔰 *GATEWAY:* %s", escapeMarkdownV2(traffic.Location))
	separator := escapeMarkdownV2("───────────────")

	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n", alertTitle, eventLine, gatewayLine, separator)
	messageBuilder.WriteString(header)

	deviceLine := fmt.Sprintf("  🟥 *DEVICE:* %s\n", escapeMarkdownV2(traffic.DeviceName))
	sensorLine := fmt.Sprintf("   ├─ *SENSOR:* %s\n", escapeMarkdownV2(traffic.SensorFullName))
	valueLine := fmt.Sprintf("   ├─ *VALUE :* %s `(LOW)`\n", escapeMarkdownV2(traffic.Value))
	lastDownLine := fmt.Sprintf("   └─ *LAST UP :* %s\n\n", escapeMarkdownV2(traffic.LastUp))

	messageBuilder.WriteString(deviceLine)
	messageBuilder.WriteString(sensorLine)
	messageBuilder.WriteString(valueLine)
	messageBuilder.WriteString(lastDownLine)

	footer := fmt.Sprintf("_Last checked: %s_", escapeMarkdownV2(traffic.LastCheck))
	messageBuilder.WriteString(footer)

	return t.sendMessage(messageBuilder.String())
}

func (t *telegramNotifier) SendPrtgNIFDownAlert(nif types.PRTGDownAlert) error {
	var messageBuilder strings.Builder

	alertTitle := "🔴 *CRITICAL ALERT*"
	eventLine := "🍎 *EVENT:* NIF TRAFFIC LOW"
	gatewayLine := fmt.Sprintf("🔰 *GATEWAY:* %s", escapeMarkdownV2(nif.Location))
	separator := escapeMarkdownV2("───────────────")

	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n", alertTitle, eventLine, gatewayLine, separator)
	messageBuilder.WriteString(header)

	deviceLine := fmt.Sprintf("  🟥 *DEVICE:* %s\n", escapeMarkdownV2(nif.DeviceName))
	sensorLine := fmt.Sprintf("   ├─ *SENSOR:* %s\n", escapeMarkdownV2(nif.SensorFullName))
	valueLine := fmt.Sprintf("   ├─ *VALUE :* %s `(LOW)`\n", escapeMarkdownV2(nif.Value))
	lastDownLine := fmt.Sprintf("   └─ *LAST UP :* %s\n\n", escapeMarkdownV2(nif.LastUp))

	messageBuilder.WriteString(deviceLine)
	messageBuilder.WriteString(sensorLine)
	messageBuilder.WriteString(valueLine)
	messageBuilder.WriteString(lastDownLine)

	footer := fmt.Sprintf("_Last checked: %s_", escapeMarkdownV2(nif.LastCheck))
	messageBuilder.WriteString(footer)

	return t.sendMessage(messageBuilder.String())
}

func (t *telegramNotifier) SendPrtgUpAlert(alert types.PRTGUpAlert) error {
	var messageBuilder strings.Builder

	title := "🟢 *RECOVERY INFO*"
	eventType := fmt.Sprintf("🍏 *EVENT:* %s RECOVERED", escapeMarkdownV2(alert.SensorType))
	gatewayLine := fmt.Sprintf("🔰 *GATEWAY:* %s", escapeMarkdownV2(alert.Location))
	separator := escapeMarkdownV2("───────────────")

	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n", title, eventType, gatewayLine, separator)
	messageBuilder.WriteString(header)

	deviceLine := fmt.Sprintf("  🟩 *DEVICE:* %s\n", escapeMarkdownV2(alert.DeviceName))
	sensorLine := fmt.Sprintf("   ├─ *SENSOR:* %s\n", escapeMarkdownV2(alert.SensorFullName))
	recoveryLine := fmt.Sprintf("   └─ *RECOVERED AT:* %s", escapeMarkdownV2(alert.RecoveryTime.Format("2006-01-02 15:04:05 WIB")))

	messageBuilder.WriteString(deviceLine)
	messageBuilder.WriteString(sensorLine)
	messageBuilder.WriteString(recoveryLine)

	return t.sendMessage(messageBuilder.String())
}

func (t *telegramNotifier) SendModemDownAlert(alerts []types.ModemDownAlert, deviceType string) error {
	if len(alerts) == 0 {
		return nil
	}
	var messageBuilder strings.Builder
	friendlyGatewayName := t.DetermineFriendlyGatewayName(alerts[0].GatewayName)
	deviceTypeUpper := strings.ToUpper(deviceType) + "S"

	alertTitle := "🔴 *CRITICAL ALERT*"
	eventLine := fmt.Sprintf("🍎 *EVENT:* %d %s DOWN", len(alerts), escapeMarkdownV2(deviceTypeUpper))
	gatewayLine := fmt.Sprintf("🔰 *GATEWAY:* %s", escapeMarkdownV2(friendlyGatewayName))
	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n", alertTitle, eventLine, gatewayLine, escapeMarkdownV2("───────────────"))
	messageBuilder.WriteString(header)

	for _, alert := range alerts {
		startTime := alert.StartTime.Format("2006-01-02 15:04:05 WIB")

		alarmState := "Unknown"
		if alert.AlarmState != "" {
			alarmState = alert.AlarmState
		}

		emoji := AlarmStateToEmoji(alarmState)

		info := fmt.Sprintf(
			"  %s *DEVICE:* %s\n"+
				"   ├─ *ALARM STATE :* %s\n"+
				"   └─ *Start :* %s\n\n",
			escapeMarkdownV2(emoji),
			escapeMarkdownV2(alert.DeviceName),
			escapeMarkdownV2(alarmState),
			escapeMarkdownV2(startTime),
		)
		messageBuilder.WriteString(info)
	}
	messageBuilder.WriteString("*ACTION:* Immediate investigation required\\.")
	return t.sendMessage(messageBuilder.String())
}

func (t *telegramNotifier) SendModemUpAlert(alerts []types.ModemUpAlert, deviceType string) error {
	if len(alerts) == 0 {
		return nil
	}
	var messageBuilder strings.Builder
	friendlyGatewayName := t.DetermineFriendlyGatewayName(alerts[0].GatewayName)
	deviceTypeUpper := strings.ToUpper(deviceType) + "S"

	title := "🟢 *RECOVERY INFO*"
	eventLine := fmt.Sprintf("🍏 *EVENT:* %d %s UP", len(alerts), escapeMarkdownV2(deviceTypeUpper))
	gatewayLine := fmt.Sprintf("🔰 *GATEWAY:* %s", escapeMarkdownV2(friendlyGatewayName))
	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n", title, eventLine, gatewayLine, escapeMarkdownV2("───────────────"))
	messageBuilder.WriteString(header)

	for _, alert := range alerts {
		recoveryTime := alert.RecoveryTime.Format("2006-01-02 15:04:05 WIB")
		info := fmt.Sprintf(
			"  🟩 *DEVICE:* %s\n"+
				"   └─ *RECOVERED AT:* %s\n\n",
			escapeMarkdownV2(alert.DeviceName),
			escapeMarkdownV2(recoveryTime),
		)
		messageBuilder.WriteString(info)
	}
	return t.sendMessage(messageBuilder.String())
}
