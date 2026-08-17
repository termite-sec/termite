package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

type Finding struct {
	File     string
	Line     int
	Type     string
	Law      string
	Severity string
	Message  string
	Fix      string
	Fine     string
}

type Report struct {
	ScanPath   string
	FileCount  int
	Findings   []Finding
	ScannedAt  time.Time
	HighCount  int
	MedCount   int
	LowCount   int
	Exposure   string
	IssueCount int
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func SMTPFromEnv() (SMTPConfig, error) {
	host := os.Getenv("KITIN_SMTP_HOST")
	if host == "" {
		return SMTPConfig{}, fmt.Errorf("KITIN_SMTP_HOST not set")
	}

	port := 587
	if p := os.Getenv("KITIN_SMTP_PORT"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 {
			return SMTPConfig{}, fmt.Errorf("invalid KITIN_SMTP_PORT")
		}
		port = n
	}

	user := os.Getenv("KITIN_SMTP_USER")
	pass := os.Getenv("KITIN_SMTP_PASS")
	from := os.Getenv("KITIN_SMTP_FROM")
	if from == "" {
		from = user
	}
	if from == "" {
		from = "kitin@localhost"
	}

	return SMTPConfig{Host: host, Port: port, User: user, Password: pass, From: from}, nil
}

func SendReport(to string, cfg SMTPConfig, report Report) error {
	if to == "" {
		return fmt.Errorf("no recipient email")
	}

	report.IssueCount = len(report.Findings)
	if report.ScannedAt.IsZero() {
		report.ScannedAt = time.Now()
	}

	body, err := renderHTML(report)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("kitin scan report — %d issue(s) found", report.IssueCount)
	if report.IssueCount == 0 {
		subject = "kitin scan report — no issues found"
	}

	return sendHTML(cfg, to, subject, body)
}

func sendHTML(cfg SMTPConfig, to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	var msg bytes.Buffer
	msg.WriteString("From: " + cfg.From + "\r\n")
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)

	if cfg.Port == 465 {
		return sendSMTPS(addr, cfg, cfg.From, []string{to}, msg.Bytes())
	}

	return smtp.SendMail(addr, auth, cfg.From, []string{to}, msg.Bytes())
}

