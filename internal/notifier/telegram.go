package notifier

import (
	"bella/internal/types"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"
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
	case "minor":
		return "🐤"
	case "major":
		return "🙉"
	case "critical":
		return "🐶"
	case "timeout":
		return "🐷"
	default:
		return state
	}
}

func formatDuration(start time.Time) string {
	const layout = "2006-01-02T15:04:05"

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Printf("Gagal load timezone: %v", err)
		return "Invalid Timezone"
	}

	now := time.Now().In(loc)

	nowStr := now.Format(layout)
	startStr := start.Format(layout)

	nowClean, _ := time.ParseInLocation(layout, nowStr, loc)
	startClean, _ := time.ParseInLocation(layout, startStr, loc)

	duration := nowClean.Sub(startClean)

	if duration < 0 {
		duration = 0
	}

	seconds := int(duration.Seconds())

	if seconds < 60 {
		if seconds <= 1 {
			return "1 second"
		}
		return fmt.Sprintf("%d seconds", seconds)
	}

	minutes := seconds / 60
	if minutes < 60 {
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}

	hours := minutes / 60
	if hours < 24 {
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}

	days := hours / 24
	if days < 30 {
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}

	months := days / 30
	if months < 12 {
		if months == 1 {
			return "1 month"
		}
		return fmt.Sprintf("%d months", months)
	}

	years := days / 365
	if years == 1 {
		return "1 year"
	}
	return fmt.Sprintf("%d years", years)
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "S"
}

func (t *telegramNotifier) SendSatnetAlert(report types.GatewayReport) error {
	if len(report.Satnets) == 0 {
		return nil
	}

	var messageBuilder strings.Builder
	friendlyGatewayName := t.DetermineFriendlyGatewayName(report.FriendlyName)
	count := len(report.Satnets)

	alertTitle := "🚨 *CRITICAL ALERT* 🚨"
	eventLine := fmt.Sprintf("🗒 EVENT : *%d SATNET%s DOWN 🐶*", count, pluralSuffix(count))
	gatewayLine := fmt.Sprintf("📡 GATEWAY : *%s*", escapeMarkdownV2(friendlyGatewayName))
	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n",
		alertTitle,
		eventLine,
		gatewayLine,
		escapeMarkdownV2("━━━━━━━ ✦ ━━━━━━━"),
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
			startIssueStr = satnet.StartIssue.Format("2006/01/02 15:04")
		}

		durationStr := escapeMarkdownV2("N/A")
		if satnet.StartIssue != nil {
			durationStr = formatDuration(*satnet.StartIssue)
		}

		fwdStr := escapeMarkdownV2(fmt.Sprintf("%.2f", satnet.FwdTp))
		rtnStr := escapeMarkdownV2(fmt.Sprintf("%.2f", satnet.RtnTp))

		satnetInfo := fmt.Sprintf(
			"   *SATNET:* %s\n"+
				"   ├─ *FWD :* `%s kbps` *\\(LOW\\)*\n"+
				"   ├─ *RTN :* `%s kbps`\n"+
				"   ├─ *Online UT :* `%s`\n"+
				"   ├─ *Offline UT :* `%s`\n"+
				"   ├─ *Start :* `%s`\n"+
				"   └─ *Duration :* `%s`\n\n",
			escapeMarkdownV2(satnet.Name),
			fwdStr,
			rtnStr,
			onlineStr,
			offlineStr,
			startIssueStr,
			escapeMarkdownV2(durationStr),
		)
		messageBuilder.WriteString(satnetInfo)
	}

	return t.sendMessage(messageBuilder.String())
}

