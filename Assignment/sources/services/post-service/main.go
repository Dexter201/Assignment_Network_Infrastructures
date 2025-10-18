package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	PORT           = "5000"
	HEADER_USER_ID = "X-User-ID"
)

var (
	POSTGRES_DSN = os.Getenv("POSTGRES_DSN")
)

type Post struct {
	Author    uuid.UUID `json:"author_uuid"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

var db *sql.DB = nil

func getUserIDFromHeader(r *http.Request) (uuid.UUID, error) {
	uid, err := uuid.Parse(r.Header.Get(HEADER_USER_ID))
	if err != nil {
		return uuid.Nil, http.ErrNoCookie
	}
	return uid, nil
}

func createPostHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromHeader(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Content == "" {
		http.Error(w, "Invalid content", http.StatusBadRequest)
		return
	}

	var postID int
	timestamp := time.Now()

	err = db.QueryRow(`
		INSERT INTO posts (user_id, content, timestamp)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, req.Content, timestamp).Scan(&postID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	post := Post{
		Author:    userID,
		Content:   req.Content,
		Timestamp: timestamp,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}

func getPostsHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromHeader(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := db.Query(`
		SELECT user_id, content, timestamp
		FROM posts
		WHERE user_id = $1
		ORDER BY timestamp DESC
	`, userID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.Author, &post.Content, &post.Timestamp); err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		posts = append(posts, post)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func getPostsByUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
		SELECT user_id, content, timestamp
		FROM posts
		WHERE user_id = $1
		ORDER BY timestamp DESC
	`, userID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.Author, &post.Content, &post.Timestamp); err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		posts = append(posts, post)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func setupRoutes() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/posts/me", createPostHandler).Methods("POST")
	r.HandleFunc("/posts/me", getPostsHandler).Methods("GET")
	r.HandleFunc("/posts/{userId}", getPostsByUserHandler).Methods("GET")
	return r
}

func initDB(initSQL string) {
	var err error
	var retries int = 0
	for {
		if retries >= 5 {
			log.Fatalf("Could not connect to database after %d attempts: %v", retries, err)
		}
		retries++
		time.Sleep(2 * time.Second)

		if db == nil {
			db, err = sql.Open("postgres", POSTGRES_DSN)
			if err != nil {
				continue
			}
		}

		err = db.Ping()
		if err != nil {
			continue
		}

		_, err = db.Exec(initSQL)
		if err != nil {
			continue
		}

		break
	}
}

func checkEnv(requiredVars []string) {
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			log.Fatalf("Required environment variable %s is not set", v)
		}
	}
}

func main() {
	checkEnv([]string{"POSTGRES_DSN"})
	initDB(`
		CREATE TABLE IF NOT EXISTS posts (
			id SERIAL PRIMARY KEY,
			user_id UUID NOT NULL,
			content TEXT NOT NULL,
			timestamp TIMESTAMPTZ NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_user_id ON posts(user_id);
		`)
	defer db.Close()

	log.Println("Post service running on port", PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, setupRoutes()))
}
