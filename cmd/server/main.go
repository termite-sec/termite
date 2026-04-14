package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

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

func main() {
	initDB()

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/scan", handleScan)
	http.HandleFunc("/schedule", handleSchedule)
	http.HandleFunc("/schedules", handleListSchedules)
	http.HandleFunc("/report", handleReport)

	port := "8080"
	fmt.Printf("termite server running on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Println("server error:", err)
		os.Exit(1)
	}
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "/var/lib/termite/termite.db")
	if err != nil {
		fmt.Println("db error:", err)
		os.Exit(1)
	}

	db.Exec(`
		CREATE TABLE IF NOT EXISTS schedules (
			id         TEXT PRIMARY KEY,
			token      TEXT NOT NULL,
			path       TEXT NOT NULL,
			every      TEXT NOT NULL,
			slack      TEXT,
			email      TEXT,
			discord    TEXT,
			last_run   DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS scans (
			id          TEXT PRIMARY KEY,
			token       TEXT NOT NULL,
			findings    TEXT,
			status      TEXT,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status":"ok","model":"qwen2.5-coder:7b"}`)
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
	findings, err := callOllama(prompt)
	if err != nil {
		http.Error(w, "ollama error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	scanID := fmt.Sprintf("%d", time.Now().UnixNano())
	findingsJSON, _ := json.Marshal(findings)
	db.Exec(`INSERT INTO scans (id, token, findings, status) VALUES (?, ?, ?, ?)`,
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
	db.Exec(`INSERT INTO schedules (id, token, path, every, slack, email, discord) VALUES (?, ?, ?, ?, ?, ?, ?)`,
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

	rows, err := db.Query(`SELECT id, path, every, slack, email, discord, created_at FROM schedules WHERE token = ?`, token)
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
	err := db.QueryRow(`SELECT findings, created_at FROM scans WHERE token = ? ORDER BY created_at DESC LIMIT 1`, token).
		Scan(&findings, &createdAt)
	if err != nil {
		http.Error(w, "no scans found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"findings":%s,"created_at":"%s"}`, findings, createdAt)
}

func buildPrompt(files []File, modes []string) string {
	prompt := `You are a security code scanner. Analyze the code and return a JSON array of security issues.

Return ONLY a JSON array. No other text. No markdown. No backticks.
Start with [ and end with ].
If no issues found, return [].

Each issue must have these exact fields:
{"file":"filename","line":0,"type":"security","law":"law name","severity":"critical|high|medium|low","message":"description of the issue","fix":"how to fix it","fine":"legal consequence"}

Scan for: hardcoded secrets, SQL injection, XSS, GDPR violations, HIPAA violations, CCPA violations, insecure configs, weak crypto, missing encryption.

Code to scan:
`
	for _, f := range files {
		prompt += fmt.Sprintf("\n--- %s ---\n%s\n", f.Name, f.Content)
	}
	return prompt
}

func callOllama(prompt string) ([]Finding, error) {
	ollamaURL := "http://192.168.122.1:11434/api/generate"

	body, _ := json.Marshal(map[string]interface{}{
		"model":  "qwen2.5-coder:7b",
		"prompt": prompt,
		"stream": false,
		"temperature": 0.1,
	})

	resp, err := http.Post(ollamaURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("cannot reach ollama: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("cannot decode ollama response: %v", err)
	}

	response, ok := result["response"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid ollama response")
	}

	// Clean markdown fences
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Extract JSON array
	if idx := strings.Index(response, "["); idx != -1 {
		response = response[idx:]
	}
	if idx := strings.LastIndex(response, "]"); idx != -1 {
		response = response[:idx+1]
	}

	var findings []Finding
	// Add this right after the cleaning code, before json.Unmarshal
	fmt.Printf("DEBUG RAW RESPONSE: %s\n", response)
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
