package handler

import (
	"encoding/json"
	"go-webserver/internal/db"
	"log/slog"
	"net/http"
	"strings"
)

// Auth returns handlers for login and register endpoints.
type Auth struct {
	DB *db.DB
}

// Login handles POST /login — verifies username/password.
func (a *Auth) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		username, password, ok := a.parseCredentials(w, r)
		if !ok {
			return
		}

		result, err := a.DB.Login(username, password)
		if err != nil {
			slog.Error("login failed", "user", username, "error", err)
			a.respondError(w, r, http.StatusInternalServerError, "Internal Server Error")
			return
		}

		switch result {
		case db.LoginSuccess:
			slog.Info("login success", "user", username)
			http.Redirect(w, r, "/welcome.html", http.StatusFound)
		case db.LoginUserNotFound:
			slog.Warn("login failed: user not found", "user", username)
			a.respondError(w, r, http.StatusUnauthorized, "User not found")
		case db.LoginWrongPassword:
			slog.Warn("login failed: wrong password", "user", username)
			a.respondError(w, r, http.StatusUnauthorized, "Wrong password")
		}
	}
}

// Register handles POST /register — creates a new user.
func (a *Auth) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		username, password, ok := a.parseCredentials(w, r)
		if !ok {
			return
		}

		if err := a.DB.Register(username, password); err != nil {
			slog.Error("register failed", "user", username, "error", err)
			a.respondError(w, r, http.StatusForbidden, "Username already exists")
			return
		}

		slog.Info("register success", "user", username)
		http.Redirect(w, r, "/login.html", http.StatusFound)
	}
}

// parseCredentials extracts username/password from form or JSON body.
func (a *Auth) parseCredentials(w http.ResponseWriter, r *http.Request) (username, password string, ok bool) {
	ct := r.Header.Get("Content-Type")

	if strings.HasPrefix(ct, "application/json") {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			a.respondError(w, r, http.StatusBadRequest, "Invalid JSON")
			return "", "", false
		}
		return body.Username, body.Password, true
	}

	// Default: application/x-www-form-urlencoded
	if err := r.ParseForm(); err != nil {
		a.respondError(w, r, http.StatusBadRequest, "Invalid form data")
		return "", "", false
	}
	return r.FormValue("username"), r.FormValue("password"), true
}

// respondError sends an error page matching the C++ error HTML behavior.
func (a *Auth) respondError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	http.ServeFile(w, r, "resources/error.html")
}