func sendSMTPS(addr string, cfg SMTPConfig, from string, to []string, msg []byte) error {
	tlsCfg := &tls.Config{ServerName: cfg.Host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	if cfg.User != "" {
		auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}

func renderHTML(r Report) (string, error) {
	type card struct {
		Severity string
		Category string
		Line     int
		Risk     string
		Message  string
		Fix      string
		Color    string
		Border   string
	}

	cards := make([]card, 0, len(r.Findings))
	for _, f := range r.Findings {
		sev := normalizeSeverity(f.Severity)
		color, border := severityColors(sev)
		cards = append(cards, card{
			Severity: strings.ToUpper(sev),
			Category: formatCategory(f),
			Line:     f.Line,
			Risk:     formatRiskLabel(f, sev),
			Message:  f.Message,
			Fix:      f.Fix,
			Color:    color,
			Border:   border,
		})
	}

	data := struct {
		Report Report
		Cards  []card
		Date   string
		HighBar string
		MedBar  string
		LowBar  string
	}{
		Report: r,
		Cards:  cards,
		Date:   r.ScannedAt.Format("Jan 2, 2006 3:04 PM MST"),
		HighBar: severityBar(r.HighCount, maxCount(r.HighCount, r.MedCount, r.LowCount)),
		MedBar:  severityBar(r.MedCount, maxCount(r.HighCount, r.MedCount, r.LowCount)),
		LowBar:  severityBar(r.LowCount, maxCount(r.HighCount, r.MedCount, r.LowCount)),
	}

	var buf bytes.Buffer
	if err := reportTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func maxCount(a, b, c int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	if m == 0 {
		return 1
	}
	return m
}

func severityBar(count, max int) string {
	const width = 20
	filled := 0
	if count > 0 {
		filled = count * width / max
		if filled == 0 {
			filled = 1
		}
	}
	return strings.Repeat("█", filled) + strings.Repeat("&#160;", width-filled)
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

func severityColors(sev string) (text, border string) {
	switch sev {
	case "high":
		return "#e8927c", "#e8927c"
	case "medium":
		return "#c9a227", "#8a8a8a"
	default:
		return "#7cb87c", "#7cb87c"
	}
}

func formatCategory(f Finding) string {
	t := strings.ToLower(f.Type)
	msg := strings.ToLower(f.Message)
	switch {
	case strings.Contains(msg, "sql") || strings.Contains(msg, "injection") || strings.Contains(msg, "secret") || strings.Contains(msg, "credential"):
		return "injection"
	case strings.Contains(msg, "rate limit") || strings.Contains(msg, "auth") || strings.Contains(msg, "login") || strings.Contains(msg, "session"):
		return "auth & sessions"
	case strings.Contains(msg, "header") || strings.Contains(msg, "csp"):
		return "hardening"
	case strings.Contains(msg, "dependenc") || strings.Contains(msg, "outdated"):
		return "dependencies"
	case t != "" && t != "security" && t != "general":
		return t
	default:
		return "security"
	}
}

func formatRiskLabel(f Finding, sev string) string {
	if sev == "low" {
		return "informational"
	}
	if f.Fine != "" {
		fine := strings.TrimSpace(f.Fine)
		if !strings.Contains(strings.ToLower(fine), "day") {
			return "up to " + fine + "/day"
		}
		return fine
	}
	if sev == "high" {
		return "up to $50K/day"
	}
	return "up to $10K/day"
}

var reportTemplate = template.Must(template.New("report").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>kitin scan report</title>
</head>
<body style="margin:0;padding:0;background:#141414;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#e8e8e8;">
<table width="100%" cellpadding="0" cellspacing="0" style="background:#141414;padding:24px 12px;">
<tr><td align="center">
<table width="100%" cellpadding="0" cellspacing="0" style="max-width:620px;">

<tr><td style="padding-bottom:20px;">
  <div style="font-size:22px;font-weight:700;color:#ff6a2a;letter-spacing:0.5px;">kitin</div>
  <div style="font-size:13px;color:#8a8a8a;margin-top:4px;">security scan report</div>
</td></tr>

<tr><td style="padding-bottom:16px;">
  <table width="100%" cellpadding="0" cellspacing="0" style="border:1px solid #4a4a4a;border-radius:10px;background:#1c1c1c;">
  <tr><td style="padding:16px 18px;">
    <table width="100%" cellpadding="0" cellspacing="0">
    <tr>
      <td style="font-size:13px;color:#8a8a8a;padding:4px 0;">Issues found</td>
      <td align="right" style="font-size:15px;color:#e8e8e8;font-weight:600;padding:4px 0;">{{.Report.IssueCount}}</td>
    </tr>
    <tr>
      <td style="font-size:13px;color:#8a8a8a;padding:4px 0;">Est. daily exposure</td>
      <td align="right" style="font-size:15px;color:#e8e8e8;font-weight:600;padding:4px 0;">{{.Report.Exposure}}</td>
    </tr>
    <tr>
      <td style="font-size:13px;color:#8a8a8a;padding:4px 0;">Files scanned</td>
      <td align="right" style="font-size:15px;color:#e8e8e8;font-weight:600;padding:4px 0;">{{.Report.FileCount}}</td>
    </tr>
    <tr>
      <td style="font-size:13px;color:#8a8a8a;padding:4px 0;">Scan path</td>
      <td align="right" style="font-size:13px;color:#c8a882;padding:4px 0;">{{.Report.ScanPath}}</td>
    </tr>
    <tr>
      <td style="font-size:13px;color:#8a8a8a;padding:4px 0;">Scanned at</td>
      <td align="right" style="font-size:13px;color:#8a8a8a;padding:4px 0;">{{.Date}}</td>
    </tr>
    </table>
  </td></tr>
  </table>
</td></tr>

{{if gt .Report.IssueCount 0}}
<tr><td style="padding-bottom:8px;font-size:14px;font-weight:600;color:#e8e8e8;">Severity breakdown</td></tr>
<tr><td style="padding-bottom:20px;">
  <table width="100%" cellpadding="0" cellspacing="0" style="font-size:13px;font-family:monospace;">
  <tr><td style="color:#e8927c;width:70px;">HIGH</td><td style="color:#e8927c;">{{.HighBar}}</td><td style="color:#e8e8e8;width:30px;text-align:right;">{{.Report.HighCount}}</td></tr>
  <tr><td style="color:#c9a227;padding-top:6px;">MEDIUM</td><td style="color:#c9a227;padding-top:6px;">{{.MedBar}}</td><td style="color:#e8e8e8;padding-top:6px;text-align:right;">{{.Report.MedCount}}</td></tr>
  <tr><td style="color:#7cb87c;padding-top:6px;">LOW</td><td style="color:#7cb87c;padding-top:6px;">{{.LowBar}}</td><td style="color:#e8e8e8;padding-top:6px;text-align:right;">{{.Report.LowCount}}</td></tr>
  </table>
</td></tr>
{{end}}

{{range .Cards}}
<tr><td style="padding-bottom:14px;">
  <table width="100%" cellpadding="0" cellspacing="0" style="border:1px solid {{.Border}};border-radius:10px;background:#1c1c1c;">
  <tr><td style="padding:14px 16px;">
    <div style="font-size:13px;font-weight:600;color:{{.Color}};margin-bottom:10px;">
      &#9679; {{.Severity}} &middot; {{.Category}} &nbsp;&nbsp; line {{.Line}} &middot; {{.Risk}}
    </div>
    <div style="font-size:14px;color:#e8e8e8;line-height:1.5;margin-bottom:10px;">{{.Message}}</div>
    <div style="font-size:13px;color:#8a8a8a;line-height:1.5;"><strong style="color:#8a8a8a;">Fix:</strong> {{.Fix}}</div>
  </td></tr>
  </table>
</td></tr>
{{end}}

{{if eq .Report.IssueCount 0}}
<tr><td style="padding:24px;text-align:center;border:1px solid #7cb87c;border-radius:10px;background:#1c1c1c;color:#7cb87c;font-size:15px;">
  &#10004; No issues found
</td></tr>
{{end}}

<tr><td style="padding-top:20px;font-size:12px;color:#666;text-align:center;">
  Sent by kitin &mdash; security issues are legal issues
</td></tr>

</table>
</td></tr>
</table>
</body>
</html>`))
