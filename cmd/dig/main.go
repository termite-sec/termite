package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/kitin/kitin/internal/email"
	"github.com/kitin/kitin/internal/notify"
	"github.com/mattn/go-runewidth"
)

// ── palette ────────────────────────────────────────────────────────────────

var (
	rust   lipgloss.Color
	amber  lipgloss.Color
	sand   lipgloss.Color
	moss   lipgloss.Color
	soil   lipgloss.Color
	tunnel lipgloss.Color

	rustStyle  lipgloss.Style
	amberStyle lipgloss.Style
	sandStyle  lipgloss.Style
	mossStyle  lipgloss.Style
	soilStyle  lipgloss.Style

	critBadge  lipgloss.Style
	highBadge  lipgloss.Style
	medBadge   lipgloss.Style
	lowBadge   lipgloss.Style
	boxStyle   lipgloss.Style
	labelStyle lipgloss.Style
	valueStyle lipgloss.Style
	fileStyle  lipgloss.Style
	okStyle    lipgloss.Style
	warnStyle  lipgloss.Style
	boldSand   lipgloss.Style

	// report output
	mutedStyle  lipgloss.Style
	textStyle   lipgloss.Style
	highStyle   lipgloss.Style
	medStyle    lipgloss.Style
	lowStyle    lipgloss.Style
	summaryStyle lipgloss.Style
	highBorder  lipgloss.Color
	medBorder   lipgloss.Color
	lowBorder   lipgloss.Color
	boxBorder   lipgloss.Color
)

func init() {
	if lipgloss.HasDarkBackground() {
		rust   = lipgloss.Color("#ff6a2a")
		amber  = lipgloss.Color("#ffb830")
		sand   = lipgloss.Color("#ffe090")
		moss   = lipgloss.Color("#80d050")
		soil   = lipgloss.Color("#c8a882")
		tunnel = lipgloss.Color("#302418")
	} else {
		rust  = lipgloss.Color("#c04010")
		amber = lipgloss.Color("#a06800")
		sand  = lipgloss.Color("#8a6a30")
		moss  = lipgloss.Color("#3a7010")
		soil  = lipgloss.Color("#6b4a2a")
		tunnel = lipgloss.Color("#1a1008")
	}

	rustStyle  = lipgloss.NewStyle().Foreground(rust)
	amberStyle = lipgloss.NewStyle().Foreground(amber)
	sandStyle  = lipgloss.NewStyle().Foreground(sand)
	mossStyle  = lipgloss.NewStyle().Foreground(moss)
	soilStyle  = lipgloss.NewStyle().Foreground(soil)

	critBadge = lipgloss.NewStyle().
		Foreground(rust).
		Background(lipgloss.Color("#2a1200")).
		PaddingLeft(1).PaddingRight(1).
		Bold(true)
	highBadge = lipgloss.NewStyle().
		Foreground(amber).
		Background(lipgloss.Color("#241808")).
		PaddingLeft(1).PaddingRight(1).
		Bold(true)
	medBadge = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8a7040")).
		Background(lipgloss.Color("#1a1508")).
		PaddingLeft(1).PaddingRight(1).
		Bold(true)
	lowBadge = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#5a7030")).
		Background(lipgloss.Color("#0e1408")).
		PaddingLeft(1).PaddingRight(1).
		Bold(true)
	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#2a1e10")).
		PaddingLeft(1).PaddingRight(1)

	labelStyle = lipgloss.NewStyle().Foreground(rust).Width(8)
	valueStyle = lipgloss.NewStyle().Foreground(soil)
	fileStyle  = lipgloss.NewStyle().Foreground(sand)
	okStyle    = lipgloss.NewStyle().Foreground(moss)
	warnStyle  = lipgloss.NewStyle().Foreground(amber)
	boldSand   = lipgloss.NewStyle().Foreground(sand).Bold(true)

	if lipgloss.HasDarkBackground() {
		mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a8a8a"))
		textStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#e8e8e8"))
		highStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#e8927c"))
		medStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#c9a227"))
		lowStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#7cb87c"))
		highBorder   = lipgloss.Color("#e8927c")
		medBorder    = lipgloss.Color("#6a6a6a")
		lowBorder    = lipgloss.Color("#7cb87c")
		boxBorder    = lipgloss.Color("#4a4a4a")
	} else {
		mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
		textStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#1a1a1a"))
		highStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#c04010"))
		medStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a6a00"))
		lowStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#3a7010"))
		highBorder   = lipgloss.Color("#c04010")
		medBorder    = lipgloss.Color("#888888")
		lowBorder    = lipgloss.Color("#3a7010")
		boxBorder    = lipgloss.Color("#aaaaaa")
	}
	summaryStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(boxBorder).
		Padding(0, 1)
}

