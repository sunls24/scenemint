package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"scenemint/internal/config"

	"github.com/labstack/echo/v5"
)

func TestRegisterTurnstileRoutes(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		e := echo.New()
		Register(e, nil, nil, config.Security{})

		assertRoute(t, e, http.MethodGet, "/api/turnstile/status", true)
		assertRoute(t, e, http.MethodPost, "/api/turnstile/verify", false)
	})

	t.Run("enabled", func(t *testing.T) {
		e := echo.New()
		Register(e, nil, nil, turnstileConfig())

		assertRoute(t, e, http.MethodGet, "/api/turnstile/status", true)
		assertRoute(t, e, http.MethodPost, "/api/turnstile/verify", true)
	})
}

func TestTurnstileProtectsBusinessRoutes(t *testing.T) {
	e := echo.New()
	Register(e, nil, nil, turnstileConfig())

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/images/tasks/test", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestTurnstileVerifyRejectsOversizedBody(t *testing.T) {
	e := echo.New()
	Register(e, nil, nil, turnstileConfig())

	req := httptest.NewRequest(
		http.MethodPost,
		"http://example.com/api/turnstile/verify",
		strings.NewReader(`{"token":"`+strings.Repeat("a", 4<<10)+`"}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderOrigin, "http://example.com")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
}

func turnstileConfig() config.Security {
	return config.Security{
		TurnstileSiteKey:   "site-key",
		TurnstileSecretKey: "secret-key",
		TurnstileCookieTTL: time.Hour,
	}
}

func assertRoute(t *testing.T, e *echo.Echo, method, path string, exists bool) {
	t.Helper()
	_, err := e.Router().Routes().FindByMethodPath(method, path)
	if (err == nil) != exists {
		t.Fatalf("route %s %s existence = %t, want %t", method, path, err == nil, exists)
	}
}
