package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length  = 6
)

var db *sql.DB

func init() {
	var err error
	db, err = sql.Open("sqlite3", "./urlshortener.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS urls (
        short_code TEXT PRIMARY KEY,
        original_url TEXT NOT NULL
    )`)
	if err != nil {
		log.Fatal(err)
	}
}

func generateShortCode() string {
	rand.Seed(time.Now().UnixNano())
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	shortCode := generateShortCode()
	_, err := db.Exec("INSERT INTO urls (short_code, original_url) VALUES (?, ?)", shortCode, req.URL)
	if err != nil {
		http.Error(w, "Failed to store URL", http.StatusInternalServerError)
		return
	}

	resp := map[string]string{"short_url": fmt.Sprintf("http://localhost:8080/%s", shortCode)}
	json.NewEncoder(w).Encode(resp)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	shortCode := r.URL.Path[1:]

	var originalURL string
	err := db.QueryRow("SELECT original_url FROM urls WHERE short_code = ?", shortCode).Scan(&originalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Failed to retrieve URL", http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}

func main() {
	http.HandleFunc("/shorten", shortenHandler)
	http.HandleFunc("/", redirectHandler)

	fmt.Println("URL Shortener is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
