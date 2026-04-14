package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
}

// ── types ──────────────────────────────────────────────────────────────────

const serverURL = "https://api.termite-sec.com"

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
		return okStyle.Render("✓ ") + sandStyle.Render(m.message) + "\n"
	}
	return m.spinner.View() + " " + sandStyle.Render(m.message) + "\n"
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

	// ── scan ──────────────────────────────────────────────────────────────
	case "scan":
		if len(os.Args) < 3 {
			fmt.Println(amberStyle.Render("usage: termite scan [path] [--security] [--gdpr] [--hipaa] [--ccpa] [--diff] [--ci]"))
			os.Exit(1)
		}

		path  := ""
		modes := []string{}

		for _, arg := range os.Args[2:] {
			switch arg {
			case "--security":   modes = append(modes, "security")
			case "--access":     modes = append(modes, "access")
			case "--terraform":  modes = append(modes, "terraform")
			case "--kubernetes": modes = append(modes, "kubernetes")
			case "--docker":     modes = append(modes, "docker")
			case "--diff":       modes = append(modes, "diff")
			case "--ci":         modes = append(modes, "ci")
			default:             path = arg
			}
		}

		if len(modes) == 0 {
			modes = []string{"security", "gdpr", "hipaa", "ccpa", "access", "terraform", "kubernetes", "docker"}
		}

		fmt.Println(boldSand.Render("Scanning: ") + rustStyle.Render(path))

		var files []File
		var readErr error
		runWithSpinner("Reading files...", func() {
			files, readErr = readFiles(path)
		})
		if readErr != nil {
			fmt.Println(amberStyle.Render("Error reading files: " + readErr.Error()))
			os.Exit(1)
		}
		fmt.Println(soilStyle.Render(fmt.Sprintf("  found %d files", len(files))))

		var findings []Finding
		var scanErr error
		runWithSpinner("Scanning with termite AI...", func() {
			findings, scanErr = sendScan(token, files, modes)
		})
		if scanErr != nil {
			fmt.Println(amberStyle.Render("Scan error: " + scanErr.Error()))
			os.Exit(1)
		}

		printFindings(findings)

		// CI mode — exit 1 if critical findings exist
		for _, f := range findings {
			if f.Severity == "critical" {
				os.Exit(1)
			}
		}

	// ── report ────────────────────────────────────────────────────────────
	case "report":
		var result map[string]interface{}
		var fetchErr error

		runWithSpinner("Fetching latest report...", func() {
			resp, err := http.Get(fmt.Sprintf("%s/report?token=%s", serverURL, token))
			if err != nil || resp.StatusCode != 200 {
				fetchErr = fmt.Errorf("no reports found")
				return
			}
			json.NewDecoder(resp.Body).Decode(&result)
		})

		if fetchErr != nil {
			fmt.Println(amberStyle.Render("No reports found. run termite scan first."))
			os.Exit(1)
		}

		fmt.Println(labelStyle.Render("last scan") + valueStyle.Render(fmt.Sprintf("%v", result["created_at"])))
		if findings, ok := result["findings"].([]interface{}); ok {
			fmt.Println(labelStyle.Render("issues") + rustStyle.Render(fmt.Sprintf("%d", len(findings))))
		}

	// ── schedule ──────────────────────────────────────────────────────────
	case "schedule":
		if len(os.Args) < 3 {
			fmt.Println(amberStyle.Render("usage: termite schedule [path] --every [2h|3d|30m|1w] --slack [url] --discord [url] --email [address]"))
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
			resp, err := http.Post(serverURL+"/schedule", "application/json", bytes.NewBuffer(body))
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
		resp, err := http.Get(fmt.Sprintf("%s/schedules?token=%s", serverURL, token))
		if err != nil || resp.StatusCode != 200 {
			fmt.Println(amberStyle.Render("Failed to fetch schedules"))
			os.Exit(1)
		}

		var schedules []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&schedules)

		if len(schedules) == 0 {
			fmt.Println(soilStyle.Render("No schedules found. run termite schedule to add one."))
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

	// ── init ──────────────────────────────────────────────────────────────
	case "init":
		runInit()

	// ── connect ───────────────────────────────────────────────────────────
	case "connect":
		if len(os.Args) < 3 {
			fmt.Println(amberStyle.Render("usage: termite connect [github|gitlab|bitbucket|azure]"))
			os.Exit(1)
		}
		runConnect(os.Args[2])

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
			fmt.Println(amberStyle.Render("usage: termite agent [start|stop|status|configure]"))
		}

	default:
		printHelp()
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

func severityBadge(s string) string {
	switch s {
	case "critical": return critBadge.Render("critical")
	case "high":     return highBadge.Render("high")
	case "medium":   return medBadge.Render("medium")
	default:         return lowBadge.Render("low")
	}
}

