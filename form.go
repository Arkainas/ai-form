package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type FormResponse struct {
	JobRole            string `json:"jobRole"`
	ExperienceYears    string `json:"experienceYears"`
	UsesAI             string `json:"usesAI"`
	PrimaryLLM         string `json:"primaryLLM"`
	MainUsage          string `json:"mainUsage"`
	AIHelpfulForJunior string `json:"aiHelpfulForJunior"`
	CodeQualityDown    string `json:"codeQualityDown"`
	RelyingTooMuch     string `json:"relyingTooMuch"`
	WillReplaceDevs    string `json:"willReplaceDevs"`
	JobRoleOther       string `json:"jobRoleOther,omitempty"`
	PrimaryLLMOther    string `json:"primaryLLMOther,omitempty"`
	MainUsageOther     string `json:"mainUsageOther,omitempty"`
}

// Rate limiter structure
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	requests := rl.requests[ip]

	// Remove old requests
	var validRequests []time.Time
	for _, t := range requests {
		if now.Sub(t) <= rl.window {
			validRequests = append(validRequests, t)
		}
	}

	// Check if we're over the limit
	if len(validRequests) >= rl.limit {
		return false
	}

	// Add new request
	validRequests = append(validRequests, now)
	rl.requests[ip] = validRequests
	return true
}

var (
	db          *sql.DB
	rateLimiter = NewRateLimiter(10, time.Minute) // 10 requests per minute per IP
)

func initDB() {
	var err error
	if err := os.MkdirAll("database", 0755); err != nil {
		log.Fatal(err)
	}

	db, err = sql.Open("sqlite", filepath.Join("database", "responses.db"))
	if err != nil {
		log.Fatal(err)
	}

	// Create responses table with additional columns for "Others" responses
	createTable := `
	CREATE TABLE IF NOT EXISTS responses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_role TEXT,
		experience_years TEXT,
		uses_ai TEXT,
		primary_llm TEXT,
		main_usage TEXT,
		ai_helpful_junior TEXT,
		code_quality_down TEXT,
		relying_too_much TEXT,
		will_replace_devs TEXT,
		job_role_other TEXT,
		primary_llm_other TEXT,
		main_usage_other TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}
}

func handleSubmission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get client IP
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	// Check rate limit
	if !rateLimiter.Allow(ip) {
		http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
		return
	}

	// Parse the form data
	var response FormResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if response.JobRole == "" || response.ExperienceYears == "" || response.UsesAI == "" ||
		response.PrimaryLLM == "" || response.MainUsage == "" || response.AIHelpfulForJunior == "" ||
		response.CodeQualityDown == "" || response.RelyingTooMuch == "" || response.WillReplaceDevs == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// Insert response into database
	_, err := db.Exec(`
		INSERT INTO responses (
			job_role, experience_years, uses_ai, primary_llm,
			main_usage, ai_helpful_junior, code_quality_down, relying_too_much,
			will_replace_devs, job_role_other, primary_llm_other, main_usage_other
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		response.JobRole, response.ExperienceYears, response.UsesAI,
		response.PrimaryLLM, response.MainUsage, response.AIHelpfulForJunior,
		response.CodeQualityDown, response.RelyingTooMuch, response.WillReplaceDevs,
		response.JobRoleOther, response.PrimaryLLMOther, response.MainUsageOther,
	)

	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to save response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

type QuestionStats struct {
	Labels []string `json:"labels"`
	Data   []int    `json:"data"`
}

func getQuestionStats(w http.ResponseWriter, r *http.Request, column string) {
	log.Printf("Fetching stats for column: %s", column)

	// For columns that have "Others" responses, we need to group them together
	query := `
		SELECT 
			CASE 
				WHEN ` + column + ` = 'Others' THEN 'Others'
				ELSE ` + column + `
			END as label,
			COUNT(*) as count 
		FROM responses 
		GROUP BY 
			CASE 
				WHEN ` + column + ` = 'Others' THEN 'Others'
				ELSE ` + column + `
			END
		ORDER BY count DESC`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Database query error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stats QuestionStats
	for rows.Next() {
		var label string
		var count int
		if err := rows.Scan(&label, &count); err != nil {
			log.Printf("Row scan error: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		stats.Labels = append(stats.Labels, label)
		stats.Data = append(stats.Data, count)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Rows error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	log.Printf("Stats for %s: %+v", column, stats)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("JSON encoding error: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

func handleJobRoleStats(w http.ResponseWriter, r *http.Request) {
	getQuestionStats(w, r, "job_role")
}

func handleExperienceStats(w http.ResponseWriter, r *http.Request) {
	getQuestionStats(w, r, "experience_years")
}

func handleUsesAIStats(w http.ResponseWriter, r *http.Request) {
	getQuestionStats(w, r, "uses_ai")
}

func handlePrimaryLLMStats(w http.ResponseWriter, r *http.Request) {
	getQuestionStats(w, r, "primary_llm")
}

func handleMainUsageStats(w http.ResponseWriter, r *http.Request) {
	getQuestionStats(w, r, "main_usage")
}

func handleAIHelpfulStats(w http.ResponseWriter, r *http.Request) {
	getQuestionStats(w, r, "ai_helpful_junior")
}

func handleCodeQualityStats(w http.ResponseWriter, r *http.Request) {
	getQuestionStats(w, r, "code_quality_down")
}

func handleRelyingTooMuchStats(w http.ResponseWriter, r *http.Request) {
	getQuestionStats(w, r, "relying_too_much")
}

func handleWillReplaceStats(w http.ResponseWriter, r *http.Request) {
	getQuestionStats(w, r, "will_replace_devs")
}

func main() {
	initDB()
	defer db.Close()

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Handle form submission
	http.HandleFunc("/api/submit", handleSubmission)

	// Handle statistics endpoints
	http.HandleFunc("/api/stats/job-role", handleJobRoleStats)
	http.HandleFunc("/api/stats/experience", handleExperienceStats)
	http.HandleFunc("/api/stats/uses-ai", handleUsesAIStats)
	http.HandleFunc("/api/stats/primary-llm", handlePrimaryLLMStats)
	http.HandleFunc("/api/stats/main-usage", handleMainUsageStats)
	http.HandleFunc("/api/stats/ai-helpful", handleAIHelpfulStats)
	http.HandleFunc("/api/stats/code-quality", handleCodeQualityStats)
	http.HandleFunc("/api/stats/relying-too-much", handleRelyingTooMuchStats)
	http.HandleFunc("/api/stats/will-replace", handleWillReplaceStats)

	// Serve the main page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/index.html"))
		tmpl.Execute(w, nil)
	})

	log.Println("Server starting on :80...")
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatal(err)
	}
}
