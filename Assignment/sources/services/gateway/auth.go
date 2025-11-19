package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// represents a authentification Handler
type Handler struct {
	db        *sql.DB
	jwtSecret []byte
}

// represents credentials --> directly implemented from the instructions
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// represents claims --> directly implemented from the instructions
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// create a new auth handler with a given db and a jwt secret
func createAuthHandler(db *sql.DB, jwtSecret []byte) *Handler {
	return &Handler{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

// register a new user
func (handler *Handler) register(writer http.ResponseWriter, receiver *http.Request) {
	var credentials Credentials

	credentials, ok := decodeAndCheck(writer, receiver)
	if !ok {
		return
	}

	hashedPassword, ok := hashPassword(writer, credentials.Password)
	if !ok {
		return
	}

	err := handler.insertNew(writer, hashedPassword, credentials)
	if err != nil {
		return
	}

	writer.WriteHeader(http.StatusCreated)
	json.NewEncoder(writer).Encode(map[string]string{"message": "User registered successfully"})
}

// login with an incoming user
func (handler *Handler) login(writer http.ResponseWriter, receiver *http.Request) {

	credentials, ok := decodeAndCheck(writer, receiver)
	if !ok {
		return
	}

	userID, storedHash, ok := handler.fetchUserCredentials(writer, credentials.Email)
	if !ok {
		return
	}

	if !checkPasswordHash(writer, storedHash, credentials.Password) {
		return
	}

	tokenString, ok := handler.createJWT(writer, userID)
	if !ok {
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(map[string]string{
		"access_token": tokenString,
	})
}

// -------------------- middleware --------------------

// privateUserKey is a custom type to avoid context key collisions since many Hadlers use String as userID keys
// with a unique type collisions are avoided(mostly)
type privateUserKey string

const userIDKey privateUserKey = "userID"

// create a validation middleware
func (header *Handler) validationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, receiver *http.Request) {

		authHeader := receiver.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(writer, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		headerParts := strings.SplitN(authHeader, " ", 2)
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			http.Error(writer, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := headerParts[1]

		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return header.jwtSecret, nil
		})

		if err != nil {
			http.Error(writer, "Error Parsing: "+err.Error(), http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			http.Error(writer, "Invalid token", http.StatusUnauthorized)
			return
		}

		receiver.Header.Set("X-User-ID", claims.UserID)

		//Very useful to have a better logging especially for metrics:
		//Instead of simply have a metrics log: Metrics: POST /api/feed 200 0.0123s
		//We could have something more precise like: user=a0eebc99... POST /api/feed 200 0.0123s
		userContext := context.WithValue(receiver.Context(), userIDKey, claims.UserID)

		callNextHandler(next, writer, receiver.WithContext(userContext))

	})
}

// -------------------- handler utility methods --------------------

// insert a new user in the db --> registration
func (handler *Handler) insertNew(writer http.ResponseWriter, hashedPassword []byte, creds Credentials) error {
	userID := uuid.New().String()

	// Placeholders (like $1) work by separating the SQL command from the data --> prevents SQL injection
	_, err := handler.db.Exec("INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3) ON CONFLICT (email) DO NOTHING",
		userID, creds.Email, string(hashedPassword))

	if err != nil {
		log.Printf("Failed to register user: %v", err)
		http.Error(writer, "Failed to register user", http.StatusInternalServerError)
		return err
	}
	return nil // success
}

// fetch user credentials --> userID, password
func (handler *Handler) fetchUserCredentials(writer http.ResponseWriter, email string) (userID, hashedPassword string, ok bool) {
	err := handler.db.QueryRow("SELECT id, password_hash FROM users WHERE email = $1", email).Scan(&userID, &hashedPassword)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(writer, "Invalid email or password", http.StatusUnauthorized)
		} else {
			log.Printf("Database error during login for email %s: %v", email, err)
			http.Error(writer, "Database error", http.StatusInternalServerError)
		}
		return "", "", false
	}

	return userID, hashedPassword, true // Success
}

// create a new json Tokken
func (handler *Handler) createJWT(writer http.ResponseWriter, userID string) (string, bool) {

	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	// signing the token with HS256 method --> standard
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(handler.jwtSecret)

	if err != nil {
		log.Printf("Failed to create token for user %s: %v", userID, err)
		http.Error(writer, "Failed to create token", http.StatusInternalServerError)
		return "", false
	}

	return tokenString, true
}

// -------------------- auth utilities --------------------

// decode incoming credentials --> check validity
func decodeAndCheck(writer http.ResponseWriter, receiver *http.Request) (Credentials, bool) {
	var credentials Credentials

	err := json.NewDecoder(receiver.Body).Decode(&credentials)
	if err != nil || credentials.Email == "" || credentials.Password == "" {
		http.Error(writer, "Invalid request payload", http.StatusBadRequest)
		return Credentials{}, false
	}

	return credentials, true
}

// hash a password with bcrypt library
func hashPassword(writer http.ResponseWriter, password string) ([]byte, bool) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		http.Error(writer, "Failed to hash password", http.StatusInternalServerError)
		return nil, false
	}
	return hashedPassword, true
}

// chech and compare the correctness of captured hashed password
func checkPasswordHash(writer http.ResponseWriter, hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	if err != nil {
		http.Error(writer, "Invalid email or password", http.StatusUnauthorized)
		return false
	}

	return true
}
