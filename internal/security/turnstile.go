package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
)

const (
	turnstileVerifyURL  = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	turnstileHeader     = "X-Turnstile-Token"
	turnstileTimeout    = 5 * time.Second
	humanCookieName     = "_scenemint_human"
	humanVerifiedHeader = "X-SceneMint-Turnstile-Verified-Until"
)

type turnstileVerifier struct {
	secretKey string
	verifyURL string
	http      *http.Client
}

type turnstileVerifyResponse struct {
	Success bool `json:"success"`
}

func newTurnstileVerifier(secretKey string) *turnstileVerifier {
	return &turnstileVerifier{
		secretKey: strings.TrimSpace(secretKey),
		verifyURL: turnstileVerifyURL,
		http: &http.Client{
			Timeout: turnstileTimeout,
		},
	}
}

func (m *Middleware) Turnstile() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if !m.turnstileEnabled {
				return next(c)
			}
			if m.turnstileSiteKey == "" || m.turnstile.secretKey == "" {
				return reject(c, http.StatusServiceUnavailable, "机器人校验未配置")
			}
			csrfToken, hasCSRFToken := m.csrfCookieToken(c)
			if hasCSRFToken {
				if expiresAt, ok := m.validHumanCookie(c, csrfToken); ok {
					setTurnstileVerifiedHeader(c, expiresAt)
					return next(c)
				}
			}

			token := strings.TrimSpace(c.Request().Header.Get(turnstileHeader))
			if token == "" {
				return reject(c, http.StatusForbidden, "请先完成人机校验")
			}
			ok, err := m.turnstile.verify(c.Request().Context(), token)
			if err != nil {
				return reject(c, http.StatusBadGateway, "人机校验失败，请稍后重试")
			}
			if !ok {
				return reject(c, http.StatusForbidden, "人机校验失败，请重试")
			}
			if hasCSRFToken {
				m.setHumanCookie(c, csrfToken)
			}
			return next(c)
		}
	}
}

func (m *Middleware) csrfCookieToken(c *echo.Context) (string, bool) {
	cookie, err := c.Cookie(csrfCookieName)
	if err != nil || !validToken(cookie.Value) {
		return "", false
	}
	return cookie.Value, true
}

func (m *Middleware) setHumanCookie(c *echo.Context, csrfToken string) {
	expiresAt := time.Now().Add(m.humanTTL).UTC()
	c.SetCookie(&http.Cookie{
		Name:     humanCookieName,
		Value:    m.humanCookieValue(csrfToken, expiresAt),
		Path:     "/",
		MaxAge:   int(m.humanTTL.Seconds()),
		Expires:  expiresAt,
		Secure:   m.secureCookies,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	setTurnstileVerifiedHeader(c, expiresAt)
	c.Response().Header().Add(echo.HeaderVary, echo.HeaderCookie)
}

func (m *Middleware) validHumanCookie(c *echo.Context, csrfToken string) (time.Time, bool) {
	if m.turnstile == nil || m.turnstile.secretKey == "" {
		return time.Time{}, false
	}
	cookie, err := c.Cookie(humanCookieName)
	if err != nil {
		return time.Time{}, false
	}
	expiresRaw, signatureRaw, ok := strings.Cut(cookie.Value, ".")
	if !ok {
		return time.Time{}, false
	}
	expiresUnix, err := strconv.ParseInt(expiresRaw, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	expiresAt := time.Unix(expiresUnix, 0).UTC()
	if !time.Now().Before(expiresAt) {
		return time.Time{}, false
	}
	expected := m.humanCookieSignature(csrfToken, expiresRaw)
	if !hmac.Equal([]byte(signatureRaw), []byte(expected)) {
		return time.Time{}, false
	}
	return expiresAt, true
}

func (m *Middleware) humanCookieValue(csrfToken string, expiresAt time.Time) string {
	expiresRaw := strconv.FormatInt(expiresAt.Unix(), 10)
	return expiresRaw + "." + m.humanCookieSignature(csrfToken, expiresRaw)
}

func (m *Middleware) humanCookieSignature(csrfToken string, expiresRaw string) string {
	mac := hmac.New(sha256.New, []byte(m.turnstile.secretKey))
	mac.Write([]byte(expiresRaw))
	mac.Write([]byte("."))
	mac.Write([]byte(csrfToken))
	return hex.EncodeToString(mac.Sum(nil))
}

func setTurnstileVerifiedHeader(c *echo.Context, expiresAt time.Time) {
	c.Response().Header().Set(humanVerifiedHeader, expiresAt.UTC().Format(time.RFC3339))
}

func (v *turnstileVerifier) verify(ctx context.Context, token string) (bool, error) {
	form := url.Values{}
	form.Set("secret", v.secretKey)
	form.Set("response", strings.TrimSpace(token))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		v.verifyURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return false, err
	}
	req.Header.Set(echo.HeaderContentType, "application/x-www-form-urlencoded")

	resp, err := v.http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return false, fmt.Errorf("turnstile verify status %d", resp.StatusCode)
	}

	var payload turnstileVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, err
	}
	return payload.Success, nil
}
