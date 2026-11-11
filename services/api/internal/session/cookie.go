package session

import (
	"net/http"
	"strings"
	"time"
)

const (
	CookieNameDev           = "xynoss_session"
	CookieNameProd          = "__Host-xynoss_session"
	CookieNameBetterAuth    = "better-auth.session_token"
	CookieNameBetterAuthSec = "__Secure-better-auth.session_token"
)

// GetCookieName returns the appropriate cookie name depending on environment.
func GetCookieName(isProduction bool) string {
	if isProduction {
		return CookieNameProd
	}
	return CookieNameDev
}

// SetSessionCookie sets the secure HTTP-only session cookie on the response.
func SetSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time, isProduction bool) {
	name := GetCookieName(isProduction)

	cookie := &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
}

// ClearSessionCookie writes an expired cookie to clear the session on the client.
func ClearSessionCookie(w http.ResponseWriter, isProduction bool) {
	name := GetCookieName(isProduction)

	cookie := &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
}

// ExtractSessionToken retrieves the session token from cookies or Authorization header.
func ExtractSessionToken(r *http.Request) string {
	// 1. Try Authorization: Bearer <token> first (explicit client header)
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		tok := strings.TrimSpace(authHeader[7:])
		tok = strings.Trim(tok, `"`)
		if tok != "" {
			return tok
		}
	}

	// 2. Try production cookie
	if c, err := r.Cookie(CookieNameProd); err == nil && strings.TrimSpace(c.Value) != "" {
		return strings.TrimSpace(c.Value)
	}

	// 3. Try dev cookie
	if c, err := r.Cookie(CookieNameDev); err == nil && strings.TrimSpace(c.Value) != "" {
		return strings.TrimSpace(c.Value)
	}

	// 4. Try Neon Auth / Better Auth cookies
	if c, err := r.Cookie(CookieNameBetterAuthSec); err == nil && strings.TrimSpace(c.Value) != "" {
		return strings.TrimSpace(c.Value)
	}
	if c, err := r.Cookie(CookieNameBetterAuth); err == nil && strings.TrimSpace(c.Value) != "" {
		return strings.TrimSpace(c.Value)
	}

	return ""
}
