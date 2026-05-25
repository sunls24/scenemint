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
	g.POST("/quota/status", server.Wrap(quotaStore.Status), security.BodyLimit(), sec.CSRF())
	g.POST("/quota/check-in", server.Wrap(quotaStore.CheckIn), security.BodyLimit(), sec.CSRF())
	g.POST("/prompts/enhance", imageClient.EnhancePrompt, security.BodyLimit(), sec.CSRF())
	g.POST("/images/generate", server.WrapReplyResp(imageClient.GenerateReply), security.BodyLimit(), sec.CSRF())
	g.GET("/images/tasks/:id", server.WrapResp(imageClient.Task))
	g.GET("/images/tasks/:id/image", imageClient.ProxyImage)
}
