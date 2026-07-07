package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"time"

	"scenemint/internal/config"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/sunls24/gox/server"
)

const (
	csrfCookieName  = "_scenemint_csrf"
	csrfErrorHeader = "X-SceneMint-CSRF-Error"
	csrfTokenBytes  = 32
	maxBodyBytes    = 16 << 20
)

type SessionResponse struct {
	CSRFToken              string `json:"csrfToken"`
	TurnstileEnabled       bool   `json:"turnstileEnabled"`
	TurnstileSiteKey       string `json:"turnstileSiteKey,omitempty"`
	TurnstileVerifiedUntil string `json:"turnstileVerifiedUntil,omitempty"`
}

type Middleware struct {
	secureCookies    bool
	turnstileEnabled bool
	turnstileSiteKey string
	turnstile        *turnstileVerifier
}

func New(cfg config.Security) *Middleware {
	return &Middleware{
		secureCookies:    cfg.SecureCookies,
		turnstileEnabled: cfg.TurnstileEnabled,
		turnstileSiteKey: strings.TrimSpace(cfg.TurnstileSiteKey),
		turnstile:        newTurnstileVerifier(cfg.TurnstileSecretKey),
	}
}

func Headers() echo.MiddlewareFunc {
	return middleware.SecureWithConfig(middleware.SecureConfig{
		XFrameOptions:         "DENY",
		ContentSecurityPolicy: "frame-ancestors 'none'",
		ContentTypeNosniff:    "nosniff",
		ReferrerPolicy:        "same-origin",
	})
}

func BodyLimit() echo.MiddlewareFunc {
	return middleware.BodyLimit(maxBodyBytes)
}

func (m *Middleware) SourceGuard() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if !m.allowedSource(c) {
				return reject(c, http.StatusForbidden, "请求来源不被允许")
			}
			return next(c)
		}
	}
}

func (m *Middleware) CSRF() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			cookie, err := c.Cookie(csrfCookieName)
			if err != nil || !validToken(cookie.Value) {
				return rejectCSRF(c)
			}
			token := strings.TrimSpace(c.Request().Header.Get(echo.HeaderXCSRFToken))
			if !sameToken(cookie.Value, token) {
				return rejectCSRF(c)
			}
			return next(c)
		}
	}
}

func (m *Middleware) Session(c *echo.Context) error {
	token, err := m.ensureToken(c)
	if err != nil {
		return reject(c, http.StatusInternalServerError, "会话创建失败")
	}
	var verifiedUntil string
	if expiresAt, ok := m.validHumanCookie(c, token); ok {
		verifiedUntil = expiresAt.Format(time.RFC3339)
		setTurnstileVerifiedHeader(c, expiresAt)
	}
	return c.JSON(http.StatusOK, server.Envelope{
		Message: "ok",
		Data: SessionResponse{
			CSRFToken:              token,
			TurnstileEnabled:       m.turnstileEnabled,
			TurnstileSiteKey:       m.turnstileSiteKey,
			TurnstileVerifiedUntil: verifiedUntil,
		},
	})
}

func (m *Middleware) ensureToken(c *echo.Context) (string, error) {
	if cookie, err := c.Cookie(csrfCookieName); err == nil && validToken(cookie.Value) {
		m.setTokenCookie(c, cookie.Value)
		return cookie.Value, nil
	}

	token, err := newToken()
	if err != nil {
		return "", err
	}
	m.setTokenCookie(c, token)
	return token, nil
}

func (m *Middleware) setTokenCookie(c *echo.Context, token string) {
	c.SetCookie(&http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		Expires:  time.Now().Add(24 * time.Hour),
		Secure:   m.secureCookies,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	c.Response().Header().Add(echo.HeaderVary, echo.HeaderCookie)
}

func (m *Middleware) allowedSource(c *echo.Context) bool {
	req := c.Request()
	secFetchSite := strings.ToLower(strings.TrimSpace(req.Header.Get(echo.HeaderSecFetchSite)))
	if secFetchSite == "cross-site" {
		return false
	}

	origin := strings.TrimSpace(req.Header.Get(echo.HeaderOrigin))
	if origin != "" {
		return sameOrigin(c, origin)
	}

	referer := strings.TrimSpace(req.Header.Get("Referer"))
	if referer != "" {
		refererOrigin, ok := refererOrigin(referer)
		return ok && sameOrigin(c, refererOrigin)
	}

	return isSafeMethod(req.Method)
}

func sameOrigin(c *echo.Context, origin string) bool {
	normalized, ok := normalizeOrigin(origin)
	return ok && normalized == requestOrigin(c)
}

func requestOrigin(c *echo.Context) string {
	return strings.ToLower(c.Scheme()) + "://" + strings.ToLower(c.Request().Host)
}

func normalizeOrigin(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	return scheme + "://" + strings.ToLower(parsed.Host), true
}

func refererOrigin(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	return normalizeOrigin(parsed.Scheme + "://" + parsed.Host)
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func newToken() (string, error) {
	var data [csrfTokenBytes]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(data[:]), nil
}

func validToken(token string) bool {
	if len(token) != csrfTokenBytes*2 {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}

func sameToken(expected string, actual string) bool {
	if !validToken(actual) || len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func reject(c *echo.Context, status int, message string) error {
	return c.JSON(status, server.ErrMsg(message).Envelope())
}

func rejectCSRF(c *echo.Context) error {
	c.Response().Header().Set(csrfErrorHeader, "1")
	return reject(c, http.StatusForbidden, "会话校验失败，请刷新页面后重试")
}
