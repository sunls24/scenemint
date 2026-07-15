package security

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"scenemint/internal/config"

	"github.com/labstack/echo/v5"
)

func TestTurnstileAllowsWhenDisabled(t *testing.T) {
	sec := New(config.Security{})
	e := echo.New()
	e.GET("/api/protected", okHandler, sec.Turnstile())

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/protected", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestTurnstileStatus(t *testing.T) {
	sec := newTestTurnstile(time.Hour)
	e := echo.New()
	e.GET("/api/turnstile/status", sec.TurnstileStatus)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/turnstile/status", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body struct {
		Data TurnstileStatusResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Data.Enabled || body.Data.Verified || body.Data.SiteKey != "site-key" {
		t.Fatalf("unexpected status: %+v", body.Data)
	}
}

func TestTurnstileRejectsMissingCookie(t *testing.T) {
	sec := newTestTurnstile(time.Hour)
	e := echo.New()
	e.GET("/api/protected", okHandler, sec.Turnstile())

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/protected", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestVerifyTurnstileSetsCookie(t *testing.T) {
	var verifyCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		verifyCalled = true
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("secret") != "secret-key" || r.Form.Get("response") != "valid-token" {
			t.Fatalf("unexpected verify form: %v", r.Form)
		}
		w.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer ts.Close()

	sec := newTestTurnstile(time.Hour)
	sec.turnstile.verifyURL = ts.URL
	e := echo.New()
	e.POST("/api/turnstile/verify", sec.VerifyTurnstile)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/turnstile/verify",
		strings.NewReader(`{"token":"valid-token"}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !verifyCalled {
		t.Fatal("verify endpoint was not called")
	}
	cookie := findCookie(rec.Result().Cookies(), turnstileCookieName)
	if cookie == nil {
		t.Fatalf("%s cookie was not set", turnstileCookieName)
	}

	protectedReq := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	protectedReq.AddCookie(cookie)
	if !sec.turnstileVerified(protectedReq) {
		t.Fatal("expected verification cookie to be valid")
	}
}

func TestTurnstileRejectsInvalidToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		_, _ = w.Write([]byte(`{"success":false}`))
	}))
	defer ts.Close()

	sec := newTestTurnstile(time.Hour)
	sec.turnstile.verifyURL = ts.URL
	e := echo.New()
	e.POST("/api/turnstile/verify", sec.VerifyTurnstile)

	req := httptest.NewRequest(http.MethodPost, "/api/turnstile/verify", strings.NewReader(`{"token":"invalid"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestTurnstileRejectsOversizedTokenWithoutCallingUpstream(t *testing.T) {
	var verifyCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		verifyCalled = true
		w.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer ts.Close()

	sec := newTestTurnstile(time.Hour)
	sec.turnstile.verifyURL = ts.URL
	e := echo.New()
	e.POST("/api/turnstile/verify", sec.VerifyTurnstile)

	body := `{"token":"` + strings.Repeat("a", maxTurnstileTokenBytes+1) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/turnstile/verify", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if verifyCalled {
		t.Fatal("verify endpoint should not be called for an oversized token")
	}
}

func TestTurnstileCookieValidation(t *testing.T) {
	sec := newTestTurnstile(time.Hour)

	t.Run("valid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
		req.AddCookie(sec.newTurnstileCookie(time.Now()))
		if !sec.turnstileVerified(req) {
			t.Fatal("expected cookie to be valid")
		}
	})

	t.Run("tampered", func(t *testing.T) {
		cookie := sec.newTurnstileCookie(time.Now())
		cookie.Value += "tampered"
		req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
		req.AddCookie(cookie)
		if sec.turnstileVerified(req) {
			t.Fatal("expected tampered cookie to be rejected")
		}
	})

	t.Run("expired", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
		req.AddCookie(sec.newTurnstileCookie(time.Now().Add(-2 * time.Hour)))
		if sec.turnstileVerified(req) {
			t.Fatal("expected expired cookie to be rejected")
		}
	})
}

func TestNewTurnstileCookie(t *testing.T) {
	sec := newTestTurnstile(6 * time.Hour)
	sec.secureCookies = true
	now := time.Unix(1_700_000_000, 0)
	cookie := sec.newTurnstileCookie(now)

	if cookie.MaxAge != 6*60*60 {
		t.Fatalf("max age = %d, want %d", cookie.MaxAge, 6*60*60)
	}
	if !cookie.Expires.Equal(now.Add(6 * time.Hour)) {
		t.Fatalf("expires = %s, want %s", cookie.Expires, now.Add(6*time.Hour))
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected cookie flags: %+v", cookie)
	}
}

func newTestTurnstile(ttl time.Duration) *Middleware {
	return New(config.Security{
		TurnstileSiteKey:   "site-key",
		TurnstileSecretKey: "secret-key",
		TurnstileCookieTTL: ttl,
	})
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
