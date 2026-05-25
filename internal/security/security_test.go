package security

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func okHandler(c *echo.Context) error {
	return c.JSON(http.StatusOK, server.Envelope{Message: "ok"})
}
