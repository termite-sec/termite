package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// SendPush POSTs a scan report JSON to any URL — e.g. http://192.168.1.42:8080/kitin
// or a local ntfy server at http://192.168.1.42:2586/my-topic
func SendPush(url string, report ScanReport) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("no push url")
	}
	if report.ScannedAt.IsZero() {
		report.ScannedAt = time.Now()
	}

	// ntfy.sh and self-hosted ntfy use plain-text body
	if isNtfyURL(url) {
		return sendNtfy(url, report)
	}

	body, err := json.Marshal(reportPayload(report))
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push url returned %d: %s", resp.StatusCode, truncate(string(b), 200))
	}
	return nil
}

func isNtfyURL(url string) bool {
	u := strings.ToLower(url)
	return strings.Contains(u, "ntfy.sh") ||
		strings.Contains(u, ":2586/") ||
		strings.HasSuffix(u, "/ntfy") ||
		os.Getenv("KITIN_PUSH_NTFY") == "1"
}

func sendNtfy(url string, report ScanReport) error {
	title := fmt.Sprintf("kitin: %d issue(s)", len(report.Findings))
	if len(report.Findings) == 0 {
		title = "kitin: no issues"
	}
	message := fmt.Sprintf(
		"Path: %s\nFiles: %d\nExposure: %s\nHIGH %d · MED %d · LOW %d",
		report.ScanPath, report.FileCount, report.Exposure, report.High, report.Med, report.Low,
	)
	for i, f := range report.Findings {
		if i >= 5 {
			message += fmt.Sprintf("\n…+%d more", len(report.Findings)-5)
			break
		}
		message += fmt.Sprintf("\n• [%s] %s", strings.ToUpper(f.Severity), f.Message)
	}

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(message))
	if err != nil {
		return err
	}
	req.Header.Set("Title", title)
	req.Header.Set("Priority", "default")
	if report.High > 0 {
		req.Header.Set("Priority", "high")
	}
	req.Header.Set("Tags", "kitin,security")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned %d", resp.StatusCode)
	}
	return nil
}

func reportPayload(r ScanReport) map[string]interface{} {
	findings := make([]map[string]interface{}, len(r.Findings))
	for i, f := range r.Findings {
		findings[i] = map[string]interface{}{
			"file": f.File, "line": f.Line, "type": f.Type,
			"severity": f.Severity, "message": f.Message, "fix": f.Fix,
		}
	}
	return map[string]interface{}{
		"source":     "kitin",
		"scan_path":  r.ScanPath,
		"file_count": r.FileCount,
		"exposure":   r.Exposure,
		"high":       r.High,
		"medium":     r.Med,
		"low":        r.Low,
		"scanned_at": r.ScannedAt.Format(time.RFC3339),
		"findings":   findings,
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
