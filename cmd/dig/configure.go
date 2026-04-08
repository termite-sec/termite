package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ── config types ───────────────────────────────────────────────────────────

type FullConfig struct {
	Token         string        `json:"token"`
	Regulations   []string      `json:"regulations"`
	Platform      string        `json:"platform"`
	PlatformToken string        `json:"platform_token"`
	RepoOwner     string        `json:"repo_owner"`
	RepoName      string        `json:"repo_name"`
	AgentBounds   AgentBounds   `json:"agent_bounds"`
	Notifications Notifications `json:"notifications"`
}

type AgentBounds struct {
	Critical     string `json:"critical"`
	High         string `json:"high"`
	Medium       string `json:"medium"`
	Low          string `json:"low"`
	AutoPR       bool   `json:"auto_pr"`
	AssignAuthor bool   `json:"assign_author"`
}

type Notifications struct {
	Slack          string `json:"slack"`
	Discord        string `json:"discord"`
	Email          string `json:"email"`
	MicrosoftTeams string `json:"microsoft_teams"`
}

// ── regulation registry ────────────────────────────────────────────────────

type Regulation struct {
	Key      string
	Label    string
	Desc     string
	Fine     string
	Category []string
}

var allRegulations = []Regulation{
	// federal
	{"hipaa",        "HIPAA",           "Health Insurance Portability and Accountability Act",                    "fines up to $1.9M per year per violation category",              []string{"federal", "health"}},
	{"coppa",        "COPPA",           "Children's Online Privacy Protection Act",                               "fines up to $51,744 per violation",                              []string{"federal", "children", "data"}},
	{"can-spam",     "CAN-SPAM",        "Controlling the Assault of Non-Solicited Pornography and Marketing Act", "fines up to $51,744 per email",                                  []string{"federal", "data"}},
	{"tcpa",         "TCPA",            "Telephone Consumer Protection Act",                                      "fines $500–$1,500 per violation",                                []string{"federal", "data"}},
	{"vppa",         "VPPA",            "Video Privacy Protection Act",                                           "fines up to $2,500 per violation",                               []string{"federal", "data"}},
	{"glba",         "GLBA",            "Gramm-Leach-Bliley Act — financial data privacy",                        "fines up to $100,000 per violation",                             []string{"federal", "financial"}},
	{"eo14117",      "Executive Order 14117", "Preventing Access to Americans' Bulk Sensitive Personal Data",    "civil and criminal penalties",                                   []string{"federal", "data"}},
	{"nist-800-53",  "NIST 800-53",     "US federal security and privacy controls framework",                    "federal contract loss, audit findings",                          []string{"federal"}},
	// international
	{"gdpr",         "GDPR",            "General Data Protection Regulation (EU)",                               "fines up to €20M or 4% of global annual revenue",               []string{"international", "data"}},
	// california
	{"ccpa",         "CCPA",            "California Consumer Privacy Act",                                        "fines up to $7,500 per intentional violation",                  []string{"state", "data", "california"}},
	{"cpra",         "CPRA",            "California Privacy Rights Act (extends CCPA)",                           "fines up to $7,500 per violation, triples for minors",          []string{"state", "data", "california"}},
	{"caadc",        "CAADC",           "California Age-Appropriate Design Code (SB 362)",                        "fines up to $7,500 per affected child per violation",           []string{"state", "children", "california"}},
	{"cipa",         "CIPA",            "California Invasion of Privacy Act",                                     "fines up to $5,000 per violation",                              []string{"state", "data", "california"}},
	{"shine-the-light", "Shine the Light", "California data sharing disclosure law",                             "fines up to $3,000 per violation",                              []string{"state", "data", "california"}},
	{"caloppa",      "CalOPPA",         "California Online Privacy Protection Act — privacy policy requirement",  "fines up to $2,500 per violation",                              []string{"state", "data", "california"}},
	{"ca-delete-act","California Delete Act", "SB 362 — data broker deletion requests",                          "fines up to $200/day per consumer per data broker",             []string{"state", "data", "california", "broker"}},
	{"ca-iot-sb327", "California IoT Security Law (SB 327)", "Security requirements for connected devices",      "injunctive relief, civil penalties",                            []string{"state", "california"}},
	// health (state)
	{"mhmd",         "MHMD",            "Washington My Health My Data Act",                                       "fines up to $7,500 per violation, private right of action",     []string{"state", "health"}},
	{"nv-health",    "Nevada Consumer Health Data Privacy Law", "Nevada health data privacy protections",         "civil penalties up to $15,000 per violation",                  []string{"state", "health"}},
	// biometric
	{"bipa",         "BIPA",            "Illinois Biometric Information Privacy Act",                             "fines $1,000–$5,000 per violation, private right of action",    []string{"state", "biometric"}},
	// new york
	{"ny-shield",    "New York SHIELD Act", "Stop Hacks and Improve Electronic Data Security Act",               "fines up to $250,000",                                          []string{"state", "data"}},
	{"nydfs",        "NYDFS",           "NY Dept. of Financial Services Cybersecurity Regulations — 23 NYCRR 500","fines up to $1,000 per violation per day",                    []string{"state", "financial"}},
	// financial / security
	{"pci-dss",      "PCI-DSS",         "Payment Card Industry Data Security Standard",                           "fines $5,000–$100,000 per month",                               []string{"financial"}},
	{"soc2",         "SOC2",            "Service Organization Control 2 — enterprise compliance",                 "loss of enterprise contracts, audit failures",                  []string{"financial"}},
	{"oh-sb200",     "Ohio SB 200",     "Ohio Cybersecurity Safe Harbor — affirmative defense if compliant",      "safe harbor from tort liability",                               []string{"state", "financial"}},
	// state privacy laws
	{"co-privacy",   "Colorado Privacy Act",                     "Consumer data rights and controller obligations",       "fines up to $20,000 per violation",  []string{"state", "data"}},
	{"ct-privacy",   "Connecticut Data Privacy Act",             "Consumer data rights for CT residents",                 "fines up to $5,000 per violation",   []string{"state", "data"}},
	{"de-privacy",   "Delaware Personal Data Privacy Act",       "Consumer data privacy rights for DE residents",         "fines up to $10,000 per violation",  []string{"state", "data"}},
	{"fl-privacy",   "Florida Data Privacy and Security Act",    "SB 262 — consumer data rights for FL residents",        "fines up to $50,000 per violation",  []string{"state", "data"}},
	{"in-privacy",   "Indiana Consumer Data Protection Act",     "Consumer data privacy rights for IN residents",         "fines up to $7,500 per violation",   []string{"state", "data"}},
	{"ia-privacy",   "Iowa Consumer Data Protection Act",        "Consumer data privacy rights for IA residents",         "fines up to $7,500 per violation",   []string{"state", "data"}},
	{"ky-privacy",   "Kentucky Consumer Data Protection Act",    "Consumer data privacy rights for KY residents",         "fines up to $7,500 per violation",   []string{"state", "data"}},
	{"md-privacy",   "Maryland Online Data Privacy Act",         "Consumer data privacy rights for MD residents",         "fines up to $10,000 per violation",  []string{"state", "data"}},
	{"mn-privacy",   "Minnesota Consumer Data Privacy Act",      "Consumer data privacy rights for MN residents",         "fines up to $7,500 per violation",   []string{"state", "data"}},
	{"mt-privacy",   "Montana Consumer Data Privacy Act",        "Consumer data privacy rights for MT residents",         "fines up to $7,500 per violation",   []string{"state", "data"}},
	{"ne-privacy",   "Nebraska Data Privacy Act",                "Consumer data privacy rights for NE residents",         "fines up to $7,500 per violation",   []string{"state", "data"}},
	{"nh-privacy",   "New Hampshire Consumer Expectation of Privacy Act", "Consumer data privacy for NH residents",       "fines up to $10,000 per violation",  []string{"state", "data"}},
	{"nj-privacy",   "New Jersey Personal Data Privacy Act",     "Consumer data privacy rights for NJ residents",         "fines up to $10,000 per violation",  []string{"state", "data"}},
	{"or-privacy",   "Oregon Consumer Privacy Act",              "Consumer data privacy rights for OR residents",         "fines up to $7,500 per violation",   []string{"state", "data"}},
	{"ri-privacy",   "Rhode Island Data Transparency and Privacy Protection Act", "Consumer data privacy for RI residents","fines up to $10,000 per violation", []string{"state", "data"}},
	{"tn-privacy",   "Tennessee Information Protection Act",     "Consumer data privacy rights for TN residents",         "fines up to $15,000 per violation",  []string{"state", "data"}},
	{"tx-privacy",   "Texas Data Privacy and Security Act",      "Consumer data privacy rights for TX residents",         "fines up to $7,500 per violation",   []string{"state", "data"}},
	{"ut-privacy",   "Utah Consumer Privacy Act",                "Consumer data privacy rights for UT residents",         "fines up to $7,500 per violation",   []string{"state", "data"}},
	{"va-privacy",   "Virginia Consumer Data Protection Act",    "Consumer data privacy rights for VA residents",         "fines up to $7,500 per violation",   []string{"state", "data"}},
	// data broker
	{"vt-broker",    "Vermont Data Broker Registration Law",     "Data brokers must register and meet security standards", "fines up to $10,000 per day",       []string{"state", "data", "broker"}},
	{"or-broker",    "Oregon Data Broker Law",                   "Data broker registration and consumer opt-out",          "fines up to $1,000 per day",        []string{"state", "data", "broker"}},
	{"tx-broker",    "Texas Data Broker Law",                    "Data broker registration requirements in Texas",         "fines up to $10,000 per violation", []string{"state", "data", "broker"}},
}

