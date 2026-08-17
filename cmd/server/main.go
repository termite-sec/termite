package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kitin/kitin/internal/db"
	"github.com/kitin/kitin/internal/regulations"
)

type ScanRequest struct {
	Token string   `json:"token"`
	Files []File   `json:"files"`
	Modes []string `json:"modes"`
}

type File struct {
	Name    string `json:"name"`
	Content string `json:"content"`
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

var bedrockHTTPClient = &http.Client{Timeout: bedrockTimeout()}

func main() {
	loadEnvFiles()

	if err := db.Init(); err != nil {
		fmt.Println("db error:", err)
		os.Exit(1)
	}

	if err := validateBedrockConfig(); err != nil {
		fmt.Println("bedrock error:", err)
		os.Exit(1)
	}

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/scan", handleScan)
	http.HandleFunc("/schedule", handleSchedule)
	http.HandleFunc("/schedules", handleListSchedules)
	http.HandleFunc("/report", handleReport)

	port := "8080"
	fmt.Printf("kitin server running on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Println("server error:", err)
		os.Exit(1)
	}
}

func bedrockModel() string {
	if m := os.Getenv("KITIN_BEDROCK_MODEL"); m != "" {
		return m
	}
	return "openai.gpt-oss-120b"
}

func validateBedrockConfig() error {
	if strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/") == "" {
		return fmt.Errorf("OPENAI_BASE_URL not set — export it or add to ~/.kitin/env or .env in the project root")
	}
	if os.Getenv("OPENAI_API_KEY") == "" {
		return fmt.Errorf("OPENAI_API_KEY not set — export it or add to ~/.kitin/env or .env in the project root")
	}
	fmt.Printf("bedrock ready: model=%s endpoint=%s/responses\n", bedrockModel(), strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/"))
	return nil
}

func loadEnvFiles() {
	if home, err := os.UserHomeDir(); err == nil {
		loadEnvFile(filepath.Join(home, ".kitin", "env"))
	}
	loadEnvFile(".env")
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func bedrockTimeout() time.Duration {
	if s := os.Getenv("KITIN_BEDROCK_TIMEOUT_SECONDS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 300 * time.Second
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","model":"%s"}`, bedrockModel())
}

func handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	prompt := buildPrompt(req.Files, req.Modes)
	findings, err := callBedrock(prompt)
	if err != nil {
		http.Error(w, "llm error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	scanID := fmt.Sprintf("%d", time.Now().UnixNano())
	findingsJSON, _ := json.Marshal(findings)
	db.DB.Exec(`INSERT INTO scans (id, token, findings, status) VALUES (?, ?, ?, ?)`,
		scanID, req.Token, string(findingsJSON), "complete")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ScanResponse{
		ID:       scanID,
		Findings: findings,
		Status:   "complete",
	})
}

func handleSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	db.DB.Exec(`INSERT INTO schedules (id, token, path, every, slack, email, discord) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, req.Token, req.Path, req.Every, req.Slack, req.Email, req.Discord)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"scheduled","id":"%s"}`, id)
}

func handleListSchedules(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	rows, err := db.DB.Query(`SELECT id, path, every, slack, email, discord, created_at FROM schedules WHERE token = ?`, token)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Schedule struct {
		ID        string `json:"id"`
		Path      string `json:"path"`
		Every     string `json:"every"`
		Slack     string `json:"slack"`
		Email     string `json:"email"`
		Discord   string `json:"discord"`
		CreatedAt string `json:"created_at"`
	}

	var schedules []Schedule
	for rows.Next() {
		var s Schedule
		rows.Scan(&s.ID, &s.Path, &s.Every, &s.Slack, &s.Email, &s.Discord, &s.CreatedAt)
		schedules = append(schedules, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedules)
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	var findings string
	var createdAt string
	err := db.DB.QueryRow(`SELECT findings, created_at FROM scans WHERE token = ? ORDER BY created_at DESC LIMIT 1`, token).
		Scan(&findings, &createdAt)
	if err != nil {
		http.Error(w, "no scans found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"findings":%s,"created_at":"%s"}`, findings, createdAt)
}

func buildPrompt(files []File, modes []string) string {
	compliance := regulations.FormatForPrompt(modes)
	prompt := fmt.Sprintf(`You are a security code scanner. Analyze the code and return a JSON array of security issues.

Return ONLY a JSON array. No other text. No markdown. No backticks.
Start with [ and end with ].
If no issues found, return [].

Each issue must have these exact fields:
{"file":"filename","line":0,"type":"security","law":"law name","severity":"critical|high|medium|low","message":"description of the issue","fix":"how to fix it","fine":"legal consequence"}

Scan for: hardcoded secrets, SQL injection, XSS, insecure configs, weak crypto, missing encryption.
Also check compliance with these regulations and frameworks: %s.
Map each finding's "law" field to the specific regulation violated.

Code to scan:
`, compliance)
	for _, f := range files {
		prompt += fmt.Sprintf("\n--- %s ---\n%s\n", f.Name, f.Content)
	}
	return prompt
}

func callBedrock(prompt string) ([]Finding, error) {
	baseURL := strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("OPENAI_BASE_URL not set")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	model := bedrockModel()

	body, err := json.Marshal(map[string]interface{}{
		"model": model,
		"input": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/responses", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := bedrockHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach Bedrock: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read Bedrock response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bedrock returned %d: %s", resp.StatusCode, truncate(string(respBody), 500))
	}

	text := extractOutputText(respBody)
	return parseFindings(text)
}

func extractOutputText(respBody []byte) string {
	var result struct {
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return string(respBody)
	}

	var parts []string
	for _, out := range result.Output {
		for _, item := range out.Content {
			if item.Type == "output_text" && item.Text != "" {
				parts = append(parts, item.Text)
			}
		}
	}

	if len(parts) == 0 {
		return string(respBody)
	}

	return strings.Join(parts, "")
}

func parseFindings(response string) ([]Finding, error) {
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	if idx := strings.Index(response, "["); idx != -1 {
		response = response[idx:]
	}
	if idx := strings.LastIndex(response, "]"); idx != -1 {
		response = response[:idx+1]
	}

	var findings []Finding
	if err := json.Unmarshal([]byte(response), &findings); err != nil {
		findings = []Finding{{
			Type:     "general",
			Severity: "info",
			Message:  response,
			Fix:      "review manually",
		}}
	}

	return findings, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