func (t *telegramNotifier) SendSatnetUpAlert(alerts []types.SatnetUpAlert) error {
	if len(alerts) == 0 {
		return nil
	}

	var messageBuilder strings.Builder
	friendlyGatewayName := t.DetermineFriendlyGatewayName(alerts[0].GatewayName)
	count := len(alerts)

	title := "🌟 *RECOVERY INFO* 🌟"
	eventLine := fmt.Sprintf("🗒 EVENT : *%d SATNET%s UP*", count, pluralSuffix(count))
	gatewayLine := fmt.Sprintf("📡 GATEWAY : *%s*", escapeMarkdownV2(friendlyGatewayName))
	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n",
		title,
		eventLine,
		gatewayLine,
		escapeMarkdownV2("━━━━━━━ ✦ ━━━━━━━"),
	)
	messageBuilder.WriteString(header)

	for _, alert := range alerts {
		timestamp := alert.RecoveryTime.Format("2006/01/02 15:04")
		durationStr := formatDuration(alert.TimeDown)

		line := fmt.Sprintf("  🛟 *SATNET :* `%s`\n   ├─ *RECOVERED AT:* `%s`\n   └─ *DURATION:* `%s`\n\n",
			escapeMarkdownV2(alert.SatnetName),
			escapeMarkdownV2(timestamp),
			escapeMarkdownV2(durationStr),
		)
		messageBuilder.WriteString(line)
	}

	return t.sendMessage(messageBuilder.String())
}

func (t *telegramNotifier) SendPrtgTrafficDownAlert(traffic types.PRTGDownAlert) error {
	var messageBuilder strings.Builder

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.Local
	}

	const lastCheckLayout = "2006-01-02 15:04:05 MST"
	var formattedLastCheck string

	if parsedCheckTime, err := time.ParseInLocation(lastCheckLayout, traffic.LastCheck, loc); err == nil {
		formattedLastCheck = parsedCheckTime.Format("2006/01/02 15:04")
	} else {
		formattedLastCheck = traffic.LastCheck
		slog.Warn("Gagal parse LastCheck", "raw", traffic.LastCheck, "err", err)
	}

	alertTitle := "🚨 *CRITICAL ALERT* 🚨"
	eventLine := "🗒 EVENT : *IPTX TRAFFIC LOW*"
	gatewayLine := fmt.Sprintf("📡 GATEWAY : *%s*", escapeMarkdownV2(traffic.Location))
	lastCheck := fmt.Sprintf("🕐 LAST CHECKED : *%s*", escapeMarkdownV2(formattedLastCheck))
	separator := escapeMarkdownV2("━━━━━━━ ✦ ━━━━━━━")

	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s\n\n", alertTitle, eventLine, gatewayLine, lastCheck, separator)
	messageBuilder.WriteString(header)

	const lastUpLayout = "2006-01-02 15:04:05 MST"

	var durationStr string
	var timeLast string
	if lastUpTime, err := time.ParseInLocation(lastUpLayout, traffic.LastUp, loc); err == nil {
		durationStr = formatDuration(lastUpTime)
		timeLast = lastUpTime.Format("2006/01/02 15:04")
	} else {
		durationStr = "N/A"
		timeLast = "N/A"
		slog.Warn("Gagal parse LastUp", "raw", traffic.LastUp, "err", err)
	}

	deviceLine := fmt.Sprintf("   *DEVICE :* `%s`\n", escapeMarkdownV2(traffic.DeviceName))
	sensorLine := fmt.Sprintf("   ├─ *SENSOR :* `%s`\n", escapeMarkdownV2(traffic.SensorFullName))
	valueLine := fmt.Sprintf("   ├─ *VALUE :* `%s` *\\(LOW\\)*\n", escapeMarkdownV2(traffic.Value))
	lastDownLine := fmt.Sprintf("   ├─ *LAST UP :* `%s`\n", escapeMarkdownV2(timeLast))
	durationLine := fmt.Sprintf("   └─ *DURATION :* `%s`\n\n", escapeMarkdownV2(durationStr))

	messageBuilder.WriteString(deviceLine)
	messageBuilder.WriteString(sensorLine)
	messageBuilder.WriteString(valueLine)
	messageBuilder.WriteString(lastDownLine)
	messageBuilder.WriteString(durationLine)

	return t.sendMessage(messageBuilder.String())
}

