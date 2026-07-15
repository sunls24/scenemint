package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Host        string      `env:"HOST" envDefault:"127.0.0.1"`
	Port        string      `env:"PORT" envDefault:"3000"`
	ChatGPT2API ChatGPT2API `envPrefix:"CHATGPT2API_"`
	Security    Security    `envPrefix:"SECURITY_"`
}

type ChatGPT2API struct {
	BaseURL     string `env:"BASE_URL"`
	APIKey      string `env:"API_KEY"`
	ImageModel  string `env:"IMAGE_MODEL" envDefault:"gpt-image-2"`
	PromptModel string `env:"PROMPT_MODEL" envDefault:"gpt-5.5"`
}

type Security struct {
	SecureCookies      bool          `env:"SECURE_COOKIES" envDefault:"false"`
	TurnstileSiteKey   string        `env:"TURNSTILE_SITE_KEY"`
	TurnstileSecretKey string        `env:"TURNSTILE_SECRET_KEY"`
	TurnstileCookieTTL time.Duration `env:"TURNSTILE_COOKIE_TTL" envDefault:"6h"`
}

func MustNew() *Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}
	cfg.Security.TurnstileSiteKey = strings.TrimSpace(cfg.Security.TurnstileSiteKey)
	cfg.Security.TurnstileSecretKey = strings.TrimSpace(cfg.Security.TurnstileSecretKey)
	if (cfg.Security.TurnstileSiteKey == "") != (cfg.Security.TurnstileSecretKey == "") {
		panic("SECURITY_TURNSTILE_SITE_KEY and SECURITY_TURNSTILE_SECRET_KEY must be set together")
	}
	if cfg.Security.TurnstileEnabled() && cfg.Security.TurnstileCookieTTL <= 0 {
		panic(fmt.Sprintf("invalid SECURITY_TURNSTILE_COOKIE_TTL: %s", cfg.Security.TurnstileCookieTTL))
	}
	return &cfg
}

func (c Security) TurnstileEnabled() bool {
	return c.TurnstileSiteKey != "" && c.TurnstileSecretKey != ""
}
