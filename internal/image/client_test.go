package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"scenemint/internal/quota"

	"github.com/labstack/echo/v5"
	"github.com/sunls24/gox/network/client"
	"github.com/sunls24/gox/openai"
)

func TestSubmitGenerationTaskUsesJSON(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/image-tasks/generations" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}

		var body taskSubmitRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Decode request body: %v", err)
		}
		if body.ClientTaskID != "task-1" ||
			body.Prompt != "quiet studio scene" ||
			body.Model != "gpt-image-2" ||
			body.Size != "1:1" ||
			body.Image != "" {
			t.Fatalf("unexpected request body: %+v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(upstreamTask{
			ID:     "task-1",
			Status: "queued",
			Mode:   "generation",
			Model:  "gpt-image-2",
			Size:   "1:1",
		}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}))
	defer ts.Close()

	c := &Client{
		taskAPIRoot: ts.URL,
		apiKey:      "test-key",
		http:        client.New(),
	}

	task, err := c.submitGenerationTask(context.Background(), taskSubmitRequest{
		ClientTaskID: "task-1",
		Prompt:       "quiet studio scene",
		Model:        "gpt-image-2",
		Size:         "1:1",
	})
	if err != nil {
		t.Fatalf("submitGenerationTask returned error: %v", err)
	}
	if task.ID != "task-1" || task.Status != "queued" || task.Mode != "generation" {
		t.Fatalf("unexpected task: %+v", task)
	}
}

func TestSubmitEditTaskUsesMultipartFile(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/image-tasks/edits" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data; boundary=") {
			t.Fatalf("Content-Type = %q, want multipart/form-data", got)
		}
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}

		wantFields := map[string]string{
			"client_task_id": "task-1",
			"prompt":         "replace the background",
			"model":          "gpt-image-2",
			"size":           "1:1",
		}
		for name, want := range wantFields {
			if got := r.FormValue(name); got != want {
				t.Fatalf("%s = %q, want %q", name, got, want)
			}
		}
		files := r.MultipartForm.File["image"]
		if len(files) != 1 {
			t.Fatalf("image files = %d, want 1", len(files))
		}
		if got := files[0].Filename; got != "reference.png" {
			t.Fatalf("image filename = %q, want reference.png", got)
		}
		if got := files[0].Header.Get("Content-Type"); got != "image/png" {
			t.Fatalf("image Content-Type = %q, want image/png", got)
		}
		file, err := files[0].Open()
		if err != nil {
			t.Fatalf("Open image file: %v", err)
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("Read image file: %v", err)
		}
		if string(data) != "\x89PNG\r\n\x1a\n" {
			t.Fatalf("image bytes = %q, want PNG header", string(data))
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(upstreamTask{
			ID:     "task-1",
			Status: "queued",
			Mode:   "edit",
			Model:  "gpt-image-2",
			Size:   "1:1",
		}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}))
	defer ts.Close()

	c := &Client{
		taskAPIRoot: ts.URL,
		apiKey:      "test-key",
		rawHTTP:     ts.Client(),
	}

	task, err := c.submitEditTask(context.Background(), taskSubmitRequest{
		ClientTaskID: "task-1",
		Prompt:       "replace the background",
		Model:        "gpt-image-2",
		Size:         "1:1",
		Image:        "data:image/png;base64,iVBORw0KGgo=",
	})
	if err != nil {
		t.Fatalf("submitEditTask returned error: %v", err)
	}
	if task.ID != "task-1" || task.Status != "queued" || task.Mode != "edit" {
		t.Fatalf("unexpected task: %+v", task)
	}
}

func TestSubmitEditTaskSendsLargeReferenceImageAsFile(t *testing.T) {
	t.Parallel()

	largeImage := bytes.Repeat([]byte{0xab}, (1024*1024)+1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data; boundary=") {
			t.Fatalf("Content-Type = %q, want multipart/form-data", got)
		}
		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("MultipartReader: %v", err)
		}

		var foundImage bool
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextPart: %v", err)
			}
			if part.FormName() != "image" {
				_, _ = io.Copy(io.Discard, part)
				continue
			}
			foundImage = true
			if part.FileName() == "" {
				t.Fatal("image part was sent as a form field, want file part")
			}
			if got := part.Header.Get("Content-Type"); got != "image/png" {
				t.Fatalf("image Content-Type = %q, want image/png", got)
			}
			got, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("Read image part: %v", err)
			}
			if !bytes.Equal(got, largeImage) {
				t.Fatalf("image payload length = %d, want %d", len(got), len(largeImage))
			}
		}
		if !foundImage {
			t.Fatal("image part was not sent")
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(upstreamTask{
			ID:     "task-1",
			Status: "queued",
			Mode:   "edit",
			Model:  "gpt-image-2",
			Size:   "1:1",
		}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}))
	defer ts.Close()

	c := &Client{
		taskAPIRoot: ts.URL,
		apiKey:      "test-key",
		rawHTTP:     ts.Client(),
	}

	_, err := c.submitEditTask(context.Background(), taskSubmitRequest{
		ClientTaskID: "task-1",
		Prompt:       "replace the background",
		Model:        "gpt-image-2",
		Size:         "1:1",
		Image:        "data:image/png;base64," + base64.StdEncoding.EncodeToString(largeImage),
	})
	if err != nil {
		t.Fatalf("submitEditTask returned error: %v", err)
	}
}

