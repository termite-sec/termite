package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Finding struct {
	File     string
	Line     int
	Type     string
	Severity string
	Message  string
	Fix      string
}

type ScanReport struct {
	ScanPath  string
	FileCount int
	Findings  []Finding
	Exposure  string
	High      int
	Med       int
	Low       int
	ScannedAt time.Time
}

func SendSlack(webhookURL string, report ScanReport) error {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return fmt.Errorf("no slack webhook url")
	}

	if report.ScannedAt.IsZero() {
		report.ScannedAt = time.Now()
	}

	payload := buildSlackPayload(report)
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned %d", resp.StatusCode)
	}
	return nil
}

func buildSlackPayload(r ScanReport) map[string]interface{} {
	issueCount := len(r.Findings)
	header := fmt.Sprintf("kitin scan — %d issue(s)", issueCount)
	if issueCount == 0 {
		header = "kitin scan — no issues found"
	}

	blocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]string{"type": "plain_text", "text": header},
		},
		{
			"type": "section",
			"fields": []map[string]string{
				{"type": "mrkdwn", "text": fmt.Sprintf("*Exposure:*\n%s", r.Exposure)},
				{"type": "mrkdwn", "text": fmt.Sprintf("*Files:*\n%d", r.FileCount)},
				{"type": "mrkdwn", "text": fmt.Sprintf("*HIGH / MED / LOW:*\n%d / %d / %d", r.High, r.Med, r.Low)},
				{"type": "mrkdwn", "text": fmt.Sprintf("*Path:*\n`%s`", r.ScanPath)},
			},
		},
		{"type": "context", "elements": []map[string]string{
			{"type": "mrkdwn", "text": r.ScannedAt.Format("Jan 2, 2006 3:04 PM MST")},
		}},
	}

	if issueCount == 0 {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]string{"type": "mrkdwn", "text": ":white_check_mark: *No issues found*"},
		})
		return map[string]interface{}{"blocks": blocks}
	}

	blocks = append(blocks, map[string]interface{}{"type": "divider"})

	limit := 8
	for i, f := range r.Findings {
		if i >= limit {
			blocks = append(blocks, map[string]interface{}{
				"type": "context",
				"elements": []map[string]string{
					{"type": "mrkdwn", "text": fmt.Sprintf("_…and %d more issue(s)_", issueCount-limit)},
				},
			})
			break
		}
		sev := strings.ToUpper(normalizeSeverity(f.Severity))
		emoji := severityEmoji(sev)
		text := fmt.Sprintf(
			"%s *%s · %s*  `line %d`\n%s\n_Fix:_ %s",
			emoji, sev, formatCategory(f), f.Line, f.Message, f.Fix,
		)
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]string{"type": "mrkdwn", "text": text},
		})
	}

	return map[string]interface{}{"blocks": blocks}
}

func severityEmoji(sev string) string {
	switch sev {
	case "HIGH":
		return ":red_circle:"
	case "MEDIUM":
		return ":large_yellow_circle:"
	default:
		return ":large_green_circle:"
	}
}

func normalizeSeverity(s string) string {
	switch strings.ToLower(s) {
	case "critical", "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}

func formatCategory(f Finding) string {
	msg := strings.ToLower(f.Message)
	switch {
	case strings.Contains(msg, "sql") || strings.Contains(msg, "injection") || strings.Contains(msg, "secret"):
		return "injection"
	case strings.Contains(msg, "auth") || strings.Contains(msg, "login") || strings.Contains(msg, "rate limit"):
		return "auth & sessions"
	case strings.Contains(msg, "header") || strings.Contains(msg, "csp"):
		return "hardening"
	case strings.Contains(msg, "dependenc"):
		return "dependencies"
	default:
		if t := strings.ToLower(f.Type); t != "" && t != "security" {
			return t
		}
		return "security"
	}
}
