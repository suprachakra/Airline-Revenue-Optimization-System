package usermgmt

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// UserController handles HTTP requests for user registration, login, and role management.
func UserController(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		if r.URL.Path == "/login" {
			handleLogin(w, r)
		} else if r.URL.Path == "/register" {
			handleRegistration(w, r)
		} else {
			http.NotFound(w, r)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	userID := r.FormValue("user_id")
	// In production, validate credentials and generate a JWT.
	authHandler := NewAuthHandler()
	token, err := authHandler.GenerateToken(userID, []string{"user"})
	if err != nil {
		log.Printf("Login failed for user %s: %v", userID, err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}
	response := map[string]interface{}{
		"token":     token,
		"timestamp": time.Now().UTC(),
	}
	json.NewEncoder(w).Encode(response)
}

func handleRegistration(w http.ResponseWriter, r *http.Request) {
	// Placeholder for registration logic.
	response := map[string]interface{}{
		"message":   "User registered successfully",
		"timestamp": time.Now().UTC(),
	}
	json.NewEncoder(w).Encode(response)
}