func TestSubmitEditTaskReturnsUpstreamErrorBody(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		http.Error(w, `{"detail":"bad request"}`, http.StatusUnprocessableEntity)
	}))
	defer ts.Close()

	c := &Client{
		taskAPIRoot: ts.URL,
		apiKey:      "test-key",
		rawHTTP:     ts.Client(),
	}

	_, err := c.submitEditTask(context.Background(), taskSubmitRequest{
		ClientTaskID: "task-1",
		Prompt:       "replace the background",
		Image:        "data:image/png;base64,iVBORw0KGgo=",
	})
	if err == nil {
		t.Fatal("submitEditTask returned nil error")
	}
	if got := err.Error(); !strings.Contains(got, "422 Unprocessable Entity") || !strings.Contains(got, "bad request") {
		t.Fatalf("error = %q, want status and body", got)
	}
}

func TestGenerateSpendsCreditAfterSuccessfulSubmit(t *testing.T) {
	t.Parallel()

	store := newQuotaStore(t)
	if _, err := store.ApplyCheckIn("fingerprint123"); err != nil {
		t.Fatalf("ApplyCheckIn returned error: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/image-tasks/generations" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Decode request body: %v", err)
		}
		if got := body["prompt"]; got != "quiet studio scene" {
			t.Fatalf("prompt = %q, want original prompt", got)
		}
		if _, ok := body["style"]; ok {
			t.Fatalf("request body should not include style: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(upstreamTask{
			ID:     "task-1",
			Status: "queued",
			Mode:   "generation",
			Model:  "gpt-image-2",
			Size:   "1:1",
		}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}))
	defer ts.Close()

	c := &Client{
		openAIBaseURL: ts.URL + "/v1",
		taskAPIRoot:   ts.URL,
		apiKey:        "test-key",
		model:         "gpt-image-2",
		quota:         store,
		http:          client.New(),
	}

	resp, err := c.Generate(context.Background(), GenerateRequest{
		Prompt:      "quiet studio scene",
		Size:        "1:1",
		Fingerprint: "fingerprint123",
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.RemainingCredits == nil || *resp.RemainingCredits != quota.DailyGrant-1 {
		t.Fatalf("RemainingCredits = %v, want %d", resp.RemainingCredits, quota.DailyGrant-1)
	}

	status, err := store.Get("fingerprint123")
	if err != nil {
		t.Fatalf("quota Get returned error: %v", err)
	}
	if status.Balance != quota.DailyGrant-1 {
		t.Fatalf("quota balance = %d, want %d", status.Balance, quota.DailyGrant-1)
	}
}

func TestGenerateRejectsZeroQuota(t *testing.T) {
	t.Parallel()

	var called atomic.Bool
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called.Store(true)
	}))
	defer ts.Close()

	c := &Client{
		openAIBaseURL: ts.URL + "/v1",
		taskAPIRoot:   ts.URL,
		apiKey:        "test-key",
		model:         "gpt-image-2",
		quota:         newQuotaStore(t),
		http:          client.New(),
	}

	_, err := c.Generate(context.Background(), GenerateRequest{
		Prompt:      "quiet studio scene",
		Fingerprint: "fingerprint123",
	})
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if !strings.Contains(err.Error(), "额度不足") {
		t.Fatalf("Generate error = %q, want quota message", err.Error())
	}
	if called.Load() {
		t.Fatal("upstream was called despite zero quota")
	}
}

