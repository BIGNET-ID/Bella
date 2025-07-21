package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type Notifier interface {
	SendSatnetAlert(gatewayName string, degradedSatnets []DegradedSatnetInfo) error
}

type DegradedSatnetInfo struct {
	Name string
	Fwd  string
	Rtn  string
}

type telegramNotifier struct {
	botToken string
	chatID   string
}

func NewTelegramNotifier(token, chatID string) Notifier {
	return &telegramNotifier{botToken: token, chatID: chatID}
}

func (t *telegramNotifier) determineFriendlyGatewayName(satnetName string) string {
	upperSatnetName := strings.ToUpper(satnetName)
	if strings.HasPrefix(upperSatnetName, "J") || strings.HasPrefix(upperSatnetName, "JYPN") {
		return "JAYAPURA"
	}
	if strings.HasPrefix(upperSatnetName, "M") || strings.HasPrefix(upperSatnetName, "MNKN") {
		return "MANOKWARI"
	}
	if strings.HasPrefix(upperSatnetName, "T") || strings.HasPrefix(upperSatnetName, "TMKN") {
		return "TIMIKA"
	}
	return "GATEWAY TIDAK DIKENALI"
}

func (t *telegramNotifier) SendSatnetAlert(gatewayName string, degradedSatnets []DegradedSatnetInfo) error {
	if len(degradedSatnets) == 0 {
		return nil
	}

	var messageBuilder strings.Builder
	detectionTime := time.Now().Format("2006-01-02 15:04:05 WIB")

	friendlyGatewayName := t.determineFriendlyGatewayName(degradedSatnets[0].Name)

	alertTitle := fmt.Sprintf("🚨 CRITICAL ALERT: %d SATNETS DOWN 🚨", len(degradedSatnets))
	gatewayLine := fmt.Sprintf("GATEWAY: %s", friendlyGatewayName)

	const totalWidth = 44

	createCenteredLine := func(text string) string {
		textLen := utf8.RuneCountInString(text)
		if textLen >= totalWidth {
			return text
		}
		paddingSize := (totalWidth - textLen) / 2
		padding := strings.Repeat(" ", paddingSize)
		return padding + text
	}

	centeredAlertTitle := createCenteredLine(alertTitle)
	centeredGatewayLine := createCenteredLine(gatewayLine)

	// 5. Bangun header
	header := fmt.Sprintf("`%s`\n`%s`\n%s\n\n",
		centeredAlertTitle,
		centeredGatewayLine,
		escapeMarkdownV2("────────────────────────────────"),
	)
	messageBuilder.WriteString(header)

	for _, alarm := range degradedSatnets {
		satnetInfo := fmt.Sprintf(
			"🔻 *SATNET:* %s\n"+
				"   ├─ *Fwd:* %s kbps `(LOW)`\n"+
				"   └─ *Rtn:* %s kbps\n\n",
			escapeMarkdownV2(alarm.Name),
			escapeMarkdownV2(alarm.Fwd),
			escapeMarkdownV2(alarm.Rtn),
		)
		messageBuilder.WriteString(satnetInfo)
	}

	tagLine := "👥 *CC:* @legor1 @legor2 \\(mohon perhatiannya\\)"

	footer := fmt.Sprintf("🕒 *Time of Detection:* %s\n\n%s\n\n*ACTION:* Immediate investigation required\\.",
		escapeMarkdownV2(detectionTime),
		tagLine,
	)
	messageBuilder.WriteString(footer)

	return t.sendMessage(messageBuilder.String())
}

// sendMessage adalah helper privat untuk mengirim request ke API Telegram.
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
		return fmt.Errorf("telegram API Error: %s (status: %d)", body.String(), resp.StatusCode)
	}
	return nil
}

func escapeMarkdownV2(text string) string {
	replacer := strings.NewReplacer("_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-", "=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!")
	return replacer.Replace(text)
}

func centerText(text string, width int) string {
	textLen := utf8.RuneCountInString(text)

	if strings.Contains(text, "🚨") {
		textLen += strings.Count(text, "🚨")
	}

	if textLen >= width {
		return text
	}
	padding := (width - textLen) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + text
}
