package handler

import (
	"context"
	"errors"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/golang-jwt/jwt/v5"

	"github.com/vantutran2k1/SignalFlow/internal/service"
	"github.com/vantutran2k1/SignalFlow/web"
)

const sessionCookieName = "sf_session"

type SessionHandler struct {
	svc       *service.AuthService
	jwtSecret string
	loginTmpl *template.Template
}

func NewSessionHandler(svc *service.AuthService, jwtSecret string) *SessionHandler {
	tmplFS, err := fs.Sub(web.FS, "templates")
	if err != nil {
		panic(err)
	}
	loginTmpl := template.Must(template.ParseFS(tmplFS, "login.html"))

	return &SessionHandler{
		svc:       svc,
		jwtSecret: jwtSecret,
		loginTmpl: loginTmpl,
	}
}

func (h *SessionHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	h.renderLogin(w, "", "")
}

func (h *SessionHandler) DoLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderLogin(w, "", "invalid form data")
		return
	}
	email := r.FormValue("email")
	password := r.FormValue("password")

	token, err := h.svc.Login(r.Context(), email, password)
	if err != nil {
		msg := "invalid email or password"
		if !errors.Is(err, service.ErrInvalidCredentials) {
			msg = "login failed: " + err.Error()
		}
		h.renderLogin(w, email, msg)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *SessionHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *SessionHandler) renderLogin(w http.ResponseWriter, email, errMsg string) {
	status := http.StatusOK
	if errMsg != "" {
		status = http.StatusUnauthorized
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = h.loginTmpl.Execute(w, map[string]any{
		"Email": email,
		"Error": errMsg,
	})
}

// CookieAuthMiddleware protects browser-facing pages. On a missing or invalid
// session cookie, it redirects the request to /login instead of returning a
// JSON 401 (which would be useless to the browser). Successful auth stores
// the user_id in the request context under the shared userIDKey.
func CookieAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			userID, ok := claims["sub"].(string)
			if !ok {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
