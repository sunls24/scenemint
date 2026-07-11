package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/sunls24/gox/openai"
	"github.com/sunls24/gox/server"
)

const (
	enhanceDirectionDetails  = "details"
	enhanceDirectionCreative = "creative"
)

type enhancePromptRequest struct {
	Prompt    string `json:"prompt"`
	Direction string `json:"direction"`
}

type flushWriter struct {
	writer  io.Writer
	flusher http.Flusher
}

func (w flushWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if n > 0 {
		w.flusher.Flush()
	}
	return n, err
}

func (c *Client) EnhancePrompt(ec *echo.Context) error {
	var req enhancePromptRequest
	if err := ec.Bind(&req); err != nil {
		return promptError(ec, http.StatusBadRequest, "请求格式不正确")
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.Direction = strings.TrimSpace(req.Direction)
	if req.Prompt == "" {
		return promptError(ec, http.StatusBadRequest, "请输入需要增强的提示词")
	}
	if !validEnhanceDirection(req.Direction) {
		return promptError(ec, http.StatusBadRequest, "提示词增强方向不支持")
	}
	if msg := c.promptConfigErrorMessage(); msg != "" {
		slog.Error("提示词服务配置错误", "detail", msg)
		return promptError(ec, http.StatusBadGateway, "提示词服务暂不可用，请稍后再试")
	}

	reader, err := c.submitPromptEnhancement(ec.Request().Context(), req)
	if err != nil {
		return promptError(ec, http.StatusBadGateway, "提示词增强失败")
	}
	defer reader.Close()

	resp := ec.Response()
	flusher, ok := resp.(http.Flusher)
	if !ok {
		return promptError(ec, http.StatusInternalServerError, "当前连接不支持流式响应")
	}

	headers := resp.Header()
	headers.Set(echo.HeaderContentType, "text/event-stream")
	headers.Set(echo.HeaderCacheControl, "no-cache")
	headers.Set("X-Accel-Buffering", "no")
	resp.WriteHeader(http.StatusOK)

	// 这里保持上游 SSE 原样透传，前端负责解析 OpenAI chunk。
	_, err = io.CopyBuffer(
		flushWriter{writer: resp, flusher: flusher},
		reader,
		make([]byte, 1024),
	)
	if errors.Is(err, context.Canceled) || errors.Is(ec.Request().Context().Err(), context.Canceled) {
		return nil
	}
	return err
}

func (c *Client) submitPromptEnhancement(
	ctx context.Context,
	req enhancePromptRequest,
) (io.ReadCloser, error) {
	stream := true
	temperature := enhanceTemperature(req.Direction)
	return c.openAIClient().ChatCompletions(ctx, openai.ChatRequest{
		Model: c.promptModel,
		Messages: []*openai.Message{
			openai.SystemMessage(enhanceSystemPrompt(req.Direction)),
			openai.UserMessage(fmt.Sprintf("原始提示词：\n%s", req.Prompt)),
		},
		Temperature: &temperature,
		Stream:      &stream,
	})
}

func (c *Client) openAIClient() *openai.OpenAI {
	return openai.New(c.openAIBaseURL, c.apiKey, openai.WithClient(c.rawHTTP))
}

func (c *Client) promptConfigErrorMessage() string {
	if msg := c.configErrorMessage(); msg != "" {
		return msg
	}
	if c.promptModel == "" {
		return "CHATGPT2API_PROMPT_MODEL 未配置"
	}
	return ""
}

func validEnhanceDirection(direction string) bool {
	return direction == enhanceDirectionDetails || direction == enhanceDirectionCreative
}

func enhanceTemperature(direction string) float64 {
	if direction == enhanceDirectionCreative {
		return 0.7
	}
	return 0.35
}

func enhanceSystemPrompt(direction string) string {
	base := strings.Join([]string{
		"你是 SceneMint 的提示词增强器。",
		"把用户输入改写成一段可直接用于 AI 生图模型的高质量提示词。",
		"只输出增强后的提示词，不要解释、标题、Markdown 或列点。",
		"保留用户的核心主体、数量、关系、场景、风格、媒介、画面类型、",
		"限制条件和明确要求。",
		"输出语言默认跟随原始输入。",
		"根据原始提示词的信息密度自适应增强：短输入可补全关键视觉信息；",
		"完整输入只做轻量润色、去重、整理和少量补强。",
		"避免重复、空泛形容词、无关细节，",
		"以及 high quality、masterpiece、best quality 这类空泛质量词。",
		"不要输出尺寸、比例或横竖幅；若原文包含这类信息，省略它们。",
	}, " ")

	if direction == enhanceDirectionCreative {
		return base + " " + strings.Join([]string{
			"在不改变核心意图的前提下，",
			"只选择 1-2 个创意变量强化，例如叙事感、空间层次、特殊视角、",
			"氛围反差或想象力元素；",
			"不要跑题、替换主要对象或堆叠无关元素。",
		}, " ")
	}
	return base + " " + strings.Join([]string{
		"补足缺失的主体细节、环境、构图、光线、材质、色彩和氛围，",
		"不新增与原意无关的叙事或设定。",
	}, " ")
}

func promptError(ec *echo.Context, status int, message string) error {
	return ec.JSON(status, server.ErrMsg(message).Envelope())
}