func (t *telegramNotifier) SendPrtgNIFDownAlert(nif types.PRTGDownAlert) error {
	var messageBuilder strings.Builder

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.Local
	}

	const lastCheckLayout = "2006-01-02 15:04:05 MST"
	var formattedLastCheck string

	if parsedCheckTime, err := time.ParseInLocation(lastCheckLayout, nif.LastCheck, loc); err == nil {
		formattedLastCheck = parsedCheckTime.Format("2006/01/02 15:04")
	} else {
		formattedLastCheck = nif.LastCheck
		slog.Warn("Gagal parse LastCheck", "raw", nif.LastCheck, "err", err)
	}

	alertTitle := "🚨 *CRITICAL ALERT* 🚨"
	eventLine := "🗒 EVENT : *NIF TRAFFIC LOW*"
	gatewayLine := fmt.Sprintf("📡 GATEWAY : *%s*", escapeMarkdownV2(nif.Location))
	lastCheck := fmt.Sprintf("🕐 LAST CHECKED : *%s*", escapeMarkdownV2(formattedLastCheck))

	separator := escapeMarkdownV2("━━━━━━━ ✦ ━━━━━━━")

	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s\n\n", alertTitle, eventLine, gatewayLine, lastCheck, separator)
	messageBuilder.WriteString(header)

	const lastUpLayout = "2006-01-02 15:04:05 MST"

	var durationStr string
	var timeLast string
	if lastUpTime, err := time.ParseInLocation(lastUpLayout, nif.LastUp, loc); err == nil {
		durationStr = formatDuration(lastUpTime)
		timeLast = lastUpTime.Format("2006/01/02 15:04")
	} else {
		durationStr = "N/A"
		timeLast = "N/A"
		slog.Warn("Gagal parse LastUp", "raw", nif.LastUp, "err", err)
	}

	deviceLine := fmt.Sprintf("   *DEVICE :* `%s`\n", escapeMarkdownV2(nif.DeviceName))
	sensorLine := fmt.Sprintf("   ├─ *SENSOR :* `%s`\n", escapeMarkdownV2(nif.SensorFullName))
	valueLine := fmt.Sprintf("   ├─ *VALUE :* `%s` *\\(LOW\\)*\n", escapeMarkdownV2(nif.Value))
	lastDownLine := fmt.Sprintf("   ├─ *LAST UP :* `%s`\n", escapeMarkdownV2(timeLast))
	durationLine := fmt.Sprintf("   └─ *DURATION :* `%s`\n\n", escapeMarkdownV2(durationStr))

	messageBuilder.WriteString(deviceLine)
	messageBuilder.WriteString(sensorLine)
	messageBuilder.WriteString(valueLine)
	messageBuilder.WriteString(lastDownLine)
	messageBuilder.WriteString(durationLine)

	return t.sendMessage(messageBuilder.String())
}

func (t *telegramNotifier) SendPrtgUpAlert(alert types.PRTGUpAlert) error {
	var messageBuilder strings.Builder

	title := "🌟 *RECOVERY INFO* 🌟"
	eventType := fmt.Sprintf("🗒 EVENT : *%s RECOVERED*", escapeMarkdownV2(alert.SensorType))
	gatewayLine := fmt.Sprintf("📡 GATEWAY : *%s*", escapeMarkdownV2(alert.Location))
	separator := escapeMarkdownV2("━━━━━━━ ✦ ━━━━━━━")

	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n", title, eventType, gatewayLine, separator)
	messageBuilder.WriteString(header)

	lastDown := alert.LastDown
	durationStr := formatDuration(lastDown)

	deviceLine := fmt.Sprintf("  🛟 *DEVICE :* `%s`\n", escapeMarkdownV2(alert.DeviceName))
	sensorLine := fmt.Sprintf("   ├─ *SENSOR :* `%s`\n", escapeMarkdownV2(alert.SensorFullName))
	recoveryLine := fmt.Sprintf("   ├─ *RECOVERED AT :* `%s`\n", escapeMarkdownV2(alert.RecoveryTime.Format("2006/01/02 15:04")))
	durationLine := fmt.Sprintf("   └─ *DURATION:* `%s`", escapeMarkdownV2(durationStr))

	messageBuilder.WriteString(deviceLine)
	messageBuilder.WriteString(sensorLine)
	messageBuilder.WriteString(recoveryLine)
	messageBuilder.WriteString(durationLine)

	return t.sendMessage(messageBuilder.String())
}

