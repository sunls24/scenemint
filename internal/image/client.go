package image

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"scenemint/internal/config"
	"scenemint/internal/quota"

	"github.com/labstack/echo/v5"
	"github.com/sunls24/gox/network/client"
	"github.com/sunls24/gox/server"
)

type Client struct {
	openAIBaseURL string
	taskAPIRoot   string
	apiKey        string
	model         string
	promptModel   string
	quota         *quota.Store
	http          *client.Client
	rawHTTP       *http.Client
}

type GenerateRequest struct {
	Prompt         string `json:"prompt"`
	Size           string `json:"size"`
	ReferenceImage string `json:"referenceImage"`
	Fingerprint    string `json:"fingerprint"`
}

type GenerateResponse struct {
	ID               string `json:"id"`
	Mode             string `json:"mode"`
	Prompt           string `json:"prompt,omitempty"`
	Size             string `json:"size,omitempty"`
	Image            string `json:"image,omitempty"`
	Status           string `json:"status"`
	Error            string `json:"error,omitempty"`
	RevisedPrompt    string `json:"revisedPrompt,omitempty"`
	CreatedAt        string `json:"createdAt,omitempty"`
	UpdatedAt        string `json:"updatedAt,omitempty"`
	RemainingCredits *int   `json:"remainingCredits,omitempty"`
}

func NewClient(cfg *config.Config, quotaStore *quota.Store) *Client {
	baseURL := strings.TrimSpace(cfg.ChatGPT2API.BaseURL)
	return &Client{
		openAIBaseURL: normalizeOpenAIBaseURL(baseURL),
		taskAPIRoot:   normalizeTaskAPIRoot(baseURL),
		apiKey:        strings.TrimSpace(cfg.ChatGPT2API.APIKey),
		model:         strings.TrimSpace(cfg.ChatGPT2API.ImageModel),
		promptModel:   strings.TrimSpace(cfg.ChatGPT2API.PromptModel),
		quota:         quotaStore,
		http:          client.New(),
		rawHTTP:       &http.Client{Timeout: 10 * time.Minute},
	}
}

func (c *Client) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	if err := c.validateConfig(); err != nil {
		return GenerateResponse{}, err
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return GenerateResponse{}, server.ErrMsg("请输入图片提示词")
	}
	if c.quota == nil {
		return GenerateResponse{}, server.ErrMsg("额度服务未初始化")
	}

	spend, remaining, err := c.quota.Spend(req.Fingerprint)
	if err != nil {
		return GenerateResponse{}, quotaError(err)
	}

	normalized := normalize(req)
	id, err := taskID()
	if err != nil {
		_ = spend.Refund()
		return GenerateResponse{}, server.ErrMsg("图片任务创建失败").WithErr(err)
	}

	mode := "text"
	var task upstreamTask
	if strings.TrimSpace(req.ReferenceImage) == "" {
		task, err = c.submitGenerationTask(ctx, taskSubmitRequest{
			ClientTaskID: id,
			Prompt:       prompt,
			Model:        c.model,
			Size:         normalized.Size,
		})
	} else {
		mode = "image"
		task, err = c.submitEditTask(ctx, taskSubmitRequest{
			ClientTaskID: id,
			Prompt:       prompt,
			Model:        c.model,
			Size:         normalized.Size,
			Image:        req.ReferenceImage,
		})
	}
	if err != nil {
		_ = spend.Refund()
		return GenerateResponse{}, server.ErrMsg("图片任务提交失败").WithErr(err)
	}
	task.ID = pickString(task.ID, id)
	task.Mode = pickString(task.Mode, mode)
	task.Size = pickString(task.Size, normalized.Size)

	resp := c.taskResponse(task)
	resp.Prompt = prompt
	resp.CreatedAt = pickString(resp.CreatedAt, time.Now().Format(time.RFC3339))
	resp.RemainingCredits = intPtr(remaining.Balance)
	return resp, nil
}