func printFindings(findings []Finding) {
	if len(findings) == 0 {
		fmt.Println()
		fmt.Println(okStyle.Render("✓ No issues found!"))
		return
	}

	fmt.Println()
	fmt.Println(boldSand.Render(fmt.Sprintf("Found %d issues:", len(findings))))
	fmt.Println()

	for _, f := range findings {
		header := severityBadge(f.Severity) + "  " + sandStyle.Render(f.Type)
		risk := ""
		if f.Fine != "" {
			risk = "\n" + labelStyle.Render("risk") + warnStyle.Render(f.Fine)
		}
		detail := boxStyle.Render(
			labelStyle.Render("file")  + fileStyle.Render(fmt.Sprintf("%s line %d", f.File, f.Line)) + "\n" +
			labelStyle.Render("law")   + valueStyle.Render(f.Law) + "\n" +
			labelStyle.Render("issue") + valueStyle.Render(f.Message) + "\n" +
			labelStyle.Render("fix")   + mossStyle.Render(f.Fix) +
			risk,
		)
		fmt.Println(header)
		fmt.Println(detail)
		fmt.Println()
	}
}

func loadOrCreateToken() string {
	// check legacy ~/.dig location first
	legacyDir  := filepath.Join(os.Getenv("HOME"), ".dig")
	legacyFile := filepath.Join(legacyDir, "config.json")

	configDir  := filepath.Join(os.Getenv("HOME"), ".termite")
	configFile := filepath.Join(configDir, "config.json")

	// migrate from .dig to .termite if needed
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if data, err := os.ReadFile(legacyFile); err == nil {
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

	fmt.Println(okStyle.Render("✓ Termite initialized — token saved to ~/.termite/config.json"))
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

func sendScan(token string, files []File, modes []string) ([]Finding, error) {
	req := ScanRequest{Token: token, Files: files, Modes: modes}
	body, _ := json.Marshal(req)
	resp, err := http.Post(serverURL+"/scan", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("Cannot reach termite server — is it running?")
	}
	defer resp.Body.Close()
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

	tagline := soilStyle.Render("  termite — ") + amberStyle.Render("security issues are legal issues")

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
	fmt.Println(col("termite configure", "interactive setup wizard"))
	fmt.Println(col("termite init", "create termite.yml in current repo"))
	fmt.Println(col("termite connect [platform]", "connect github / gitlab / bitbucket / azure / other"))
	fmt.Println()

	fmt.Println(boldSand.Render("  Scanning:"))
	fmt.Println(col("termite scan [path]", "scan entire codebase"))
	fmt.Println(flag("  --security", "security vulnerabilities"))
	fmt.Println(flag("  --terraform", "terraform misconfigs"))
	fmt.Println(flag("  --kubernetes", "kubernetes misconfigs"))
	fmt.Println(flag("  --docker", "dockerfile issues"))
	fmt.Println(flag("  --diff", "scan only changed files (fast)"))
	fmt.Println(flag("  --ci", "CI mode — exit 1 on critical findings"))
	fmt.Println()

	fmt.Println(boldSand.Render("  Agent:"))
	fmt.Println(col("termite agent start", "start the autonomous agent"))
	fmt.Println(col("termite agent stop", "stop the agent"))
	fmt.Println(col("termite agent status", "show agent status and current bounds"))
	fmt.Println(col("termite agent configure", "set what agent does per severity"))
	fmt.Println()

	fmt.Println(boldSand.Render("  reports & schedules:"))
	fmt.Println(col("termite report", "show latest scan report"))
	fmt.Println(col("termite schedule [path]", "schedule recurring scans"))
	fmt.Println(col("termite schedule-list", "list all schedules"))
	fmt.Println()

	fmt.Println(boldSand.Render("  Examples:"))
	fmt.Println(soilStyle.Render("    termite configure"))
	fmt.Println(soilStyle.Render("    termite connect github"))
	fmt.Println(soilStyle.Render("    termite init"))
	fmt.Println(soilStyle.Render("    termite scan ."))
	fmt.Println(soilStyle.Render("    termite scan . --gdpr --hipaa"))
	fmt.Println(soilStyle.Render("    termite scan . --diff --ci"))
	fmt.Println(soilStyle.Render("    termite agent start"))
	fmt.Println(soilStyle.Render("    termite agent configure"))
	fmt.Println(soilStyle.Render("    termite schedule . --every 2h --slack https://hooks.slack.com/..."))
	fmt.Println()

	fmt.Println(boldSand.Render("  CI/CD:"))
	fmt.Println(soilStyle.Render("    curl -sSL https://get.termite.dev | sh && termite scan . --ci"))
	fmt.Println(soilStyle.Render("    works in GitHub Actions, GitLab, Bitbucket, Azure DevOps, Jenkins"))
	fmt.Println(soilStyle.Render("    run termite init for full pipeline config templates"))
	fmt.Println()
}
