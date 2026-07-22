package internal

import (
	"fmt"
	"log/slog"
	"os"
	"scenemint/internal/config"
	"scenemint/internal/image"
	"scenemint/internal/poll"
	"scenemint/internal/quota"
	"scenemint/internal/router"
	"scenemint/internal/security"

	"github.com/rs/zerolog"
	slogzerolog "github.com/samber/slog-zerolog/v2"
	"github.com/sunls24/gox/server"
)

type App struct{}

func NewApp() App {
	return App{}
}

func setupLogger() *slog.Logger {
	logger := slog.New(slogzerolog.Option{Level: slog.LevelInfo, Logger: new(zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "06-01-02 15:04:05"}))}.NewZerologHandler())
	slog.SetDefault(logger)
	return logger
}

func (app App) Run() error {
	logger := setupLogger()
	cfg := config.MustNew()
	quotaStore, err := quota.Open(quota.DefaultPath)
	if err != nil {
		return err
	}
	defer quotaStore.Close()
	pollStore, err := poll.Open(poll.DefaultPath)
	if err != nil {
		return err
	}
	defer pollStore.Close()

	imageClient := image.NewClient(cfg, quotaStore)

	return server.Start(fmt.Sprintf("%s:%s", cfg.Host, cfg.Port), func(srv *server.Server) {
		srv.Echo.Logger = logger
		srv.Echo.Use(security.Headers())
		srv.Echo.Static("/", "web/dist")
		router.Register(srv.Echo, imageClient, quotaStore, pollStore, cfg.Security)
	})
}