func (c *Client) Task(ctx context.Context) (GenerateResponse, error) {
	if err := c.validateConfig(); err != nil {
		return GenerateResponse{}, err
	}
	id := strings.TrimSpace(server.EchoContext(ctx).Param("id"))
	if id == "" {
		return GenerateResponse{}, server.ErrMsg("任务 ID 不能为空")
	}
	task, missing, err := c.fetchTask(ctx, id)
	if err != nil {
		return GenerateResponse{}, server.ErrMsg("图片任务查询失败").WithErr(err)
	}
	if missing {
		return GenerateResponse{
			ID:     id,
			Mode:   "text",
			Status: "failed",
			Error:  "任务不存在或已过期",
		}, nil
	}
	return c.taskResponse(task), nil
}

func (c *Client) ProxyImage(ec *echo.Context) error {
	if msg := c.configErrorMessage(); msg != "" {
		return imageProxyError(ec, http.StatusBadGateway, msg)
	}
	id := strings.TrimSpace(ec.Param("id"))
	if id == "" {
		return imageProxyError(ec, http.StatusBadRequest, "任务 ID 不能为空")
	}

	task, missing, err := c.fetchTask(ec.Request().Context(), id)
	if err != nil {
		return imageProxyError(ec, http.StatusBadGateway, "图片任务查询失败")
	}
	if missing || mapStatus(task.Status) != "completed" {
		return imageProxyError(ec, http.StatusNotFound, "图片尚未生成或已过期")
	}

	imageURL := firstImageURL(task)
	if !validImageURL(imageURL) {
		return imageProxyError(ec, http.StatusNotFound, "图片地址不可用")
	}

	req, err := http.NewRequestWithContext(ec.Request().Context(), http.MethodGet, imageURL, nil)
	if err != nil {
		return imageProxyError(ec, http.StatusBadGateway, "图片请求创建失败")
	}
	resp, err := c.rawHTTP.Do(req)
	if err != nil {
		return imageProxyError(ec, http.StatusBadGateway, "图片读取失败")
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return imageProxyError(ec, http.StatusBadGateway, "上游图片读取失败")
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		ec.Response().Header().Set("Content-Length", contentLength)
	}
	ec.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	return ec.Stream(http.StatusOK, contentType, resp.Body)
}

func (c *Client) validateConfig() error {
	if msg := c.configErrorMessage(); msg != "" {
		return server.ErrMsg(msg)
	}
	return nil
}

func (c *Client) configErrorMessage() string {
	if c.openAIBaseURL == "" || c.taskAPIRoot == "" {
		return "CHATGPT2API_BASE_URL 未配置"
	}
	if c.apiKey == "" {
		return "CHATGPT2API_API_KEY 未配置"
	}
	return ""
}

func normalizeOpenAIBaseURL(baseURL string) string {
	root := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if root == "" || strings.HasSuffix(root, "/v1") {
		return root
	}
	return root + "/v1"
}

func normalizeTaskAPIRoot(baseURL string) string {
	root := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return strings.TrimSuffix(root, "/v1")
}

func taskID() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	data[6] = (data[6] & 0x0f) | 0x40
	data[8] = (data[8] & 0x3f) | 0x80
	dst := make([]byte, 36)
	hex.Encode(dst[0:8], data[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], data[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], data[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], data[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], data[10:16])
	return string(dst), nil
}

func pickString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func quotaError(err error) error {
	if respErr, ok := quota.InvalidFingerprintResponseError(err); ok {
		return respErr
	}
	if errors.Is(err, quota.ErrNoCredits) {
		return server.ErrMsg(fmt.Sprintf("额度不足，请先签到领取 %d 张额度", quota.DailyGrant))
	}
	return server.ErrMsg("额度查询失败").WithErr(err)
}

func intPtr(value int) *int {
	return &value
}

func validImageURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func imageProxyError(ec *echo.Context, status int, message string) error {
	return ec.JSON(status, server.ErrMsg(message).Envelope())
}
