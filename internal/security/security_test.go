package security

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"scenemint/internal/config"

	"github.com/labstack/echo/v5"
	"github.com/sunls24/gox/server"
)

func TestSourceGuardAllowsSameOrigin(t *testing.T) {
	sec := New(config.Security{})
	e := echo.New()
	e.GET("/api/protected", okHandler, sec.SourceGuard())

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/protected", nil)
	req.Host = "example.com"
	req.Header.Set(echo.HeaderOrigin, "http://example.com")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestSourceGuardAllowsForwardedHTTPSOrigin(t *testing.T) {
	sec := New(config.Security{})
	e := echo.New()
	e.POST("/api/protected", okHandler, sec.SourceGuard())

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", nil)
	req.Host = "example.com"
	req.Header.Set(echo.HeaderXForwardedProto, "https")
	req.Header.Set(echo.HeaderOrigin, "https://example.com")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestSourceGuardRejectsUntrustedOrigin(t *testing.T) {
	sec := New(config.Security{})
	e := echo.New()
	e.GET("/api/protected", okHandler, sec.SourceGuard())

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/protected", nil)
	req.Host = "example.com"
	req.Header.Set(echo.HeaderOrigin, "https://evil.example")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestSourceGuardRejectsUnsafeRequestWithoutSource(t *testing.T) {
	sec := New(config.Security{})
	e := echo.New()
	e.POST("/api/protected", okHandler, sec.SourceGuard())

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", nil)
	req.Host = "example.com"
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestSourceGuardRejectsCrossSiteFetchMetadata(t *testing.T) {
	sec := New(config.Security{})
	e := echo.New()
	e.GET("/api/protected", okHandler, sec.SourceGuard())

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/protected", nil)
	req.Host = "example.com"
	req.Header.Set(echo.HeaderSecFetchSite, "cross-site")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestCSRFRejectsMissingToken(t *testing.T) {
	sec := New(config.Security{})
	e := echo.New()
	e.POST("/api/protected", okHandler, sec.SourceGuard(), sec.CSRF())

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", nil)
	req.Host = "example.com"
	req.Header.Set(echo.HeaderOrigin, "http://example.com")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if got := rec.Header().Get(csrfErrorHeader); got != "1" {
		t.Fatalf("%s = %q, want %q", csrfErrorHeader, got, "1")
	}
}

func TestCSRFAllowsValidToken(t *testing.T) {
	sec := New(config.Security{})
	e := echo.New()
	e.GET("/api/session", sec.Session, sec.SourceGuard())
	e.POST("/api/protected", okHandler, sec.SourceGuard(), sec.CSRF())

	sessionReq := httptest.NewRequest(http.MethodGet, "http://example.com/api/session", nil)
	sessionReq.Host = "example.com"
	sessionRec := httptest.NewRecorder()
	e.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("session status = %d, want %d; body: %s", sessionRec.Code, http.StatusOK, sessionRec.Body.String())
	}

	var envelope struct {
		Data SessionResponse `json:"data"`
	}
	if err := json.Unmarshal(sessionRec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode session response: %v", err)
	}
	if envelope.Data.CSRFToken == "" {
		t.Fatal("csrfToken is empty")
	}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", nil)
	req.Host = "example.com"
	req.Header.Set(echo.HeaderOrigin, "http://example.com")
	req.Header.Set(echo.HeaderXCSRFToken, envelope.Data.CSRFToken)
	for _, cookie := range sessionRec.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestTurnstileAllowsWhenDisabled(t *testing.T) {
	sec := New(config.Security{})
	e := echo.New()
	e.POST("/api/protected", okHandler, sec.Turnstile())

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestTurnstileRejectsMissingToken(t *testing.T) {
	sec := New(config.Security{
		TurnstileEnabled:   true,
		TurnstileSiteKey:   "site-key",
		TurnstileSecretKey: "secret-key",
	})
	e := echo.New()
	e.POST("/api/protected", okHandler, sec.Turnstile())

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestTurnstileAllowsValidToken(t *testing.T) {
	var verifyCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		verifyCalled = true
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse verify form: %v", err)
		}
		if got := r.Form.Get("secret"); got != "secret-key" {
			t.Fatalf("secret = %q, want %q", got, "secret-key")
		}
		if got := r.Form.Get("response"); got != "valid-token" {
			t.Fatalf("response = %q, want %q", got, "valid-token")
		}
		w.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer ts.Close()

	sec := New(config.Security{
		TurnstileEnabled:   true,
		TurnstileSiteKey:   "site-key",
		TurnstileSecretKey: "secret-key",
	})
	sec.turnstile.verifyURL = ts.URL
	e := echo.New()
	e.POST("/api/protected", okHandler, sec.Turnstile())

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", strings.NewReader("{}"))
	req.Header.Set(turnstileHeader, "valid-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !verifyCalled {
		t.Fatal("verify endpoint was not called")
	}
}

func TestTurnstileSetsHumanCookieAfterValidToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer ts.Close()

	sec := New(config.Security{
		TurnstileEnabled:   true,
		TurnstileSiteKey:   "site-key",
		TurnstileSecretKey: "secret-key",
	})
	sec.turnstile.verifyURL = ts.URL
	e := echo.New()
	e.POST("/api/protected", okHandler, sec.Turnstile())

	csrfToken, err := newToken()
	if err != nil {
		t.Fatalf("new csrf token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", nil)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	req.Header.Set(turnstileHeader, "valid-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	humanCookie := findCookie(rec.Result().Cookies(), humanCookieName)
	if humanCookie == nil {
		t.Fatalf("%s cookie was not set", humanCookieName)
	}
	if humanCookie.MaxAge != int(humanCookieTTL.Seconds()) {
		t.Fatalf("%s MaxAge = %d, want %d", humanCookieName, humanCookie.MaxAge, int(humanCookieTTL.Seconds()))
	}
	if got := rec.Header().Get(humanVerifiedHeader); got == "" {
		t.Fatalf("%s header is empty", humanVerifiedHeader)
	}
}

func TestTurnstileAllowsHumanCookieWithoutToken(t *testing.T) {
	var verifyCalls int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		verifyCalls++
		w.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer ts.Close()

	sec := New(config.Security{
		TurnstileEnabled:   true,
		TurnstileSiteKey:   "site-key",
		TurnstileSecretKey: "secret-key",
	})
	sec.turnstile.verifyURL = ts.URL
	e := echo.New()
	e.POST("/api/protected", okHandler, sec.Turnstile())

	csrfToken, err := newToken()
	if err != nil {
		t.Fatalf("new csrf token: %v", err)
	}
	firstReq := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", nil)
	firstReq.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	firstReq.Header.Set(turnstileHeader, "valid-token")
	firstRec := httptest.NewRecorder()
	e.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d; body: %s", firstRec.Code, http.StatusOK, firstRec.Body.String())
	}
	humanCookie := findCookie(firstRec.Result().Cookies(), humanCookieName)
	if humanCookie == nil {
		t.Fatalf("%s cookie was not set", humanCookieName)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", nil)
	secondReq.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	secondReq.AddCookie(humanCookie)
	secondRec := httptest.NewRecorder()
	e.ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d; body: %s", secondRec.Code, http.StatusOK, secondRec.Body.String())
	}
	if verifyCalls != 1 {
		t.Fatalf("verify calls = %d, want %d", verifyCalls, 1)
	}
}

func TestTurnstileRejectsHumanCookieForDifferentCSRFToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer ts.Close()

	sec := New(config.Security{
		TurnstileEnabled:   true,
		TurnstileSiteKey:   "site-key",
		TurnstileSecretKey: "secret-key",
	})
	sec.turnstile.verifyURL = ts.URL
	e := echo.New()
	e.POST("/api/protected", okHandler, sec.Turnstile())

	csrfToken, err := newToken()
	if err != nil {
		t.Fatalf("new csrf token: %v", err)
	}
	firstReq := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", nil)
	firstReq.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	firstReq.Header.Set(turnstileHeader, "valid-token")
	firstRec := httptest.NewRecorder()
	e.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d; body: %s", firstRec.Code, http.StatusOK, firstRec.Body.String())
	}
	humanCookie := findCookie(firstRec.Result().Cookies(), humanCookieName)
	if humanCookie == nil {
		t.Fatalf("%s cookie was not set", humanCookieName)
	}

	otherCSRFToken, err := newToken()
	if err != nil {
		t.Fatalf("new other csrf token: %v", err)
	}
	secondReq := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", nil)
	secondReq.AddCookie(&http.Cookie{Name: csrfCookieName, Value: otherCSRFToken})
	secondReq.AddCookie(humanCookie)
	secondRec := httptest.NewRecorder()
	e.ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusForbidden {
		t.Fatalf("second status = %d, want %d; body: %s", secondRec.Code, http.StatusForbidden, secondRec.Body.String())
	}
}

func TestTurnstileRejectsVerifyUpstreamError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer ts.Close()

	sec := New(config.Security{
		TurnstileEnabled:   true,
		TurnstileSiteKey:   "site-key",
		TurnstileSecretKey: "secret-key",
	})
	sec.turnstile.verifyURL = ts.URL
	e := echo.New()
	e.POST("/api/protected", okHandler, sec.Turnstile())

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/protected", strings.NewReader("{}"))
	req.Header.Set(turnstileHeader, "valid-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadGateway, rec.Body.String())
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func okHandler(c *echo.Context) error {
	return c.JSON(http.StatusOK, server.Envelope{Message: "ok"})
}
