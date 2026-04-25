package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

func signToken(t *testing.T, secret string, claims jwt.MapClaims, method jwt.SigningMethod) string {
	t.Helper()
	tok := jwt.NewWithClaims(method, claims)
	signed, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

func captureUserID(out *string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*out = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}
}

// ---------------- Bearer middleware ----------------

func TestAuthMiddleware_HappyPath(t *testing.T) {
	token := signToken(t, testSecret, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}, jwt.SigningMethodHS256)

	var seenUserID string
	h := AuthMiddleware(testSecret)(captureUserID(&seenUserID))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if seenUserID != "user-123" {
		t.Errorf("userID in ctx = %q, want user-123", seenUserID)
	}
}

func TestAuthMiddleware_Rejects(t *testing.T) {
	wrongSecretToken := signToken(t, "other-secret", jwt.MapClaims{"sub": "u"}, jwt.SigningMethodHS256)
	expiredToken := signToken(t, testSecret, jwt.MapClaims{
		"sub": "u",
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	}, jwt.SigningMethodHS256)
	noSubToken := signToken(t, testSecret, jwt.MapClaims{
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}, jwt.SigningMethodHS256)

	cases := []struct {
		name   string
		header string
	}{
		{"missing header", ""},
		{"wrong scheme", "Basic abc"},
		{"malformed", "Bearer"},
		{"garbage token", "Bearer not-a-token"},
		{"wrong secret", "Bearer " + wrongSecretToken},
		{"expired token", "Bearer " + expiredToken},
		{"missing sub", "Bearer " + noSubToken},
	}

	called := false
	h := AuthMiddleware(testSecret)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rr.Code)
			}
			if called {
				t.Error("downstream handler must not run when auth fails")
			}
		})
	}
}

// ---------------- Cookie middleware ----------------

func TestCookieAuthMiddleware_HappyPath(t *testing.T) {
	token := signToken(t, testSecret, jwt.MapClaims{
		"sub": "user-9",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}, jwt.SigningMethodHS256)

	var seenUserID string
	h := CookieAuthMiddleware(testSecret)(captureUserID(&seenUserID))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if seenUserID != "user-9" {
		t.Errorf("userID in ctx = %q, want user-9", seenUserID)
	}
}

func TestCookieAuthMiddleware_RedirectsToLogin(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*http.Request)
	}{
		{"no cookie", func(*http.Request) {}},
		{"garbage cookie", func(r *http.Request) {
			r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "not-a-jwt"})
		}},
		{"wrong secret", func(r *http.Request) {
			tok := signToken(t, "other-secret", jwt.MapClaims{"sub": "u"}, jwt.SigningMethodHS256)
			r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: tok})
		}},
	}
	called := false
	h := CookieAuthMiddleware(testSecret)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tc.setup(req)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusSeeOther {
				t.Errorf("status = %d, want 303 SeeOther", rr.Code)
			}
			if loc := rr.Header().Get("Location"); !strings.HasPrefix(loc, "/login") {
				t.Errorf("Location = %q, want /login", loc)
			}
			if called {
				t.Error("downstream handler must not run when cookie auth fails")
			}
		})
	}
}

// ---------------- Algorithm confusion guard ----------------

// HS-signing the parsed token requires HMAC. A forged token claiming "alg: none"
// or "alg: RS256" must be rejected, otherwise an attacker can mint tokens.
func TestAuthMiddleware_RejectsNonHMAC(t *testing.T) {
	// alg=none token: header.payload.<empty signature>. Easy to construct manually.
	header := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0"              // {"alg":"none","typ":"JWT"}
	payload := "eyJzdWIiOiJhdHRhY2tlciIsImV4cCI6OTk5OTk5OTk5OX0" // {"sub":"attacker","exp":9999999999}
	noneTok := header + "." + payload + "."

	h := AuthMiddleware(testSecret)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("alg=none token must not pass auth")
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+noneTok)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("alg=none accepted: status = %d", rr.Code)
	}
}
