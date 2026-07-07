package config

import (
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
	TurnstileEnabled   bool          `env:"TURNSTILE_ENABLED" envDefault:"false"`
	TurnstileSiteKey   string        `env:"TURNSTILE_SITE_KEY"`
	TurnstileSecretKey string        `env:"TURNSTILE_SECRET_KEY"`
	TurnstileHumanTTL  time.Duration `env:"TURNSTILE_HUMAN_TTL"`
}

func MustNew() *Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}
	return &cfg
}