func TestGenerateRefundsCreditWhenSubmitFails(t *testing.T) {
	t.Parallel()

	store := newQuotaStore(t)
	if _, err := store.ApplyCheckIn("fingerprint123"); err != nil {
		t.Fatalf("ApplyCheckIn returned error: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		http.Error(w, `{"detail":"upstream down"}`, http.StatusBadGateway)
	}))
	defer ts.Close()

	c := &Client{
		openAIBaseURL: ts.URL + "/v1",
		taskAPIRoot:   ts.URL,
		apiKey:        "test-key",
		model:         "gpt-image-2",
		quota:         store,
		http:          client.New(),
	}

	_, err := c.Generate(context.Background(), GenerateRequest{
		Prompt:      "quiet studio scene",
		Fingerprint: "fingerprint123",
	})
	if err == nil {
		t.Fatal("Generate returned nil error")
	}

	status, err := store.Get("fingerprint123")
	if err != nil {
		t.Fatalf("quota Get returned error: %v", err)
	}
	if status.Balance != quota.DailyGrant {
		t.Fatalf("quota balance = %d, want refunded balance %d", status.Balance, quota.DailyGrant)
	}
}

func TestGenerateRequiresChatGPT2APIConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		client Client
		want   string
	}{
		{
			name: "missing base url",
			client: Client{
				apiKey: "test-key",
			},
			want: "CHATGPT2API_BASE_URL 未配置",
		},
		{
			name: "missing api key",
			client: Client{
				openAIBaseURL: "http://127.0.0.1:3200/v1",
				taskAPIRoot:   "http://127.0.0.1:3200",
			},
			want: "CHATGPT2API_API_KEY 未配置",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.client.Generate(context.Background(), GenerateRequest{Prompt: "quiet studio scene"})
			if err == nil {
				t.Fatal("Generate returned nil error")
			}
			if got := err.Error(); !strings.Contains(got, tt.want) {
				t.Fatalf("error = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeUsesAspectRatioSizes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size string
		want string
	}{
		{
			name: "empty falls back to square ratio",
			size: "",
			want: "1:1",
		},
		{
			name: "square ratio",
			size: "1:1",
			want: "1:1",
		},
		{
			name: "landscape ratio",
			size: "16:9",
			want: "16:9",
		},
		{
			name: "portrait ratio",
			size: "9:16",
			want: "9:16",
		},
		{
			name: "legacy resolution falls back",
			size: "1024x1024",
			want: "1:1",
		},
		{
			name: "auto falls back",
			size: "auto",
			want: "1:1",
		},
		{
			name: "unknown ratio falls back",
			size: "4:3",
			want: "1:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalize(GenerateRequest{Size: tt.size})
			if got.Size != tt.want {
				t.Fatalf("normalize(%q).Size = %q, want %q", tt.size, got.Size, tt.want)
			}
		})
	}
}

func TestNormalizeBaseURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		in         string
		wantOpenAI string
		wantTask   string
	}{
		{
			name:       "root",
			in:         "http://127.0.0.1:3200",
			wantOpenAI: "http://127.0.0.1:3200/v1",
			wantTask:   "http://127.0.0.1:3200",
		},
		{
			name:       "v1 suffix",
			in:         "http://127.0.0.1:3200/v1",
			wantOpenAI: "http://127.0.0.1:3200/v1",
			wantTask:   "http://127.0.0.1:3200",
		},
		{
			name:       "v1 suffix with trailing slash",
			in:         " http://127.0.0.1:3200/v1/ ",
			wantOpenAI: "http://127.0.0.1:3200/v1",
			wantTask:   "http://127.0.0.1:3200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeOpenAIBaseURL(tt.in); got != tt.wantOpenAI {
				t.Fatalf("normalizeOpenAIBaseURL(%q) = %q, want %q", tt.in, got, tt.wantOpenAI)
			}
			if got := normalizeTaskAPIRoot(tt.in); got != tt.wantTask {
				t.Fatalf("normalizeTaskAPIRoot(%q) = %q, want %q", tt.in, got, tt.wantTask)
			}
		})
	}
}

