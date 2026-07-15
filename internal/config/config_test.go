package config

import "testing"

func TestMustNewTurnstileDisabled(t *testing.T) {
	setTurnstileEnv(t, "", "", "6h")

	if MustNew().Security.TurnstileEnabled() {
		t.Fatal("expected Turnstile to be disabled")
	}
}

func TestMustNewTurnstileEnabled(t *testing.T) {
	setTurnstileEnv(t, "site-key", "secret-key", "1h")

	if !MustNew().Security.TurnstileEnabled() {
		t.Fatal("expected Turnstile to be enabled")
	}
}

func TestMustNewRejectsPartialTurnstileConfig(t *testing.T) {
	tests := []struct {
		name      string
		siteKey   string
		secretKey string
	}{
		{name: "missing secret key", siteKey: "site-key"},
		{name: "missing site key", secretKey: "secret-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTurnstileEnv(t, tt.siteKey, tt.secretKey, "6h")
			assertPanics(t, MustNew)
		})
	}
}

func TestMustNewRejectsInvalidTurnstileTTL(t *testing.T) {
	setTurnstileEnv(t, "site-key", "secret-key", "0s")
	assertPanics(t, MustNew)
}

func setTurnstileEnv(t *testing.T, siteKey, secretKey, ttl string) {
	t.Helper()
	t.Setenv("SECURITY_TURNSTILE_SITE_KEY", siteKey)
	t.Setenv("SECURITY_TURNSTILE_SECRET_KEY", secretKey)
	t.Setenv("SECURITY_TURNSTILE_COOKIE_TTL", ttl)
}

func assertPanics(t *testing.T, fn func() *Config) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = fn()
}
