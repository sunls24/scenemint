package image

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"scenemint/internal/quota"

	"github.com/labstack/echo/v5"
	"github.com/sunls24/gox/network/client"
	"github.com/sunls24/gox/openai"
	"github.com/sunls24/gox/server"
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
			body.Size != "1:1" {
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

func testReferenceUpload(data []byte) *referenceUpload {
	return &referenceUpload{
		Reader:      bytes.NewReader(data),
		Filename:    "reference.png",
		ContentType: "image/png",
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
		if r.ContentLength <= 0 {
			t.Fatalf("ContentLength = %d, want known positive length", r.ContentLength)
		}
		if len(r.TransferEncoding) != 0 {
			t.Fatalf("TransferEncoding = %v, want no chunked transfer encoding", r.TransferEncoding)
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
		imageUpload:  testReferenceUpload([]byte("\x89PNG\r\n\x1a\n")),
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
		if r.ContentLength <= 0 {
			t.Fatalf("ContentLength = %d, want known positive length", r.ContentLength)
		}
		if len(r.TransferEncoding) != 0 {
			t.Fatalf("TransferEncoding = %v, want no chunked transfer encoding", r.TransferEncoding)
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
		imageUpload:  testReferenceUpload(largeImage),
	})
	if err != nil {
		t.Fatalf("submitEditTask returned error: %v", err)
	}
}

func TestSubmitEditTaskRequiresReferenceUpload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body taskSubmitRequest
	}{
		{
			name: "missing upload",
			body: taskSubmitRequest{},
		},
		{
			name: "nil reader",
			body: taskSubmitRequest{
				imageUpload: &referenceUpload{
					Filename:    "reference.png",
					ContentType: "image/png",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&Client{}).submitEditTask(context.Background(), tt.body)
			if err == nil {
				t.Fatal("submitEditTask returned nil error")
			}
			if got := err.Error(); !strings.Contains(got, "参考图不能为空") {
				t.Fatalf("error = %q, want reference upload message", got)
			}
		})
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
		imageUpload:  testReferenceUpload([]byte("\x89PNG\r\n\x1a\n")),
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

const (
	testGenerateHTTPFingerprint     = "fingerprint123"
	testGenerateHTTPPrompt          = "replace the background"
	testGenerateHTTPSize            = "1:1"
	testGenerateHTTPModel           = "gpt-image-2"
	testGenerateHTTPReferenceName   = "reference.png"
	testGenerateHTTPReferenceType   = "image/png"
	testGenerateHTTPGenerationsPath = "/api/image-tasks/generations"
	testGenerateHTTPEditsPath       = "/api/image-tasks/edits"
	testGenerateHTTPGeneratePath    = "/api/images/generate"
	testGenerateHTTPAuthorization   = "Bearer test-key"
	testGenerateHTTPReferenceBytes  = "\x89PNG\r\n\x1a\n"
)

func newGenerateHTTPQuotaStore(t *testing.T) *quota.Store {
	t.Helper()
	store := newQuotaStore(t)
	if _, err := store.ApplyCheckIn(testGenerateHTTPFingerprint); err != nil {
		t.Fatalf("ApplyCheckIn returned error: %v", err)
	}
	return store
}

func newGenerateHTTPTestClient(store *quota.Store, ts *httptest.Server) *Client {
	return &Client{
		openAIBaseURL: ts.URL + "/v1",
		taskAPIRoot:   ts.URL,
		apiKey:        "test-key",
		model:         testGenerateHTTPModel,
		quota:         store,
		http:          client.New(client.WithClient(ts.Client())),
		rawHTTP:       ts.Client(),
	}
}

func newJSONGenerateRequest(t *testing.T) *http.Request {
	t.Helper()
	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(GenerateRequest{
		Prompt:      testGenerateHTTPPrompt,
		Size:        testGenerateHTTPSize,
		Fingerprint: testGenerateHTTPFingerprint,
	})
	if err != nil {
		t.Fatalf("Encode JSON request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, testGenerateHTTPGeneratePath, &body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	return req
}

func newMultipartGenerateRequest(t *testing.T, imageBytes []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fields := [][2]string{
		{"prompt", testGenerateHTTPPrompt},
		{"size", testGenerateHTTPSize},
		{"fingerprint", testGenerateHTTPFingerprint},
	}
	for _, field := range fields {
		if err := writer.WriteField(field[0], field[1]); err != nil {
			t.Fatalf("WriteField %s: %v", field[0], err)
		}
	}
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     "image",
		"filename": testGenerateHTTPReferenceName,
	}))
	partHeader.Set("Content-Type", testGenerateHTTPReferenceType)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(imageBytes); err != nil {
		t.Fatalf("Write image: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, testGenerateHTTPGeneratePath, &body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	return req
}

func serveGenerateHTTP(t *testing.T, c *Client, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	srv := server.New(func(srv *server.Server) {
		srv.Echo.POST(testGenerateHTTPGeneratePath, server.WrapReplyResp(c.GenerateReply))
	})
	rec := httptest.NewRecorder()
	srv.Echo.ServeHTTP(rec, req)
	return rec
}

func TestGenerateHTTPAcceptsJSONTextGeneration(t *testing.T) {
	t.Parallel()

	store := newGenerateHTTPQuotaStore(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != testGenerateHTTPGenerationsPath {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != testGenerateHTTPAuthorization {
			t.Fatalf("Authorization = %q, want %q", got, testGenerateHTTPAuthorization)
		}
		if got := r.Header.Get("Content-Type"); got != echo.MIMEApplicationJSON {
			t.Fatalf("Content-Type = %q, want %q", got, echo.MIMEApplicationJSON)
		}

		var body taskSubmitRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Decode request body: %v", err)
		}
		if body.ClientTaskID == "" {
			t.Fatal("client_task_id is empty")
		}
		if body.Prompt != testGenerateHTTPPrompt ||
			body.Model != testGenerateHTTPModel ||
			body.Size != testGenerateHTTPSize {
			t.Fatalf("unexpected request body: %+v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(upstreamTask{
			ID:     "task-1",
			Status: "queued",
			Mode:   "generation",
			Model:  testGenerateHTTPModel,
			Size:   testGenerateHTTPSize,
		}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}))
	defer ts.Close()

	c := newGenerateHTTPTestClient(store, ts)
	rec := serveGenerateHTTP(t, c, newJSONGenerateRequest(t))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var env struct {
		Code int              `json:"code"`
		Data GenerateResponse `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("Decode response: %v", err)
	}
	if env.Code != 0 {
		t.Fatalf("response code = %d, want 0", env.Code)
	}
	if env.Data.Mode != "text" || env.Data.Status != "queued" {
		t.Fatalf("unexpected response data: %+v", env.Data)
	}
	if env.Data.RemainingCredits == nil || *env.Data.RemainingCredits != quota.DailyGrant-1 {
		t.Fatalf("RemainingCredits = %v, want %d", env.Data.RemainingCredits, quota.DailyGrant-1)
	}
}

func TestGenerateHTTPFallsBackToSubmittedTaskFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		request  func(t *testing.T) *http.Request
		path     string
		wantMode string
	}{
		{
			name:     "json text generation",
			request:  newJSONGenerateRequest,
			path:     testGenerateHTTPGenerationsPath,
			wantMode: "text",
		},
		{
			name: "multipart reference image",
			request: func(t *testing.T) *http.Request {
				t.Helper()
				return newMultipartGenerateRequest(t, []byte(testGenerateHTTPReferenceBytes))
			},
			path:     testGenerateHTTPEditsPath,
			wantMode: "image",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := newGenerateHTTPQuotaStore(t)
			submittedIDs := make(chan string, 1)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					t.Fatalf("unexpected path %q", r.URL.Path)
				}

				var submittedID string
				if tt.path == testGenerateHTTPGenerationsPath {
					var body taskSubmitRequest
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatalf("Decode request body: %v", err)
					}
					submittedID = body.ClientTaskID
				} else {
					if err := r.ParseMultipartForm(20 << 20); err != nil {
						t.Fatalf("ParseMultipartForm: %v", err)
					}
					submittedID = r.FormValue("client_task_id")
				}
				if strings.TrimSpace(submittedID) == "" {
					t.Fatal("client_task_id is empty")
				}
				submittedIDs <- submittedID

				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(upstreamTask{
					Status: "queued",
				}); err != nil {
					t.Fatalf("Encode: %v", err)
				}
			}))
			defer ts.Close()

			c := newGenerateHTTPTestClient(store, ts)
			rec := serveGenerateHTTP(t, c, tt.request(t))

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
			}
			var env struct {
				Code int              `json:"code"`
				Data GenerateResponse `json:"data"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
				t.Fatalf("Decode response: %v", err)
			}
			if env.Code != 0 {
				t.Fatalf("response code = %d, want 0", env.Code)
			}

			var submittedID string
			select {
			case submittedID = <-submittedIDs:
			case <-time.After(time.Second):
				t.Fatal("upstream did not receive submitted task id")
			}
			if env.Data.ID != submittedID {
				t.Fatalf("response id = %q, want submitted client_task_id %q", env.Data.ID, submittedID)
			}
			if env.Data.Mode != tt.wantMode {
				t.Fatalf("response mode = %q, want %q", env.Data.Mode, tt.wantMode)
			}
			if env.Data.Size != testGenerateHTTPSize {
				t.Fatalf("response size = %q, want %q", env.Data.Size, testGenerateHTTPSize)
			}
			if _, err := time.Parse(time.RFC3339, env.Data.CreatedAt); err != nil {
				t.Fatalf("response createdAt = %q, want RFC3339: %v", env.Data.CreatedAt, err)
			}
		})
	}
}