func TestEnhancePromptStreamsUpstreamSSE(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}

		var body openai.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Decode request body: %v", err)
		}
		if body.Model != "gpt-5.5" {
			t.Fatalf("model = %q, want gpt-5.5", body.Model)
		}
		if body.Stream == nil || !*body.Stream {
			t.Fatalf("stream = %v, want true", body.Stream)
		}
		if body.Temperature == nil || *body.Temperature != 0.35 {
			t.Fatalf("temperature = %v, want 0.35", body.Temperature)
		}
		if len(body.Messages) != 2 {
			t.Fatalf("messages length = %d, want 2", len(body.Messages))
		}
		systemPrompt := body.Messages[0].Content
		if body.Messages[0].Role != openai.RSystem ||
			!strings.Contains(systemPrompt, "自适应增强") ||
			!strings.Contains(systemPrompt, "完整输入只做轻量润色") ||
			!strings.Contains(systemPrompt, "不要输出尺寸、比例或横竖幅") {
			t.Fatalf("unexpected system message: %+v", body.Messages[0])
		}
		if body.Messages[1].Role != openai.RUser || !strings.Contains(body.Messages[1].Content, "雨天咖啡馆") {
			t.Fatalf("unexpected user message: %+v", body.Messages[1])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"增强\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer ts.Close()

	c := &Client{
		openAIBaseURL: ts.URL + "/v1",
		taskAPIRoot:   ts.URL,
		apiKey:        "test-key",
		promptModel:   "gpt-5.5",
		rawHTTP:       ts.Client(),
	}
	e := echo.New()
	e.POST("/api/prompts/enhance", c.EnhancePrompt)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/prompts/enhance",
		strings.NewReader(`{"prompt":"雨天咖啡馆","direction":"details"}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get(echo.HeaderContentType); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	if got := rec.Header().Get(echo.HeaderCacheControl); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
	if got := rec.Body.String(); got != "data: {\"choices\":[{\"delta\":{\"content\":\"增强\"}}]}\n\ndata: [DONE]\n\n" {
		t.Fatalf("stream body = %q", got)
	}
}

func TestEnhanceSystemPromptAdaptsByDirection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		direction string
		want      []string
	}{
		{
			name:      "details keeps complete prompts restrained",
			direction: enhanceDirectionDetails,
			want: []string{
				"风格、媒介、画面类型",
				"自适应增强",
				"完整输入只做轻量润色",
				"不要输出尺寸、比例或横竖幅",
				"省略它们",
				"补足缺失的主体细节",
				"不新增与原意无关",
			},
		},
		{
			name:      "creative adds imaginative but bounded changes",
			direction: enhanceDirectionCreative,
			want: []string{
				"风格、媒介、画面类型",
				"自适应增强",
				"完整输入只做轻量润色",
				"不要输出尺寸、比例或横竖幅",
				"省略它们",
				"只选择 1-2 个创意变量",
				"叙事感",
				"不要跑题",
				"堆叠无关元素",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enhanceSystemPrompt(tt.direction)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Fatalf("enhanceSystemPrompt(%q) missing %q in %q", tt.direction, want, got)
				}
			}
		})
	}
}

func TestEnhancePromptValidatesRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		client Client
		body   string
		status int
		want   string
	}{
		{
			name: "empty prompt",
			client: Client{
				openAIBaseURL: "http://127.0.0.1:3200/v1",
				taskAPIRoot:   "http://127.0.0.1:3200",
				apiKey:        "test-key",
				promptModel:   "gpt-5.5",
			},
			body:   `{"prompt":" ","direction":"details"}`,
			status: http.StatusBadRequest,
			want:   "请输入需要增强的提示词",
		},
		{
			name: "unknown direction",
			client: Client{
				openAIBaseURL: "http://127.0.0.1:3200/v1",
				taskAPIRoot:   "http://127.0.0.1:3200",
				apiKey:        "test-key",
				promptModel:   "gpt-5.5",
			},
			body:   `{"prompt":"quiet studio","direction":"photo"}`,
			status: http.StatusBadRequest,
			want:   "提示词增强方向不支持",
		},
		{
			name: "missing prompt model",
			client: Client{
				openAIBaseURL: "http://127.0.0.1:3200/v1",
				taskAPIRoot:   "http://127.0.0.1:3200",
				apiKey:        "test-key",
			},
			body:   `{"prompt":"quiet studio","direction":"details"}`,
			status: http.StatusBadGateway,
			want:   "CHATGPT2API_PROMPT_MODEL 未配置",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.POST("/api/prompts/enhance", tt.client.EnhancePrompt)
			req := httptest.NewRequest(http.MethodPost, "/api/prompts/enhance", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.status, rec.Body.String())
			}
			if got := rec.Body.String(); !strings.Contains(got, tt.want) {
				t.Fatalf("body = %q, want to contain %q", got, tt.want)
			}
		})
	}
}

func newQuotaStore(t *testing.T) *quota.Store {
	t.Helper()
	store, err := quota.Open(filepath.Join(t.TempDir(), "quota.db"))
	if err != nil {
		t.Fatalf("quota Open returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("quota Close returned error: %v", err)
		}
	})
	return store
}
