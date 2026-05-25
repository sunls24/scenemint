package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/sunls24/gox/network/header"
)

type taskSubmitRequest struct {
	ClientTaskID string `json:"client_task_id"`
	Prompt       string `json:"prompt"`
	Model        string `json:"model"`
	Size         string `json:"size,omitempty"`
	Image        string `json:"image,omitempty"`
}

type upstreamTaskList struct {
	Items      []upstreamTask `json:"items"`
	MissingIDs []string       `json:"missing_ids"`
}

type upstreamTask struct {
	ID        string              `json:"id"`
	Status    string              `json:"status"`
	Mode      string              `json:"mode"`
	Model     string              `json:"model"`
	Size      string              `json:"size"`
	Data      []upstreamTaskImage `json:"data"`
	Error     string              `json:"error"`
	CreatedAt string              `json:"created_at"`
	UpdatedAt string              `json:"updated_at"`
}

type upstreamTaskImage struct {
	URL           string `json:"url"`
	RevisedPrompt string `json:"revised_prompt"`
}

func (c *Client) submitGenerationTask(ctx context.Context, body taskSubmitRequest) (upstreamTask, error) {
	data, err := c.http.Post(
		ctx,
		c.taskAPIRoot+"/api/image-tasks/generations",
		body,
		header.New().ContentTypeJSON().Authorization(c.apiKey).Get()...,
	)
	if err != nil {
		return upstreamTask{}, err
	}
	return decodeTask(data)
}

func (c *Client) submitEditTask(ctx context.Context, body taskSubmitRequest) (upstreamTask, error) {
	imageData, imageType, err := decodeDataURLImage(body.Image)
	if err != nil {
		return upstreamTask{}, err
	}

	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	fields := map[string]string{
		"client_task_id": body.ClientTaskID,
		"prompt":         body.Prompt,
		"model":          body.Model,
		"size":           body.Size,
	}
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if err := writer.WriteField(name, value); err != nil {
			_ = writer.Close()
			return upstreamTask{}, err
		}
	}

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     "image",
		"filename": "reference." + imageExtension(imageType),
	}))
	partHeader.Set("Content-Type", imageType)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		_ = writer.Close()
		return upstreamTask{}, err
	}
	if _, err := part.Write(imageData); err != nil {
		_ = writer.Close()
		return upstreamTask{}, err
	}
	if err := writer.Close(); err != nil {
		return upstreamTask{}, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.taskAPIRoot+"/api/image-tasks/edits",
		&form,
	)
	if err != nil {
		return upstreamTask{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.rawHTTP.Do(req)
	if err != nil {
		return upstreamTask{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return upstreamTask{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if len(data) == 0 {
			return upstreamTask{}, fmt.Errorf("%s", resp.Status)
		}
		return upstreamTask{}, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return decodeTask(data)
}

func decodeDataURLImage(value string) ([]byte, string, error) {
	header, payload, ok := strings.Cut(strings.TrimSpace(value), ",")
	if !ok || !strings.HasPrefix(strings.ToLower(header), "data:") {
		return nil, "", fmt.Errorf("参考图格式不正确")
	}
	imageType := strings.TrimSpace(strings.TrimPrefix(strings.Split(header, ";")[0], "data:"))
	if !strings.HasPrefix(strings.ToLower(imageType), "image/") {
		return nil, "", fmt.Errorf("参考图必须是图片")
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", fmt.Errorf("参考图格式不正确: %w", err)
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("参考图不能为空")
	}
	return data, imageType, nil
}

func imageExtension(imageType string) string {
	switch strings.ToLower(imageType) {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/webp":
		return "webp"
	default:
		return "png"
	}
}

func decodeTask(data []byte) (upstreamTask, error) {
	var task upstreamTask
	if err := json.Unmarshal(data, &task); err != nil {
		return upstreamTask{}, err
	}
	return task, nil
}

func (c *Client) fetchTask(ctx context.Context, id string) (upstreamTask, bool, error) {
	data, err := c.http.Get(
		ctx,
		c.taskAPIRoot+"/api/image-tasks?ids="+url.QueryEscape(id),
		header.New().Authorization(c.apiKey).Get()...,
	)
	if err != nil {
		return upstreamTask{}, false, err
	}

	var resp upstreamTaskList
	if err := json.Unmarshal(data, &resp); err != nil {
		return upstreamTask{}, false, err
	}
	for _, task := range resp.Items {
		if task.ID == id {
			return task, false, nil
		}
	}
	for _, missing := range resp.MissingIDs {
		if missing == id {
			return upstreamTask{}, true, nil
		}
	}
	return upstreamTask{}, true, nil
}

func (c *Client) taskResponse(task upstreamTask) GenerateResponse {
	status := mapStatus(task.Status)
	image := ""
	if status == "completed" {
		image = "/api/images/tasks/" + task.ID + "/image"
	}
	first := firstImage(task)
	mode := "text"
	if task.Mode == "edit" || task.Mode == "image" {
		mode = "image"
	}
	return GenerateResponse{
		ID:            task.ID,
		Mode:          mode,
		Size:          task.Size,
		Image:         image,
		Status:        status,
		Error:         task.Error,
		RevisedPrompt: first.RevisedPrompt,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
	}
}

func mapStatus(status string) string {
	switch status {
	case "queued", "running":
		return status
	case "success", "completed":
		return "completed"
	case "error", "failed":
		return "failed"
	default:
		return "queued"
	}
}

func firstImage(task upstreamTask) upstreamTaskImage {
	if len(task.Data) == 0 {
		return upstreamTaskImage{}
	}
	return task.Data[0]
}

func firstImageURL(task upstreamTask) string {
	return strings.TrimSpace(firstImage(task).URL)
}
