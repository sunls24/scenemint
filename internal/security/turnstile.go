package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/sunls24/gox/server"
)

const (
	turnstileCookieName    = "_scenemint_turnstile"
	turnstileVerifyURL     = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	turnstileTimeout       = 10 * time.Second
	maxTurnstileTokenBytes = 2048
)

type TurnstileStatusResponse struct {
	Enabled  bool   `json:"enabled"`
	Verified bool   `json:"verified"`
	SiteKey  string `json:"siteKey"`
}

type turnstileVerifyRequest struct {
	Token string `json:"token"`
}

type turnstileVerifier struct {
	secretKey string
	verifyURL string
	http      *http.Client
}

func newTurnstileVerifier(secretKey string) *turnstileVerifier {
	return &turnstileVerifier{
		secretKey: secretKey,
		verifyURL: turnstileVerifyURL,
		http:      &http.Client{Timeout: turnstileTimeout},
	}
}

func (m *Middleware) TurnstileStatus(c *echo.Context) error {
	return c.JSON(http.StatusOK, server.Envelope{
		Message: "ok",
		Data: TurnstileStatusResponse{
			Enabled:  m.turnstileEnabled,
			Verified: m.turnstileEnabled && m.turnstileVerified(c.Request()),
			SiteKey:  m.turnstileSiteKey,
		},
	})
}

func (m *Middleware) VerifyTurnstile(c *echo.Context) error {
	var req turnstileVerifyRequest
	if err := c.Bind(&req); err != nil {
		return reject(c, http.StatusBadRequest, "请求参数错误")
	}
	token := strings.TrimSpace(req.Token)
	if token == "" || len(token) > maxTurnstileTokenBytes {
		return reject(c, http.StatusBadRequest, "请求参数错误")
	}

	ok, err := m.turnstile.verify(c.Request().Context(), token)
	if err != nil {
		return reject(c, http.StatusBadGateway, "人机验证失败，请稍后重试")
	}
	if !ok {
		return reject(c, http.StatusUnauthorized, "人机验证失败")
	}

	c.SetCookie(m.newTurnstileCookie(time.Now()))
	return c.JSON(http.StatusOK, server.Envelope{Message: "ok"})
}

func (m *Middleware) Turnstile() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if m.turnstileEnabled && !m.turnstileVerified(c.Request()) {
				return reject(c, http.StatusUnauthorized, "请先完成人机验证")
			}
			return next(c)
		}
	}
}

func (m *Middleware) turnstileVerified(req *http.Request) bool {
	cookie, err := req.Cookie(turnstileCookieName)
	if err != nil {
		return false
	}

	expiresRaw, signatureRaw, ok := strings.Cut(cookie.Value, ".")
	if !ok {
		return false
	}
	expires, err := strconv.ParseInt(expiresRaw, 10, 64)
	if err != nil || time.Now().Unix() >= expires {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(signatureRaw)
	if err != nil {
		return false
	}
	return hmac.Equal(signature, m.signTurnstileCookie(expiresRaw))
}

func (m *Middleware) newTurnstileCookie(now time.Time) *http.Cookie {
	expires := now.Add(m.turnstileTTL)
	payload := strconv.FormatInt(expires.Unix(), 10)
	signature := base64.RawURLEncoding.EncodeToString(m.signTurnstileCookie(payload))
	return &http.Cookie{
		Name:     turnstileCookieName,
		Value:    payload + "." + signature,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(m.turnstileTTL.Seconds()),
		HttpOnly: true,
		Secure:   m.secureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}

func (m *Middleware) signTurnstileCookie(payload string) []byte {
	mac := hmac.New(sha256.New, []byte(m.turnstile.secretKey))
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func (v *turnstileVerifier) verify(ctx context.Context, token string) (bool, error) {
	form := url.Values{
		"secret":   {v.secretKey},
		"response": {token},
	}
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
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("turnstile siteverify returned %s", resp.Status)
	}

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Success, nil
}