// ── types ──────────────────────────────────────────────────────────────────

const version = "0.1.0"

const defaultServerURL = "http://localhost:8080" // prod: export KITIN_SERVER_URL=https://api.kitin-security.com

func serverURL() string {
	if u := os.Getenv("KITIN_SERVER_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultServerURL
}

type Config struct {
	Token string `json:"token"`
}

type File struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type ScanRequest struct {
	Token string   `json:"token"`
	Files []File   `json:"files"`
	Modes []string `json:"modes"`
}

type ScheduleRequest struct {
	Token   string `json:"token"`
	Path    string `json:"path"`
	Every   string `json:"every"`
	Slack   string `json:"slack"`
	Email   string `json:"email"`
	Discord string `json:"discord"`
}

type Finding struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Type     string `json:"type"`
	Law      string `json:"law"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Fix      string `json:"fix"`
	Fine     string `json:"fine"`
}

type ScanResponse struct {
	ID       string    `json:"id"`
	Findings []Finding `json:"findings"`
	Status   string    `json:"status"`
}

// ── spinner ────────────────────────────────────────────────────────────────

type loadModel struct {
	spinner spinner.Model
	message string
	done    bool
}

type doneMsg struct{}

func newSpinner(message string) loadModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(rust)
	return loadModel{spinner: s, message: message}
}

func (m loadModel) Init() tea.Cmd { return m.spinner.Tick }

func (m loadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case doneMsg:
		m.done = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m loadModel) View() string {
	if m.done {
		return okStyle.Render("✔ ") + mutedStyle.Render(m.message) + "\n"
	}
	return m.spinner.View() + " " + mutedStyle.Render(m.message) + "\n"
}

func runWithSpinner(message string, work func()) {
	p := tea.NewProgram(newSpinner(message))
	go func() {
		work()
		p.Send(doneMsg{})
	}()
	p.Run()
}

// ── main ───────────────────────────────────────────────────────────────────

func main() {
	token := loadOrCreateToken()

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	switch os.Args[1] {

	case "--version", "-version", "version":
		fmt.Println("kitin", version)
		return

	// ── scan ──────────────────────────────────────────────────────────────
	case "scan":
		if len(os.Args) < 3 {
			fmt.Println(amberStyle.Render("usage: kitin scan [path] [--security] [--gdpr] [--hipaa] [--ccpa] [--diff] [--ci]"))
			os.Exit(1)
		}

		path     := ""
		modes    := []string{}
		ciMode   := false

		for _, arg := range os.Args[2:] {
			switch arg {
			case "--security":   modes = append(modes, "security")
			case "--gdpr":       modes = append(modes, "gdpr")
			case "--hipaa":      modes = append(modes, "hipaa")
			case "--ccpa":       modes = append(modes, "ccpa")
			case "--access":     modes = append(modes, "access")
			case "--terraform":  modes = append(modes, "terraform")
			case "--kubernetes": modes = append(modes, "kubernetes")
			case "--docker":     modes = append(modes, "docker")
			case "--diff":
				// TODO: scan only changed files
			case "--ci":
				ciMode = true
			default:
				path = arg
			}
		}

		if len(modes) == 0 {
			modes = resolveScanModes(nil)
		} else {
			modes = resolveScanModes(modes)
		}

		cfg := loadFullConfig()
		if len(cfg.Regulations) > 0 {
			fmt.Println(mutedStyle.Render(fmt.Sprintf("  checking %d saved regulation(s) from ~/.kitin/config.json", len(cfg.Regulations))))
		}

		modes = dedupeStrings(modes)

		var files []File
		var readErr error
		readFilesQuiet(path, &files, &readErr)
		if readErr != nil {
			fmt.Println(amberStyle.Render("Error reading files: " + readErr.Error()))
			os.Exit(1)
		}
		fmt.Println(mutedStyle.Render(fmt.Sprintf("found %d files", len(files))))

		var findings []Finding
		var scanErr error
		runWithSpinner("scanning with kitin AI...", func() {
			findings, scanErr = sendScan(token, files, modes)
		})
		if scanErr != nil {
			fmt.Println(amberStyle.Render("Scan error: " + scanErr.Error()))
			os.Exit(1)
		}

		printFindings(findings, len(files))
		maybeSendScanNotifications(cfg, path, len(files), findings)

		// CI mode — exit 1 if critical findings exist
		if ciMode {
			for _, f := range findings {
				if f.Severity == "critical" {
					os.Exit(1)
				}
			}
		}

	// ── report ────────────────────────────────────────────────────────────
	case "report":
		var result map[string]interface{}
		var fetchErr error

		runWithSpinner("Fetching latest report...", func() {
			resp, err := http.Get(fmt.Sprintf("%s/report?token=%s", serverURL(), token))
			if err != nil || resp.StatusCode != 200 {
				fetchErr = fmt.Errorf("no reports found")
				return
			}
			json.NewDecoder(resp.Body).Decode(&result)
		})

		if fetchErr != nil {
			fmt.Println(amberStyle.Render("No reports found. run kitin scan first."))
			os.Exit(1)
		}

		fmt.Println(labelStyle.Render("last scan") + valueStyle.Render(fmt.Sprintf("%v", result["created_at"])))
		if findings, ok := result["findings"].([]interface{}); ok {
			fmt.Println(labelStyle.Render("issues") + rustStyle.Render(fmt.Sprintf("%d", len(findings))))
		}

	// ── schedule ──────────────────────────────────────────────────────────
	case "schedule":
		if len(os.Args) < 3 {
			fmt.Println(amberStyle.Render("usage: kitin schedule [path] --every [2h|3d|30m|1w] --slack [url] --discord [url] --email [address]"))
			os.Exit(1)
		}

		path, every, slack, email, discord := "", "24h", "", "", ""

		args := os.Args[2:]
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--every":
				if i+1 < len(args) { every = args[i+1]; i++ }
			case "--slack":
				if i+1 < len(args) { slack = args[i+1]; i++ }
			case "--email":
				if i+1 < len(args) { email = args[i+1]; i++ }
			case "--discord":
				if i+1 < len(args) { discord = args[i+1]; i++ }
			default:
				path = args[i]
			}
		}

		req := ScheduleRequest{Token: token, Path: path, Every: every, Slack: slack, Email: email, Discord: discord}
		body, _ := json.Marshal(req)

		var schedErr error
		runWithSpinner("Adding schedule...", func() {
			resp, err := http.Post(serverURL()+"/schedule", "application/json", bytes.NewBuffer(body))
			time.Sleep(500 * time.Millisecond)
			if err != nil || resp.StatusCode != 200 {
				schedErr = fmt.Errorf("failed")
			}
		})

		if schedErr != nil {
			fmt.Println(amberStyle.Render("Failed to add schedule"))
			os.Exit(1)
		}

		fmt.Println(okStyle.Render("✓ Scheduled!"))
		fmt.Println(labelStyle.Render("path")  + fileStyle.Render(path))
		fmt.Println(labelStyle.Render("every") + sandStyle.Render(every))
		if slack   != "" { fmt.Println(labelStyle.Render("slack")   + soilStyle.Render(slack)) }
		if email   != "" { fmt.Println(labelStyle.Render("email")   + soilStyle.Render(email)) }
		if discord != "" { fmt.Println(labelStyle.Render("discord") + soilStyle.Render(discord)) }

	// ── schedule-list ─────────────────────────────────────────────────────
	case "schedule-list":
		resp, err := http.Get(fmt.Sprintf("%s/schedules?token=%s", serverURL(), token))
		if err != nil || resp.StatusCode != 200 {
			fmt.Println(amberStyle.Render("Failed to fetch schedules"))
			os.Exit(1)
		}

		var schedules []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&schedules)

		if len(schedules) == 0 {
			fmt.Println(soilStyle.Render("No schedules found. run kitin schedule to add one."))
			os.Exit(0)
		}

		fmt.Println(boldSand.Render("Your schedules:"))
		fmt.Println()

		for _, s := range schedules {
			fmt.Println(labelStyle.Render("id")    + rustStyle.Render(fmt.Sprintf("%v", s["id"])))
			fmt.Println(labelStyle.Render("path")  + fileStyle.Render(fmt.Sprintf("%v", s["path"])))
			fmt.Println(labelStyle.Render("every") + sandStyle.Render(fmt.Sprintf("%v", s["every"])))
			if v, ok := s["slack"]; ok && v != "" {
				fmt.Println(labelStyle.Render("slack") + soilStyle.Render(fmt.Sprintf("%v", v)))
			}
			if v, ok := s["email"]; ok && v != "" {
				fmt.Println(labelStyle.Render("email") + soilStyle.Render(fmt.Sprintf("%v", v)))
			}
			fmt.Println()
		}

	// ── configure ─────────────────────────────────────────────────────────
	case "configure":
		runConfigure()

	case "regulations":
		runRegulations()

	// ── init ──────────────────────────────────────────────────────────────
	case "init":
		runInit()

	// ── connect ───────────────────────────────────────────────────────────
	case "connect":
		if len(os.Args) < 3 {
			fmt.Println(amberStyle.Render("usage: kitin connect [github|gitlab|bitbucket|azure]"))
			os.Exit(1)
		}
		runConnect(os.Args[2])

	// ── notify (phone push) ───────────────────────────────────────────────
	case "notify":
		runNotifyCommand(os.Args[2:])

	// ── agent ─────────────────────────────────────────────────────────────
	case "agent":
		if len(os.Args) < 3 {
			runAgentStatus()
			return
		}
		switch os.Args[2] {
		case "start":     runAgentStart()
		case "stop":      runAgentStop()
		case "status":    runAgentStatus()
		case "configure": runAgentConfigure()
		default:
			fmt.Println(amberStyle.Render("usage: kitin agent [start|stop|status|configure]"))
		}

	default:
		printHelp()
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

// ── report output ──────────────────────────────────────────────────────────

func termWidth() int {
	const fallback = 80
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || width <= 0 {
		if cols := os.Getenv("COLUMNS"); cols != "" {
			if n, err := strconv.Atoi(cols); err == nil && n > 0 {
				width = n
			}
		}
	}
	if width <= 0 {
		width = fallback
	}
	if width < 40 {
		width = 40
	}
	return width
}

// inner text width inside bordered boxes (border + horizontal padding)
func contentWidth() int {
	return termWidth() - 4
}

func styledWrap(text string, style lipgloss.Style, width int) string {
	if width < 12 {
		width = 12
	}
	return style.Width(width).Render(text)
}

func truncateWidth(text string, max int) string {
	if max <= 3 || runewidth.StringWidth(text) <= max {
		return text
	}
	var b strings.Builder
	w := 0
	for _, r := range text {
		rw := runewidth.RuneWidth(r)
		if w+rw > max-3 {
			b.WriteString("...")
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String()
}

func readFilesQuiet(path string, files *[]File, errOut *error) {
	f, err := readFiles(path)
	*files = f
	*errOut = err
}

func printFindings(findings []Finding, fileCount int) {
	if len(findings) == 0 {
		fmt.Println()
		fmt.Println(okStyle.Render("✔ No issues found"))
		return
	}

	fmt.Println()
	fmt.Println(textStyle.Render(fmt.Sprintf("found %d issues:", len(findings))))
	fmt.Println()

	summary := renderSummaryBox(len(findings), estimateDailyExposure(findings), fileCount)
	fmt.Println(summary)
	fmt.Println()

	high, med, low := countSeverities(findings)
	maxCount := high
	if med > maxCount {
		maxCount = med
	}
	if low > maxCount {
		maxCount = low
	}
	if maxCount == 0 {
		maxCount = 1
	}

	fmt.Println(textStyle.Render("Severity breakdown"))
	fmt.Println(renderSeverityBar("HIGH", high, maxCount, highStyle))
	fmt.Println(renderSeverityBar("MEDIUM", med, maxCount, medStyle))
	fmt.Println(renderSeverityBar("LOW", low, maxCount, lowStyle))
	fmt.Println()

	for _, f := range findings {
		fmt.Println(renderFindingCard(f))
		fmt.Println()
	}
}

func renderSummaryBox(issues int, exposure string, files int) string {
	w := contentWidth()
	lines := []string{
		summaryLine("Issues found", fmt.Sprintf("%d", issues), w),
		summaryLine("Est. daily exposure", exposure, w),
		summaryLine("Files scanned", fmt.Sprintf("%d", files), w),
	}
	return summaryStyle.Width(w).Render(strings.Join(lines, "\n"))
}

func summaryLine(label, value string, width int) string {
	if width >= 52 {
		return mutedStyle.Render(fmt.Sprintf("%-20s", label)) + textStyle.Render(value)
	}
	return mutedStyle.Render(label) + "\n" + textStyle.Render("  "+value)
}

func countSeverities(findings []Finding) (high, med, low int) {
	for _, f := range findings {
		switch normalizeSeverity(f.Severity) {
		case "high":
			high++
		case "medium":
			med++
		case "low":
			low++
		}
	}
	return
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

func renderSeverityBar(label string, count, maxCount int, style lipgloss.Style) string {
	w := contentWidth()
	barWidth := 20
	if w < 56 {
		barWidth = 10
	}
	if barWidth > w-14 {
		barWidth = w - 14
	}
	if barWidth < 4 {
		barWidth = 4
	}

	filled := 0
	if count > 0 {
		filled = count * barWidth / maxCount
		if filled == 0 {
			filled = 1
		}
	}
	bar := style.Render(strings.Repeat("█", filled))
	pad := strings.Repeat(" ", barWidth-filled)
	return fmt.Sprintf("  %-6s %s%s  %d", label, bar, pad, count)
}

func estimateDailyExposure(findings []Finding) string {
	total := 0
	for _, f := range findings {
		total += dailyExposureUSD(f)
	}
	if total == 0 {
		return "$0"
	}
	if total >= 1_000_000 {
		return fmt.Sprintf("$%.0fM", float64(total)/1_000_000)
	}
	return fmt.Sprintf("$%.0fK", float64(total)/1_000)
}

func dailyExposureUSD(f Finding) int {
	if f.Fine != "" {
		if v := parseUSDAmount(f.Fine); v > 0 {
			return v
		}
	}
	switch normalizeSeverity(f.Severity) {
	case "high":
		return 50_000
	case "medium":
		return 10_000
	default:
		return 0
	}
}

func parseUSDAmount(s string) int {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "/day", "")
	s = strings.ReplaceAll(s, "up to ", "")
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	if strings.HasSuffix(s, "k") {
		if n, err := strconv.ParseFloat(strings.TrimSuffix(s, "k"), 64); err == nil {
			return int(n * 1000)
		}
	}
	if strings.HasSuffix(s, "m") {
		if n, err := strconv.ParseFloat(strings.TrimSuffix(s, "m"), 64); err == nil {
			return int(n * 1_000_000)
		}
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return 0
}

func formatCategory(f Finding) string {
	t := strings.ToLower(f.Type)
	msg := strings.ToLower(f.Message)
	switch {
	case strings.Contains(msg, "sql") || strings.Contains(msg, "injection") || strings.Contains(msg, "secret") || strings.Contains(msg, "credential"):
		return "injection"
	case strings.Contains(msg, "rate limit") || strings.Contains(msg, "auth") || strings.Contains(msg, "login") || strings.Contains(msg, "session"):
		return "auth & sessions"
	case strings.Contains(msg, "header") || strings.Contains(msg, "csp") || strings.Contains(msg, "hardening"):
		return "hardening"
	case strings.Contains(msg, "dependenc") || strings.Contains(msg, "outdated"):
		return "dependencies"
	case t != "" && t != "security" && t != "general":
		return t
	default:
		return "security"
	}
}

func formatRiskLabel(f Finding) string {
	if normalizeSeverity(f.Severity) == "low" {
		return "informational"
	}
	if f.Fine != "" {
		fine := strings.TrimSpace(f.Fine)
		if !strings.Contains(strings.ToLower(fine), "day") {
			return "up to " + fine + "/day"
		}
		return fine
	}
	switch normalizeSeverity(f.Severity) {
	case "high":
		return "up to $50K/day"
	case "medium":
		return "up to $10K/day"
	default:
		return "informational"
	}
}

func renderFindingCard(f Finding) string {
	w := contentWidth()
	sev := strings.ToUpper(normalizeSeverity(f.Severity))
	cat := formatCategory(f)
	risk := formatRiskLabel(f)

	meta := fmt.Sprintf("line %d · %s", f.Line, risk)
	if runewidth.StringWidth(meta) > w {
		meta = fmt.Sprintf("line %d · %s", f.Line, truncateWidth(risk, w-10))
	}

	header := fmt.Sprintf("● %s · %s", sev, cat)

	var border lipgloss.Color
	var headerStyle lipgloss.Style
	switch normalizeSeverity(f.Severity) {
	case "high":
		border = highBorder
		headerStyle = highStyle
	case "medium":
		border = medBorder
		headerStyle = medStyle
	default:
		border = lowBorder
		headerStyle = lowStyle
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Width(w)

	body := styledWrap(header, headerStyle, w) + "\n" +
		styledWrap(meta, headerStyle, w) + "\n\n" +
		styledWrap(f.Message, textStyle, w) + "\n" +
		styledWrap("Fix: "+f.Fix, mutedStyle, w)

	return cardStyle.Render(body)
}

func maybeSendScanNotifications(cfg FullConfig, scanPath string, fileCount int, findings []Finding) {
	high, med, low := countSeverities(findings)
	report := notify.ScanReport{
		ScanPath:  scanPath,
		FileCount: fileCount,
		Findings:  toNotifyFindings(findings),
		ScannedAt: time.Now(),
		Exposure:  estimateDailyExposure(findings),
		High:      high,
		Med:       med,
		Low:       low,
	}

	pushURL := strings.TrimSpace(cfg.Notifications.PushURL)
	if pushURL == "" {
		pushURL = strings.TrimSpace(os.Getenv("KITIN_PUSH_URL"))
	}
	if pushURL != "" {
		if err := notify.SendPush(pushURL, report); err != nil {
			fmt.Println(warnStyle.Render("  push failed: " + err.Error()))
		} else {
			fmt.Println(okStyle.Render("  ✔ report pushed to " + pushURL))
		}
	}

	if webhook := strings.TrimSpace(cfg.Notifications.Slack); webhook != "" {
		if err := notify.SendSlack(webhook, report); err != nil {
			fmt.Println(warnStyle.Render("  slack failed: " + err.Error()))
		} else {
			fmt.Println(okStyle.Render("  ✔ report sent to slack"))
		}
	}

	maybeSendEmailReport(cfg, scanPath, fileCount, findings)
}

func toNotifyFindings(findings []Finding) []notify.Finding {
	out := make([]notify.Finding, len(findings))
	for i, f := range findings {
		out[i] = notify.Finding{
			File: f.File, Line: f.Line, Type: f.Type,
			Severity: f.Severity, Message: f.Message, Fix: f.Fix,
		}
	}
	return out
}

func maybeSendEmailReport(cfg FullConfig, scanPath string, fileCount int, findings []Finding) {
	to := strings.TrimSpace(cfg.Notifications.Email)
	if to == "" {
		return
	}

	smtpCfg, err := email.SMTPFromEnv()
	if err != nil {
		fmt.Println(warnStyle.Render("  email skipped: " + err.Error()))
		fmt.Println(mutedStyle.Render("  set KITIN_SMTP_HOST, KITIN_SMTP_USER, KITIN_SMTP_PASS to send reports"))
		return
	}

	high, med, low := countSeverities(findings)
	report := email.Report{
		ScanPath:   scanPath,
		FileCount:  fileCount,
		Findings:   toEmailFindings(findings),
		ScannedAt:  time.Now(),
		Exposure:   estimateDailyExposure(findings),
		HighCount:  high,
		MedCount:   med,
		LowCount:   low,
		IssueCount: len(findings),
	}

	if err := email.SendReport(to, smtpCfg, report); err != nil {
		fmt.Println(warnStyle.Render("  email failed: " + err.Error()))
		return
	}
	fmt.Println(okStyle.Render("  ✔ report emailed to " + to))
}

func toEmailFindings(findings []Finding) []email.Finding {
	out := make([]email.Finding, len(findings))
	for i, f := range findings {
		out[i] = email.Finding{
			File: f.File, Line: f.Line, Type: f.Type, Law: f.Law,
			Severity: f.Severity, Message: f.Message, Fix: f.Fix, Fine: f.Fine,
		}
	}
	return out
}

func loadOrCreateToken() string {
	// check legacy ~/.dig location first
	legacyDir  := filepath.Join(os.Getenv("HOME"), ".dig")
	legacyFile := filepath.Join(legacyDir, "config.json")

	configDir  := filepath.Join(os.Getenv("HOME"), ".kitin")
	configFile := filepath.Join(configDir, "config.json")

	// migrate from .dig or .termite to .kitin if needed
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if data, err := os.ReadFile(legacyFile); err == nil {
			os.MkdirAll(configDir, 0700)
			os.WriteFile(configFile, data, 0600)
		} else if data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".termite", "config.json")); err == nil {
			os.MkdirAll(configDir, 0700)
			os.WriteFile(configFile, data, 0600)
		}
	}

	if data, err := os.ReadFile(configFile); err == nil {
		var config Config
		if json.Unmarshal(data, &config) == nil && config.Token != "" {
			return config.Token
		}
	}

	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)

	os.MkdirAll(configDir, 0700)
	data, _ := json.Marshal(Config{Token: token})
	os.WriteFile(configFile, data, 0600)

	fmt.Println(okStyle.Render("✓ Kitin initialized — token saved to ~/.kitin/config.json"))
	return token
}

func readFiles(path string) ([]File, error) {
	var files []File
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil { return nil }
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(p)
		relevant := map[string]bool{
			".js": true, ".ts": true, ".py": true, ".go": true,
			".tf": true, ".yaml": true, ".yml": true,
			".json": true, ".env": true, ".sh": true,
			".rb": true, ".java": true, ".php": true, ".cs": true,
		}
		if d.Name() == "Dockerfile" || relevant[ext] {
			content, err := os.ReadFile(p)
			if err != nil { return nil }
			if len(content) > 50000 { return nil }
			files = append(files, File{Name: p, Content: string(content)})
		}
		return nil
	})
	return files, err
}

func resolveScanModes(flagModes []string) []string {
	cfg := loadFullConfig()
	modes := append([]string{}, flagModes...)
	modes = append(modes, cfg.Regulations...)
	if len(modes) == 0 {
		return []string{"security", "gdpr", "hipaa", "ccpa", "access", "terraform", "kubernetes", "docker"}
	}
	for _, m := range modes {
		if m == "security" {
			return dedupeStrings(modes)
		}
	}
	return dedupeStrings(append([]string{"security"}, modes...))
}

func dedupeStrings(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, s := range items {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func sendScan(token string, files []File, modes []string) ([]Finding, error) {
	url := serverURL() + "/scan"
	req := ScanRequest{Token: token, Files: files, Modes: modes}
	body, _ := json.Marshal(req)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("cannot reach kitin server at %s — is it running? (set KITIN_SERVER_URL to override)", url)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(errBody))
		if msg == "" {
			return nil, fmt.Errorf("kitin server at %s returned %d", url, resp.StatusCode)
		}
		return nil, fmt.Errorf("kitin server error: %s", msg)
	}
	var result ScanResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Findings, nil
}

func printHelp() {
	logo := rustStyle.Render(`
  _    _ _   _       
 | |  (_) | (_)      
 | | ___| |_ _ _ __  
 | |/ / | __| | '_ \ 
 |   <| | |_| | | | |
 |_|\_\_|\__|_|_| |_|
                     
                     `)

	bug := soilStyle.Render(`
       --  \
         \  \
          \- \__
          |     |____
          |^  ^ |    \________________
          |     |     |              |
           -----      |              /
             \        |             /
            / |____\__\____________/
           /  |  |  \  \
          -   |  |   \  \`)

	tagline := soilStyle.Render("  kitin — ") + amberStyle.Render("security issues are legal issues")

	col := func(cmd, desc string) string {
		return rustStyle.Render(fmt.Sprintf("    %-38s", cmd)) + soilStyle.Render(desc)
	}
	flag := func(cmd, desc string) string {
		return amberStyle.Render(fmt.Sprintf("    %-38s", cmd)) + soilStyle.Render(desc)
	}

	fmt.Println()
	fmt.Println(logo)
	fmt.Println(bug)
	fmt.Println()
	fmt.Println(tagline)
	fmt.Println()

	fmt.Println(boldSand.Render("  Setup:"))
	fmt.Println(col("kitin configure", "interactive setup wizard"))
	fmt.Println(col("kitin regulations", "update saved acts (shows previous selections)"))
	fmt.Println(col("kitin init", "create kitin.yml in current repo"))
	fmt.Println(col("kitin connect [platform]", "connect github / gitlab / bitbucket / azure / other"))
	fmt.Println(col("kitin notify test", "send test push to your phone (see push_url in config)"))
	fmt.Println()

	fmt.Println(boldSand.Render("  Scanning:"))
	fmt.Println(col("kitin scan [path]", "scan entire codebase"))
	fmt.Println(flag("  --security", "security vulnerabilities"))
	fmt.Println(flag("  --terraform", "terraform misconfigs"))
	fmt.Println(flag("  --kubernetes", "kubernetes misconfigs"))
	fmt.Println(flag("  --docker", "dockerfile issues"))
	fmt.Println(flag("  --diff", "scan only changed files (fast)"))
	fmt.Println(flag("  --ci", "CI mode — exit 1 on critical findings"))
	fmt.Println()

	fmt.Println(boldSand.Render("  Agent:"))
	fmt.Println(col("kitin agent start", "start the autonomous agent"))
	fmt.Println(col("kitin agent stop", "stop the agent"))
	fmt.Println(col("kitin agent status", "show agent status and current bounds"))
	fmt.Println(col("kitin agent configure", "set what agent does per severity"))
	fmt.Println()

	fmt.Println(boldSand.Render("  reports & schedules:"))
	fmt.Println(col("kitin report", "show latest scan report"))
	fmt.Println(col("kitin schedule [path]", "schedule recurring scans"))
	fmt.Println(col("kitin schedule-list", "list all schedules"))
	fmt.Println()

	fmt.Println(boldSand.Render("  Examples:"))
	fmt.Println(soilStyle.Render("    kitin configure"))
	fmt.Println(soilStyle.Render("    kitin connect github"))
	fmt.Println(soilStyle.Render("    kitin init"))
	fmt.Println(soilStyle.Render("    kitin scan ."))
	fmt.Println(soilStyle.Render("    kitin scan . --gdpr --hipaa"))
	fmt.Println(soilStyle.Render("    kitin scan . --diff --ci"))
	fmt.Println(soilStyle.Render("    kitin agent start"))
	fmt.Println(soilStyle.Render("    kitin agent configure"))
	fmt.Println(soilStyle.Render("    kitin schedule . --every 2h --slack https://hooks.slack.com/..."))
	fmt.Println()

	fmt.Println(boldSand.Render("  CI/CD:"))
	fmt.Println(soilStyle.Render("    curl -sSL https://get.kitin.dev | sh && kitin scan . --ci"))
	fmt.Println(soilStyle.Render("    works in GitHub Actions, GitLab, Bitbucket, Azure DevOps, Jenkins"))
	fmt.Println(soilStyle.Render("    run kitin init for full pipeline config templates"))
	fmt.Println()
}