func TestGenerateHTTPAcceptsMultipartReferenceImage(t *testing.T) {
	t.Parallel()

	store := newGenerateHTTPQuotaStore(t)
	imageBytes := []byte(testGenerateHTTPReferenceBytes)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != testGenerateHTTPEditsPath {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != testGenerateHTTPAuthorization {
			t.Fatalf("Authorization = %q, want %q", got, testGenerateHTTPAuthorization)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data; boundary=") {
			t.Fatalf("Content-Type = %q, want multipart/form-data", got)
		}
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if got := r.FormValue("client_task_id"); strings.TrimSpace(got) == "" {
			t.Fatal("client_task_id is empty")
		}
		wantFields := map[string]string{
			"prompt": testGenerateHTTPPrompt,
			"model":  testGenerateHTTPModel,
			"size":   testGenerateHTTPSize,
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
		if got := files[0].Filename; got != testGenerateHTTPReferenceName {
			t.Fatalf("image filename = %q, want %q", got, testGenerateHTTPReferenceName)
		}
		if got := files[0].Header.Get("Content-Type"); got != testGenerateHTTPReferenceType {
			t.Fatalf("image Content-Type = %q, want %q", got, testGenerateHTTPReferenceType)
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
		if !bytes.Equal(data, imageBytes) {
			t.Fatalf("image bytes = %q, want %q", data, imageBytes)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(upstreamTask{
			ID:     "task-1",
			Status: "queued",
			Mode:   "edit",
			Model:  testGenerateHTTPModel,
			Size:   testGenerateHTTPSize,
		}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}))
	defer ts.Close()

	c := newGenerateHTTPTestClient(store, ts)
	rec := serveGenerateHTTP(t, c, newMultipartGenerateRequest(t, imageBytes))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var env struct {
		Code int              `json:"code"`
		Data GenerateResponse `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("Decode response: %v", err)
	}
	if env.Code != 0 {
		t.Fatalf("response code = %d, want 0", env.Code)
	}
	if env.Data.Mode != "image" || env.Data.Status != "queued" {
		t.Fatalf("unexpected response data: %+v", env.Data)
	}
	if env.Data.RemainingCredits == nil || *env.Data.RemainingCredits != quota.DailyGrant-1 {
		t.Fatalf("RemainingCredits = %v, want %d", env.Data.RemainingCredits, quota.DailyGrant-1)
	}
}

func TestGenerateHTTPRejectsInvalidMultipartReferenceImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		imageBytes []byte
		want       string
	}{
		{
			name:       "empty image",
			imageBytes: []byte{},
			want:       "参考图不能为空",
		},
		{
			name:       "oversized image",
			imageBytes: bytes.Repeat([]byte{0xab}, maxReferenceUploadBytes+1),
			want:       "参考图不能超过 10MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serveGenerateHTTP(
				t,
				&Client{},
				newMultipartGenerateRequest(t, tt.imageBytes),
			)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
			}
			var env struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
				t.Fatalf("Decode response: %v", err)
			}
			if env.Code == 0 || !strings.Contains(env.Message, tt.want) {
				t.Fatalf("unexpected response envelope: %+v, want message containing %q", env, tt.want)
			}
		})
	}
}

func TestGenerateHTTPRefundsCreditWhenMultipartSubmitFails(t *testing.T) {
	t.Parallel()

	store := newGenerateHTTPQuotaStore(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != testGenerateHTTPEditsPath {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		http.Error(w, `{"detail":"upstream down"}`, http.StatusBadGateway)
	}))
	defer ts.Close()

	c := newGenerateHTTPTestClient(store, ts)
	rec := serveGenerateHTTP(
		t,
		c,
		newMultipartGenerateRequest(t, []byte(testGenerateHTTPReferenceBytes)),
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var env struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("Decode response: %v", err)
	}
	if env.Code == 0 || !strings.Contains(env.Message, "图片任务提交失败") {
		t.Fatalf("unexpected response envelope: %+v", env)
	}

	status, err := store.Get(testGenerateHTTPFingerprint)
	if err != nil {
		t.Fatalf("quota Get returned error: %v", err)
	}
	if status.Balance != quota.DailyGrant {
		t.Fatalf("quota balance = %d, want refunded balance %d", status.Balance, quota.DailyGrant)
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
			want: "图片服务暂不可用，请稍后再试",
		},
		{
			name: "missing api key",
			client: Client{
				openAIBaseURL: "http://127.0.0.1:3200/v1",
				taskAPIRoot:   "http://127.0.0.1:3200",
			},
			want: "图片服务暂不可用，请稍后再试",
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
			want:   "提示词服务暂不可用，请稍后再试",
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