// ── shortcut groups ────────────────────────────────────────────────────────

type shortcutDef struct {
	label string
	cats  []string
}

var shortcuts = map[string]shortcutDef{
	"all":        {"all regulations",                          []string{"federal", "international", "state", "health", "data", "financial", "children", "biometric", "broker"}},
	"health":     {"all health regulations",                   []string{"health"}},
	"data":       {"all user data / privacy regulations",      []string{"data", "international"}},
	"financial":  {"all financial regulations",                []string{"financial"}},
	"children":   {"all children's privacy regulations",       []string{"children"}},
	"biometric":  {"all biometric regulations",                []string{"biometric"}},
	"state":      {"all US state privacy laws",                []string{"state"}},
	"federal":    {"all federal regulations",                  []string{"federal"}},
	"broker":     {"all data broker registration laws",        []string{"broker"}},
	"california": {"all California regulations",               []string{"california"}},
}

func getRegsByCategory(cats []string) []string {
	catSet := map[string]bool{}
	for _, c := range cats { catSet[c] = true }
	keys := []string{}
	seen := map[string]bool{}
	for _, reg := range allRegulations {
		for _, c := range reg.Category {
			if catSet[c] && !seen[reg.Key] {
				keys = append(keys, reg.Key)
				seen[reg.Key] = true
			}
		}
	}
	return keys
}

