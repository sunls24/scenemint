package router

import (
	"scenemint/internal/config"
	"scenemint/internal/handler"
	"scenemint/internal/image"
	"scenemint/internal/quota"
	"scenemint/internal/security"

	"github.com/labstack/echo/v5"
	"github.com/sunls24/gox/server"
)

func Register(e *echo.Echo, imageClient *image.Client, quotaStore *quota.Store, cfg config.Security) {
	sec := security.New(cfg)
	g := e.Group("/api")
	g.Use(sec.SourceGuard())
	g.GET("/session", sec.Session)
	g.GET("/status", server.WrapResp(handler.Status))
	g.GET("/turnstile/status", sec.TurnstileStatus)

	protected := g
	if cfg.TurnstileEnabled() {
		g.POST("/turnstile/verify", sec.VerifyTurnstile, security.TurnstileBodyLimit())
		protected = g.Group("", sec.Turnstile())
	}
	protected.POST("/quota/status", server.Wrap(quotaStore.Status), security.BodyLimit(), sec.CSRF())
	protected.POST("/quota/check-in", server.Wrap(quotaStore.CheckIn), security.BodyLimit(), sec.CSRF())
	protected.POST("/prompts/enhance", imageClient.EnhancePrompt, security.BodyLimit(), sec.CSRF())
	protected.POST("/images/generate", server.WrapReplyResp(imageClient.GenerateReply), security.BodyLimit(), sec.CSRF())
	protected.GET("/images/tasks/:id", server.WrapResp(imageClient.Task))
	protected.GET("/images/tasks/:id/image", imageClient.ProxyImage)
}