func (t *telegramNotifier) SendModemDownAlert(alerts []types.ModemDownAlert, deviceType string) error {
	if len(alerts) == 0 {
		return nil
	}
	var messageBuilder strings.Builder
	friendlyGatewayName := t.DetermineFriendlyGatewayName(alerts[0].GatewayName)
	count := len(alerts)
	deviceTypeUpper := strings.ToUpper(deviceType)

	alertTitle := "🚨 *ALARM ALERT* 🚨"
	eventLine := fmt.Sprintf("🗒 EVENT : *%d %s%s ALARM ALERT*", count, escapeMarkdownV2(deviceTypeUpper), escapeMarkdownV2(pluralSuffix(count)))
	gatewayLine := fmt.Sprintf("📡 GATEWAY : *%s*", escapeMarkdownV2(friendlyGatewayName))
	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n", alertTitle, eventLine, gatewayLine, escapeMarkdownV2("━━━━━━━ ✦ ━━━━━━━"))
	messageBuilder.WriteString(header)

	for _, alert := range alerts {
		durationStr := formatDuration(alert.StartTime)

		startTime := alert.StartTime.Format("2006/01/02 15:04")

		alarmState := "Unknown"
		if alert.AlarmState != "" {
			alarmState = alert.AlarmState
		}

		emoji := AlarmStateToEmoji(alarmState)

		info := fmt.Sprintf(
			"  %s *DEVICE :* `%s`\n"+
				"   ├─ *ALARM STATE :* `%s`\n"+
				"   ├─ *START :* `%s`\n"+
				"   └─ *DURATION :* `%s`\n\n",

			escapeMarkdownV2(emoji),
			escapeMarkdownV2(alert.DeviceName),
			escapeMarkdownV2(alarmState),
			escapeMarkdownV2(startTime),
			escapeMarkdownV2(durationStr),
		)
		messageBuilder.WriteString(info)
	}
	return t.sendMessage(messageBuilder.String())
}

func (t *telegramNotifier) SendModemUpAlert(alerts []types.ModemUpAlert, deviceType string) error {
	if len(alerts) == 0 {
		return nil
	}
	var messageBuilder strings.Builder
	friendlyGatewayName := t.DetermineFriendlyGatewayName(alerts[0].GatewayName)
	count := len(alerts)
	deviceTypeUpper := strings.ToUpper(deviceType)

	title := "🌟 *RECOVERY INFO* 🌟"
	eventLine := fmt.Sprintf("🗒 EVENT : *%d %s%s RECOVERED*", count, escapeMarkdownV2(deviceTypeUpper), escapeMarkdownV2(pluralSuffix(count)))
	gatewayLine := fmt.Sprintf("📡 GATEWAY : *%s*", escapeMarkdownV2(friendlyGatewayName))
	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n", title, eventLine, gatewayLine, escapeMarkdownV2("━━━━━━━ ✦ ━━━━━━━"))
	messageBuilder.WriteString(header)

	for _, alert := range alerts {
		recoveryTime := alert.RecoveryTime.Format("2006/01/02 15:04")
		info := fmt.Sprintf(
			"  🛟 *DEVICE :* `%s`\n"+
				"   ├─ *RECOVERED AT :* `%s`\n"+
				"   └─ *DURATION :* `%s`\n\n",
			escapeMarkdownV2(alert.DeviceName),
			escapeMarkdownV2(recoveryTime),
			escapeMarkdownV2(formatDuration(alert.TimeDown)),
		)
		messageBuilder.WriteString(info)
	}
	return t.sendMessage(messageBuilder.String())
}