// ── config helpers ─────────────────────────────────────────────────────────

func configPath() string {
	return filepath.Join(os.Getenv("HOME"), ".termite", "config.json")
}

func loadFullConfig() FullConfig {
	data, err := os.ReadFile(configPath())
	if err != nil { return FullConfig{} }
	var cfg FullConfig
	json.Unmarshal(data, &cfg)
	return cfg
}

func saveFullConfig(cfg FullConfig) {
	os.MkdirAll(filepath.Dir(configPath()), 0700)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(configPath(), data, 0600)
}

// ── input helpers ──────────────────────────────────────────────────────────

func prompt(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Print(rustStyle.Render("  → ") + sandStyle.Render(label) + soilStyle.Render(" ["+defaultVal+"]") + ": ")
	} else {
		fmt.Print(rustStyle.Render("  → ") + sandStyle.Render(label) + ": ")
	}
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" { return defaultVal }
	return input
}

func promptYN(label string, defaultYes bool) bool {
	def := "y/N"
	if defaultYes { def = "Y/n" }
	fmt.Print(rustStyle.Render("  → ") + sandStyle.Render(label) + soilStyle.Render(" ["+def+"]") + ": ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" { return defaultYes }
	return input == "y" || input == "yes"
}

func promptChoice(label string, choices []string, defaultIdx int) string {
	fmt.Println(sandStyle.Render("  " + label + ":"))
	for i, c := range choices {
		if i == defaultIdx {
			fmt.Println(rustStyle.Render("  ▸ ["+fmt.Sprintf("%d", i+1)+"] ") + sandStyle.Render(c))
		} else {
			fmt.Println(soilStyle.Render(fmt.Sprintf("    [%d] %s", i+1, c)))
		}
	}
	fmt.Print(rustStyle.Render("  → ") + sandStyle.Render("choice") + soilStyle.Render(fmt.Sprintf(" [%d]", defaultIdx+1)) + ": ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" { return choices[defaultIdx] }
	for i, c := range choices {
		if input == fmt.Sprintf("%d", i+1) { return c }
	}
	return choices[defaultIdx]
}

func printStep(n int, label string) {
	fmt.Println()
	fmt.Println(amberStyle.Render(fmt.Sprintf("  ─── step %d", n)) + boldSand.Render(" — "+label))
	fmt.Println()
}

// ── regulation selector ────────────────────────────────────────────────────

func selectRegulations(current []string) []string {
	selected := map[string]bool{}
	for _, r := range current { selected[r] = true }

	regMap := map[string]Regulation{}
	for _, r := range allRegulations { regMap[r.Key] = r }

	reader := bufio.NewReader(os.Stdin)

	groups := []struct {
		title string
		keys  []string
	}{
		{"Federal", []string{"hipaa", "coppa", "can-spam", "tcpa", "vppa", "glba", "eo14117", "nist-800-53"}},
		{"International", []string{"gdpr"}},
		{"California", []string{"ccpa", "cpra", "caadc", "cipa", "shine-the-light", "caloppa", "ca-delete-act", "ca-iot-sb327"}},
		{"Health (State)", []string{"mhmd", "nv-health"}},
		{"Biometric", []string{"bipa"}},
		{"Financial & Security", []string{"pci-dss", "soc2", "nydfs", "glba", "oh-sb200"}},
		{"New York", []string{"ny-shield", "nydfs"}},
		{"State Privacy Laws", []string{
			"co-privacy", "ct-privacy", "de-privacy", "fl-privacy",
			"in-privacy", "ia-privacy", "ky-privacy", "md-privacy",
			"mn-privacy", "mt-privacy", "ne-privacy", "nh-privacy",
			"nj-privacy", "or-privacy", "ri-privacy", "tn-privacy",
			"tx-privacy", "ut-privacy", "va-privacy",
		}},
		{"Data Broker", []string{"vt-broker", "or-broker", "tx-broker"}},
	}

	for {
		fmt.Println()
		fmt.Println(boldSand.Render("  ── quick add shortcuts ─────────────────────────────────────────"))
		fmt.Println(soilStyle.Render("    add all        ") + soilStyle.Render("— every regulation"))
		fmt.Println(soilStyle.Render("    add health     ") + soilStyle.Render("— HIPAA, MHMD, Nevada Health"))
		fmt.Println(soilStyle.Render("    add data       ") + soilStyle.Render("— GDPR, CCPA, all state privacy laws"))
		fmt.Println(soilStyle.Render("    add financial  ") + soilStyle.Render("— PCI-DSS, GLBA, NYDFS, SOC2"))
		fmt.Println(soilStyle.Render("    add children   ") + soilStyle.Render("— COPPA, CAADC"))
		fmt.Println(soilStyle.Render("    add biometric  ") + soilStyle.Render("— BIPA"))
		fmt.Println(soilStyle.Render("    add state      ") + soilStyle.Render("— all US state privacy laws"))
		fmt.Println(soilStyle.Render("    add federal    ") + soilStyle.Render("— all federal regulations"))
		fmt.Println(soilStyle.Render("    add california ") + soilStyle.Render("— all California regulations"))
		fmt.Println(soilStyle.Render("    add broker     ") + soilStyle.Render("— data broker registration laws"))
		fmt.Println(soilStyle.Render("    clear          ") + soilStyle.Render("— deselect all"))
		fmt.Println(soilStyle.Render("    done           ") + soilStyle.Render("— confirm and continue"))
		fmt.Println(boldSand.Render("  ────────────────────────────────────────────────────────────────"))
		fmt.Println()

		seen := map[string]bool{}
		for _, g := range groups {
			fmt.Println(boldSand.Render("  " + g.title))
			for _, key := range g.keys {
				if seen[key] { continue }
				seen[key] = true
				reg, ok := regMap[key]
				if !ok { continue }
				check := soilStyle.Render("[ ]")
				if selected[key] { check = okStyle.Render("[✓]") }
				fmt.Printf("    %s %-12s %s\n",
					check,
					rustStyle.Render(key),
					soilStyle.Render(reg.Label+" — "+reg.Fine),
				)
			}
			fmt.Println()
		}

		count := 0
		for _, v := range selected { if v { count++ } }
		fmt.Println(soilStyle.Render(fmt.Sprintf("  %d Regulation(s) selected", count)))
		fmt.Println()
		fmt.Print(rustStyle.Render("  → ") + sandStyle.Render("Type a key, shortcut, or 'done'") + ": ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch {
		case input == "done" || input == "":
			goto done

		case input == "clear":
			selected = map[string]bool{}
			fmt.Println(okStyle.Render("  ✓ cleared"))

		case strings.HasPrefix(input, "add "):
			sc := strings.TrimPrefix(input, "add ")
			if def, ok := shortcuts[sc]; ok {
				keys := getRegsByCategory(def.cats)
				for _, k := range keys { selected[k] = true }
				fmt.Println(okStyle.Render(fmt.Sprintf("  ✓ Added %s (%d regulations)", def.label, len(keys))))
			} else {
				fmt.Println(amberStyle.Render("  Unknown shortcut: " + sc))
				fmt.Println(soilStyle.Render("  Valid shortcuts: all, health, data, financial, children, biometric, state, federal, california, broker"))
			}

		case strings.HasPrefix(input, "remove "):
			key := strings.TrimPrefix(input, "remove ")
			if _, ok := regMap[key]; ok {
				selected[key] = false
				fmt.Println(amberStyle.Render("  ✗ Removed " + key))
			} else {
				fmt.Println(amberStyle.Render("  Unknown key: " + key))
			}

		default:
			// toggle by key
			if reg, ok := regMap[input]; ok {
				selected[input] = !selected[input]
				if selected[input] {
					fmt.Println(okStyle.Render("  ✓ added " + reg.Label))
				} else {
					fmt.Println(amberStyle.Render("  ✗ removed " + reg.Label))
				}
			} else {
				fmt.Println(amberStyle.Render("  unknown key: " + input))
				fmt.Println(soilStyle.Render("  use the key shown in the left column (e.g. gdpr, hipaa, bipa)"))
			}
		}
	}

done:
	result := []string{}
	for _, reg := range allRegulations {
		if selected[reg.Key] { result = append(result, reg.Key) }
	}
	return result
}

// ── configure command ──────────────────────────────────────────────────────

func runConfigure() {
	cfg := loadFullConfig()
	if cfg.Token == "" { cfg.Token = loadOrCreateToken() }

	fmt.Println()
	fmt.Println(boldSand.Render("  termite setup wizard"))
	fmt.Println(soilStyle.Render("  configure your legal profile, platform, and agent"))

	printStep(1, "legal profile")
	fmt.Println(soilStyle.Render("  which regulations apply to your codebase?"))
	fmt.Println(soilStyle.Render("  termite maps every finding to these laws and calculates fine exposure."))
	fmt.Println(soilStyle.Render("  toggle with the key shown on the left, or use shortcuts to bulk-add."))

	cfg.Regulations = selectRegulations(cfg.Regulations)

	fmt.Println()
	if len(cfg.Regulations) == 0 {
		fmt.Println(amberStyle.Render("  ⚠ no regulations selected"))
	} else {
		fmt.Println(okStyle.Render(fmt.Sprintf("  ✓ %d regulation(s) active", len(cfg.Regulations))))
	}

	printStep(2, "Connect your platform")
	fmt.Println(soilStyle.Render("  Termite needs access to create branches, issues, and PRs."))
	fmt.Println()

	platforms := []string{"GitHub", "GitLab", "Bitbucket", "Azure DevOps", "skip for now"}
	currentPlatformIdx := 4
	for i, p := range platforms {
		if strings.EqualFold(p, cfg.Platform) { currentPlatformIdx = i }
	}

	platform := promptChoice("Which platform?", platforms, currentPlatformIdx)
	cfg.Platform = platform

	if platform != "skip for now" {
		fmt.Println()
		switch platform {
		case "GitHub":
			fmt.Println(soilStyle.Render("  github.com → settings → developer settings → personal access tokens"))
			fmt.Println(soilStyle.Render("  Required scopes: repo, issues, pull_requests, projects"))
		case "GitLab":
			fmt.Println(soilStyle.Render("  gitlab.com → user settings → access tokens"))
			fmt.Println(soilStyle.Render("  Required scopes: api, read_repository, write_repository"))
		case "Bitbucket":
			fmt.Println(soilStyle.Render("  bitbucket.org → settings → app passwords"))
			fmt.Println(soilStyle.Render("  Required permissions: repositories (rw), issues (rw), pull requests (rw)"))
		case "Azure DevOps":
			fmt.Println(soilStyle.Render("  dev.azure.com → user settings → personal access tokens"))
			fmt.Println(soilStyle.Render("  Required scopes: code (read/write), work items (read/write)"))
		}
		fmt.Println()
		cfg.PlatformToken = prompt("Paste your token", cfg.PlatformToken)
		cfg.RepoOwner     = prompt("Repo owner (username or org)", cfg.RepoOwner)
		cfg.RepoName      = prompt("Repo name", cfg.RepoName)
		fmt.Println()
		fmt.Println(okStyle.Render("  ✓ Platform: ") + sandStyle.Render(platform+" — "+cfg.RepoOwner+"/"+cfg.RepoName))
	}

	printStep(3, "Agent Bounds")
	fmt.Println(soilStyle.Render("  What should termite do automatically when it finds issues?"))
	fmt.Println(soilStyle.Render("  Change anytime with: termite agent configure"))
	fmt.Println()

	actions    := []string{
		"auto_fix (create branch + fix + PR)",
		"ticket (create issue, assign to author)",
		"warn (comment only)",
		"ignore (log only)",
	}
	actionKeys := []string{"auto_fix", "ticket", "warn", "ignore"}

	getActionIdx := func(key string) int {
		for i, k := range actionKeys { if k == key { return i } }
		return 1
	}

	fmt.Println(boldSand.Render("  CRITICAL:") + soilStyle.Render(" e.g. SQL injection, hardcoded secret, GDPR violation"))
	c := promptChoice("action", actions, getActionIdx(cfg.AgentBounds.Critical))
	for i, a := range actions { if a == c { cfg.AgentBounds.Critical = actionKeys[i] } }

	fmt.Println()
	fmt.Println(boldSand.Render("  HIGH:") + soilStyle.Render(" e.g. weak hashing, missing auth, insecure config"))
	c = promptChoice("action", actions, getActionIdx(cfg.AgentBounds.High))
	for i, a := range actions { if a == c { cfg.AgentBounds.High = actionKeys[i] } }

	fmt.Println()
	fmt.Println(boldSand.Render("  MEDIUM:") + soilStyle.Render(" e.g. missing rate limiting, logging PII"))
	c = promptChoice("action", actions, getActionIdx(cfg.AgentBounds.Medium))
	for i, a := range actions { if a == c { cfg.AgentBounds.Medium = actionKeys[i] } }

	fmt.Println()
	fmt.Println(boldSand.Render("  LOW:") + soilStyle.Render(" e.g. missing headers, minor misconfigs"))
	c = promptChoice("action", actions, getActionIdx(cfg.AgentBounds.Low))
	for i, a := range actions { if a == c { cfg.AgentBounds.Low = actionKeys[i] } }

	fmt.Println()
	cfg.AgentBounds.AutoPR       = promptYN("Automatically open PRs for fixes?", cfg.AgentBounds.AutoPR)
	cfg.AgentBounds.AssignAuthor = promptYN("Assign issues to the commit author?", cfg.AgentBounds.AssignAuthor)

	printStep(4, "Notifications")
	fmt.Println(soilStyle.Render("  Where should termite send alerts?"))
	fmt.Println()
	cfg.Notifications.Slack          = prompt("slack webhook url", cfg.Notifications.Slack)
	cfg.Notifications.Discord        = prompt("discord webhook url", cfg.Notifications.Discord)
	cfg.Notifications.MicrosoftTeams = prompt("microsoft teams webhook url", cfg.Notifications.MicrosoftTeams)
	cfg.Notifications.Email          = prompt("email address", cfg.Notifications.Email)

	saveFullConfig(cfg)

	fmt.Println()
	fmt.Println(okStyle.Render("  ✓ Configuration saved to ~/.termite/config.json"))
	fmt.Println()
	fmt.Println(boldSand.Render("  Next steps:"))
	fmt.Println(soilStyle.Render("    termite init          — add termite.yml to your repo"))
	fmt.Println(soilStyle.Render("    termite agent start   — start the agentic loop"))
	fmt.Println(soilStyle.Render("    termite scan .        — run your first scan"))
	fmt.Println()
}

// ── init command ───────────────────────────────────────────────────────────

func runInit() {
	cfg := loadFullConfig()

	regs := cfg.Regulations
	if len(regs) == 0 { regs = []string{"gdpr", "hipaa", "ccpa"} }

	critAction := cfg.AgentBounds.Critical; if critAction == "" { critAction = "auto_fix" }
	highAction := cfg.AgentBounds.High;     if highAction == "" { highAction = "ticket" }
	medAction  := cfg.AgentBounds.Medium;   if medAction  == "" { medAction  = "warn" }
	lowAction  := cfg.AgentBounds.Low;      if lowAction  == "" { lowAction  = "ignore" }

	yml := fmt.Sprintf(`# termite.yml — drop this in your repo root
# docs: https://termite.dev/docs

version: 1

# ── regulations ──────────────────────────────────────────────────────────────
# termite maps every finding to these laws and calculates fine exposure
regulations:
%s

# ── scan gates ───────────────────────────────────────────────────────────────
gates:
  pre_commit:
    enabled: true
    scan: [secrets]
    block_on: critical

  pull_request:
    enabled: true
    scan: [sast, iac, secrets]
    diff_only: true
    block_on: critical
    warn_on: [high, medium]
    post_pr_comment: true

  build:
    enabled: true
    scan: [sca, containers, licenses]
    block_on: critical

  staging:
    enabled: true
    scan: [dast, sast, sca]
    block_on: high

  production:
    enabled: true
    gate_check_only: true
    require_all_previous_passed: true
    generate_compliance_report: true

# ── scheduled scans ──────────────────────────────────────────────────────────
schedule:
  pentest:
    cron: "0 2 * * 1"
    scan: [pentest, deep_analysis]
    environment: staging

  full_scan:
    cron: "0 2 * * *"
    scan: [sast, sca, iac, containers]

# ── agent bounds ─────────────────────────────────────────────────────────────
agent:
  github_issues: true
  auto_pr: true
  assign_to_author: true
  bounds:
    critical: %s
    high:     %s
    medium:   %s
    low:      %s

# ── notifications ─────────────────────────────────────────────────────────────
notifications:
  slack:           "%s"
  discord:         "%s"
  microsoft_teams: "%s"
  email:           "%s"
`,
		formatRegulations(regs),
		critAction, highAction, medAction, lowAction,
		cfg.Notifications.Slack,
		cfg.Notifications.Discord,
		cfg.Notifications.MicrosoftTeams,
		cfg.Notifications.Email,
	)

	ymlPath := "termite.yml"
	if _, err := os.Stat(ymlPath); err == nil {
		fmt.Println(amberStyle.Render("  termite.yml already exists."))
		if !promptYN("overwrite?", false) {
			fmt.Println(soilStyle.Render("  skipped."))
			return
		}
	}

	os.WriteFile(ymlPath, []byte(yml), 0644)
	fmt.Println()
	fmt.Println(okStyle.Render("  ✓ Created termite.yml"))
	fmt.Println(soilStyle.Render("  Commit this file to activate termite in CI/CD"))
	fmt.Println()
	fmt.Println(boldSand.Render("  CI/CD pipeline snippets:"))
	fmt.Println()
	fmt.Println(sandStyle.Render("  GitHub Actions (.github/workflows/termite.yml):"))
	fmt.Println(soilStyle.Render("    on: [push, pull_request]"))
	fmt.Println(soilStyle.Render("    steps:"))
	fmt.Println(soilStyle.Render("      - uses: actions/checkout@v4"))
	fmt.Println(soilStyle.Render("      - run: curl -sSL https://get.termite.dev | sh"))
	fmt.Println(soilStyle.Render("      - run: termite scan . --ci"))
	fmt.Println()
	fmt.Println(sandStyle.Render("  GitLab (.gitlab-ci.yml):"))
	fmt.Println(soilStyle.Render("    termite:"))
	fmt.Println(soilStyle.Render("      script:"))
	fmt.Println(soilStyle.Render("        - curl -sSL https://get.termite.dev | sh"))
	fmt.Println(soilStyle.Render("        - termite scan . --ci"))
	fmt.Println()
	fmt.Println(sandStyle.Render("  Bitbucket (bitbucket-pipelines.yml):"))
	fmt.Println(soilStyle.Render("    - step:"))
	fmt.Println(soilStyle.Render("        script:"))
	fmt.Println(soilStyle.Render("          - curl -sSL https://get.termite.dev | sh"))
	fmt.Println(soilStyle.Render("          - termite scan . --ci"))
	fmt.Println()
	fmt.Println(sandStyle.Render("  Azure DevOps (azure-pipelines.yml):"))
	fmt.Println(soilStyle.Render("    steps:"))
	fmt.Println(soilStyle.Render("      - script: |"))
	fmt.Println(soilStyle.Render("          curl -sSL https://get.termite.dev | sh"))
	fmt.Println(soilStyle.Render("          termite scan . --ci"))
	fmt.Println(soilStyle.Render("        displayName: Termite Security Scan"))
	fmt.Println()
}

func formatRegulations(regs []string) string {
	regMap := map[string]Regulation{}
	for _, r := range allRegulations { regMap[r.Key] = r }
	lines := ""
	for _, r := range regs {
		if reg, ok := regMap[r]; ok {
			lines += fmt.Sprintf("  - %-20s # %s\n", r, reg.Label)
		} else {
			lines += fmt.Sprintf("  - %s\n", r)
		}
	}
	return strings.TrimRight(lines, "\n")
}

// ── connect command ────────────────────────────────────────────────────────

func runConnect(platform string) {
	cfg := loadFullConfig()
	fmt.Println()
	fmt.Println(boldSand.Render("  Connecting termite to " + platform))
	fmt.Println()

	switch strings.ToLower(platform) {
	case "github":
		fmt.Println(soilStyle.Render("  Option 1 — GitHub App (recommended)"))
		fmt.Println(soilStyle.Render("    • @termite[bot] identity in your repo"))
		fmt.Println(soilStyle.Render("    • automatic issue/PR/branch management"))
		fmt.Println(soilStyle.Render("    • project board integration"))
		fmt.Println()
		fmt.Println(amberStyle.Render("    → install: https://github.com/apps/termite-security"))
		fmt.Println()
		fmt.Println(soilStyle.Render("  Option 2 — personal access token"))
		fmt.Println(soilStyle.Render("    github.com → settings → developer settings → fine-grained tokens"))
		fmt.Println(soilStyle.Render("    permissions: contents (rw), issues (rw), pull requests (rw), projects (rw)"))
		fmt.Println()
		cfg.PlatformToken = prompt("Paste your token", cfg.PlatformToken)
		cfg.RepoOwner     = prompt("Repo owner", cfg.RepoOwner)
		cfg.RepoName      = prompt("Repo name", cfg.RepoName)
		cfg.Platform      = "github"
	case "gitlab":
		fmt.Println(soilStyle.Render("  gitlab.com → user settings → access tokens"))
		fmt.Println(soilStyle.Render("  scopes: api, read_repository, write_repository"))
		fmt.Println()
		cfg.PlatformToken = prompt("Paste your token", cfg.PlatformToken)
		cfg.RepoOwner     = prompt("Namespace (username or group)", cfg.RepoOwner)
		cfg.RepoName      = prompt("Project name", cfg.RepoName)
		cfg.Platform      = "gitlab"
	case "bitbucket":
		fmt.Println(soilStyle.Render("  bitbucket.org → personal settings → app passwords"))
		fmt.Println(soilStyle.Render("  permissions: repositories (rw), issues (rw), pull requests (rw)"))
		fmt.Println()
		cfg.PlatformToken = prompt("Paste your app password", cfg.PlatformToken)
		cfg.RepoOwner     = prompt("Workspace", cfg.RepoOwner)
		cfg.RepoName      = prompt("Repo slug", cfg.RepoName)
		cfg.Platform      = "bitbucket"
	case "azure":
		fmt.Println(soilStyle.Render("  dev.azure.com → user settings → personal access tokens"))
		fmt.Println(soilStyle.Render("  scopes: code (read/write), work items (read/write)"))
		fmt.Println()
		cfg.PlatformToken = prompt("Paste your token", cfg.PlatformToken)
		cfg.RepoOwner     = prompt("Organization name", cfg.RepoOwner)
		cfg.RepoName      = prompt("Project/repo name", cfg.RepoName)
		cfg.Platform      = "azure"
	default:
		fmt.Println(amberStyle.Render("  Unknown platform. use: github, gitlab, bitbucket, azure"))
		return
	}

	saveFullConfig(cfg)
	fmt.Println()
	fmt.Println(okStyle.Render("  ✓ Connected: " + cfg.RepoOwner + "/" + cfg.RepoName + " on " + cfg.Platform))
	fmt.Println(soilStyle.Render("  run: termite agent start"))
	fmt.Println()
}

// ── agent commands ─────────────────────────────────────────────────────────

func runAgentStart() {
	cfg := loadFullConfig()
	fmt.Println()
	fmt.Println(boldSand.Render("  Starting termite agent"))
	fmt.Println()

	if cfg.Platform == "" || cfg.Platform == "Skip for now" {
		fmt.Println(amberStyle.Render("  no platform connected."))
		fmt.Println(soilStyle.Render("  run: termite connect github"))
		return
	}
	if cfg.PlatformToken == "" {
		fmt.Println(amberStyle.Render("  No platform token found."))
		fmt.Println(soilStyle.Render("  run: termite connect " + strings.ToLower(cfg.Platform)))
		return
	}

	fmt.Println(soilStyle.Render("  platform:    ") + sandStyle.Render(cfg.Platform))
	fmt.Println(soilStyle.Render("  repo:        ") + sandStyle.Render(cfg.RepoOwner+"/"+cfg.RepoName))
	fmt.Println(soilStyle.Render("  regulations: ") + sandStyle.Render(fmt.Sprintf("%d active", len(cfg.Regulations))))
	fmt.Println(soilStyle.Render("  bounds:"))
	fmt.Println(soilStyle.Render("    critical → ") + rustStyle.Render(cfg.AgentBounds.Critical))
	fmt.Println(soilStyle.Render("    high     → ") + amberStyle.Render(cfg.AgentBounds.High))
	fmt.Println(soilStyle.Render("    medium   → ") + sandStyle.Render(cfg.AgentBounds.Medium))
	fmt.Println(soilStyle.Render("    low      → ") + soilStyle.Render(cfg.AgentBounds.Low))
	fmt.Println()

	pidFile := filepath.Join(os.Getenv("HOME"), ".termite", "agent.pid")
	if _, err := os.Stat(pidFile); err == nil {
		fmt.Println(amberStyle.Render("  Agent already running."))
		fmt.Println(soilStyle.Render("  Termite agent status / termite agent stop"))
		return
	}

	pid := os.Getpid()
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0600)
	fmt.Println(okStyle.Render(fmt.Sprintf("  ✓ Agent started (pid %d)", pid)))
	fmt.Println()
	fmt.Println(soilStyle.Render("  The agent will:"))
	fmt.Println(soilStyle.Render("    • watch for scan findings"))
	fmt.Println(soilStyle.Render("    • create issues on "+cfg.Platform+" for violations"))
	fmt.Println(soilStyle.Render("    • open targeted fix PRs for critical findings"))
	fmt.Println(soilStyle.Render("    • move tickets through your project board"))
	fmt.Println(soilStyle.Render("    • request reviews from commit authors"))
	fmt.Println()
	fmt.Println(soilStyle.Render("  logs: ~/.termite/agent.log"))
	fmt.Println(soilStyle.Render("  stop: termite agent stop"))
	fmt.Println()
}

func runAgentStop() {
	pidFile := filepath.Join(os.Getenv("HOME"), ".termite", "agent.pid")
	if _, err := os.Stat(pidFile); err != nil {
		fmt.Println(amberStyle.Render("  Agent is not running."))
		return
	}
	os.Remove(pidFile)
	fmt.Println()
	fmt.Println(okStyle.Render("  ✓ Agent stopped."))
	fmt.Println()
}

func runAgentStatus() {
	cfg     := loadFullConfig()
	pidFile := filepath.Join(os.Getenv("HOME"), ".termite", "agent.pid")

	fmt.Println()
	fmt.Println(boldSand.Render("  Termite agent status"))
	fmt.Println()

	if _, err := os.Stat(pidFile); err != nil {
		fmt.Println(soilStyle.Render("  status: ") + amberStyle.Render("stopped"))
		fmt.Println(soilStyle.Render("  run: termite agent start"))
	} else {
		pidData, _ := os.ReadFile(pidFile)
		fmt.Println(soilStyle.Render("  status: ") + okStyle.Render("running") + soilStyle.Render(" (pid "+string(pidData)+")"))
	}

	fmt.Println()
	fmt.Println(soilStyle.Render("  platform:    ") + sandStyle.Render(cfg.Platform))
	fmt.Println(soilStyle.Render("  repo:        ") + sandStyle.Render(cfg.RepoOwner+"/"+cfg.RepoName))
	fmt.Println(soilStyle.Render("  regulations: ") + sandStyle.Render(fmt.Sprintf("%d active", len(cfg.Regulations))))
	fmt.Println()
	fmt.Println(soilStyle.Render("  bounds:"))
	fmt.Println(soilStyle.Render("    critical → ") + rustStyle.Render(cfg.AgentBounds.Critical))
	fmt.Println(soilStyle.Render("    high     → ") + amberStyle.Render(cfg.AgentBounds.High))
	fmt.Println(soilStyle.Render("    medium   → ") + sandStyle.Render(cfg.AgentBounds.Medium))
	fmt.Println(soilStyle.Render("    low      → ") + soilStyle.Render(cfg.AgentBounds.Low))
	fmt.Println()
	fmt.Println(soilStyle.Render("  Termite agent configure — change bounds"))
	fmt.Println()
}

func runAgentConfigure() {
	cfg := loadFullConfig()
	fmt.Println()
	fmt.Println(boldSand.Render("  Configure agent bounds"))
	fmt.Println(soilStyle.Render("  Set what termite does autonomously for each severity level"))
	fmt.Println()

	actions    := []string{
		"auto_fix — create branch, write fix, open PR",
		"ticket   — create issue, assign to author",
		"warn     — post comment only",
		"ignore   — log only, no action",
	}
	actionKeys := []string{"auto_fix", "ticket", "warn", "ignore"}
	getIdx     := func(key string) int {
		for i, k := range actionKeys { if k == key { return i } }
		return 1
	}

	fmt.Println(boldSand.Render("  CRITICAL") + soilStyle.Render(" — SQL injection, GDPR violation, hardcoded secret"))
	c := promptChoice("action", actions, getIdx(cfg.AgentBounds.Critical))
	for i, a := range actions { if a == c { cfg.AgentBounds.Critical = actionKeys[i] } }

	fmt.Println()
	fmt.Println(boldSand.Render("  HIGH") + soilStyle.Render(" — weak hashing, missing auth, insecure config"))
	c = promptChoice("action", actions, getIdx(cfg.AgentBounds.High))
	for i, a := range actions { if a == c { cfg.AgentBounds.High = actionKeys[i] } }

	fmt.Println()
	fmt.Println(boldSand.Render("  MEDIUM") + soilStyle.Render(" — missing rate limiting, logging PII"))
	c = promptChoice("action", actions, getIdx(cfg.AgentBounds.Medium))
	for i, a := range actions { if a == c { cfg.AgentBounds.Medium = actionKeys[i] } }

	fmt.Println()
	fmt.Println(boldSand.Render("  LOW") + soilStyle.Render(" — missing headers, minor misconfigs"))
	c = promptChoice("action", actions, getIdx(cfg.AgentBounds.Low))
	for i, a := range actions { if a == c { cfg.AgentBounds.Low = actionKeys[i] } }

	fmt.Println()
	cfg.AgentBounds.AutoPR       = promptYN("Automatically open PRs for fixes?", cfg.AgentBounds.AutoPR)
	cfg.AgentBounds.AssignAuthor = promptYN("Assign issues to commit author?", cfg.AgentBounds.AssignAuthor)

	saveFullConfig(cfg)
	fmt.Println()
	fmt.Println(okStyle.Render("  ✓ Agent bounds updated"))
	fmt.Println(soilStyle.Render("  Termite agent stop && termite agent start — to apply"))
	fmt.Println()
}
