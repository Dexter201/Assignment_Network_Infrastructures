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

type UserProfile struct {
	ID   uuid.UUID `json:"uuid"`
	Name string    `json:"username"`
	Bio  string    `json:"bio"`
}

type FriendRequest struct {
	FriendID uuid.UUID `json:"friend_uuid"`
}

var db *sql.DB = nil

func getUserIDFromHeader(r *http.Request) (uuid.UUID, error) {
	uid, err := uuid.Parse(r.Header.Get(HEADER_USER_ID))
	if err != nil {
		return uuid.Nil, http.ErrNoCookie
	}
	return uid, nil
}

func updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromHeader(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var update UserProfile
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`
		INSERT INTO users (id, name, bio)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, bio = EXCLUDED.bio
	`, userID, update.Name, update.Bio)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	update.ID = userID
	json.NewEncoder(w).Encode(update)
}

func getProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromHeader(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user UserProfile
	err = db.QueryRow(`SELECT id, name, bio FROM users WHERE id = $1`, userID).Scan(&user.ID, &user.Name, &user.Bio)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(user)
}

func getProfileByUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	var user UserProfile
	err := db.QueryRow(`SELECT id, name, bio FROM users WHERE id = $1`, userID).Scan(&user.ID, &user.Name, &user.Bio)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(user)
}

func addFriendHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromHeader(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req FriendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FriendID == userID {
		http.Error(w, "Invalid friend ID", http.StatusBadRequest)
		return
	}

	var exists bool
	err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, req.FriendID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Friend not found", http.StatusNotFound)
		return
	}

	_, err = db.Exec(`
		INSERT INTO friends (user_id, friend_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, req.FriendID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Friend added\n"))
}

func deleteFriendHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromHeader(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req FriendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FriendID == userID {
		http.Error(w, "Invalid friend ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`DELETE FROM friends WHERE user_id = $1 AND friend_id = $2`, userID, req.FriendID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Friend deleted\n"))
}

func getFriendsHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromHeader(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := db.Query(`SELECT friend_id FROM friends WHERE user_id = $1`, userID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var friends []uuid.UUID
	for rows.Next() {
		var f uuid.UUID
		rows.Scan(&f)
		friends = append(friends, f)
	}

	json.NewEncoder(w).Encode(friends)
}

func setupRoutes() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/profile/me", updateProfileHandler).Methods("POST")
	r.HandleFunc("/profile/me", getProfileHandler).Methods("GET")
	r.HandleFunc("/profile/{userId}", getProfileByUserHandler).Methods("GET")
	r.HandleFunc("/friends", addFriendHandler).Methods("POST")
	r.HandleFunc("/friends", deleteFriendHandler).Methods("DELETE")
	r.HandleFunc("/friends", getFriendsHandler).Methods("GET")
	return r
}

func checkEnv(requiredVars []string) {
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			log.Fatalf("Required environment variable %s is not set", v)
		}
	}
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

func main() {
	checkEnv([]string{"POSTGRES_DSN"})

	initDB(`
			CREATE TABLE IF NOT EXISTS users (
				id UUID PRIMARY KEY,
				name TEXT,
				bio TEXT
			);
			CREATE TABLE IF NOT EXISTS friends (
				user_id UUID,
				friend_id UUID,
				PRIMARY KEY (user_id, friend_id),
				FOREIGN KEY (user_id) REFERENCES users(id),
				FOREIGN KEY (friend_id) REFERENCES users(id)
			);
		`)
	defer db.Close()

	log.Println("User service running on port", PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, setupRoutes()))
}
