package common

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	rootcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultipartImageDataNormalizesImageAndFrameFiles(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("first_frame_image", "first.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-png"))
	require.NoError(t, err)
	part, err = writer.CreateFormFile("last_frame_image", "last.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-png-last"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/v1/videos", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	values, err := MultipartImageData(context, "first_frame_image", "last_frame_image")
	require.NoError(t, err)
	require.Len(t, values, 2)
	assert.True(t, strings.HasPrefix(values[0], "data:"))
	assert.Contains(t, values[0], ";base64,")
	assert.True(t, strings.HasPrefix(values[1], "data:"))
	assert.Contains(t, values[1], ";base64,")
}

func TestRewriteMultipartTaskModelPreservesFieldsAndFiles(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "client-model"))
	require.NoError(t, writer.WriteField("prompt", "animate"))
	part, err := writer.CreateFormFile("image", "first.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-png"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/v1/videos", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	rewritten, err := RewriteMultipartTaskModel(context, "upstream-model")
	require.NoError(t, err)
	require.NotNil(t, rewritten)
	rewrittenBytes, err := io.ReadAll(rewritten)
	require.NoError(t, err)
	_, params, err := mime.ParseMediaType(context.GetHeader("Content-Type"))
	require.NoError(t, err)
	form, err := multipart.NewReader(bytes.NewReader(rewrittenBytes), params["boundary"]).ReadForm(1 << 20)
	require.NoError(t, err)
	require.Equal(t, []string{"upstream-model"}, form.Value["model"])
	require.Equal(t, []string{"animate"}, form.Value["prompt"])
	require.Len(t, form.File["image"], 1)
}

func TestSanitizeURLForLogMasksSensitiveQueryValues(t *testing.T) {
	rawURL := "https://example.test/v1beta/models/gemini:streamGenerateContent?alt=sse&key=sk-secret&access_token=ya29-secret&api-version=2024-02-01"

	got := SanitizeURLForLog(rawURL)

	assert.NotContains(t, got, "sk-secret")
	assert.NotContains(t, got, "ya29-secret")
	parsedURL, err := url.Parse(got)
	require.NoError(t, err)
	query := parsedURL.Query()
	assert.Equal(t, "***masked***", query.Get("key"))
	assert.Equal(t, "***masked***", query.Get("access_token"))
	assert.Equal(t, "sse", query.Get("alt"))
	assert.Equal(t, "2024-02-01", query.Get("api-version"))
}

func TestSanitizeURLForLogMasksAWSAndSecretLikeQueryKeys(t *testing.T) {
	rawURL := "https://example.test/path?X-Amz-Credential=credential&X-Amz-Signature=signature&session_token=session&client_secret=secret&model=gpt-test"

	got := SanitizeURLForLog(rawURL)

	assert.NotContains(t, got, "X-Amz-Credential=credential")
	assert.NotContains(t, got, "X-Amz-Signature=signature")
	assert.NotContains(t, got, "session_token=session")
	assert.NotContains(t, got, "client_secret=secret")
	parsedURL, err := url.Parse(got)
	require.NoError(t, err)
	query := parsedURL.Query()
	assert.Equal(t, "***masked***", query.Get("X-Amz-Credential"))
	assert.Equal(t, "***masked***", query.Get("X-Amz-Signature"))
	assert.Equal(t, "***masked***", query.Get("session_token"))
	assert.Equal(t, "***masked***", query.Get("client_secret"))
	assert.Equal(t, "gpt-test", query.Get("model"))
}

func TestSanitizeURLForLogKeepsURLWithoutSensitiveQuery(t *testing.T) {
	rawURL := "https://example.test/v1/chat/completions?api-version=2024-02-01&alt=sse"

	got := SanitizeURLForLog(rawURL)

	assert.Equal(t, rawURL, got)
}

func TestValidateMultipartDirectNormalizesImageField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := strings.NewReader(`{"model":"wan2.7-i2v","prompt":"animate","image":" https://example.com/first.png "}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/video/generations", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	info := &RelayInfo{
		TaskRelayInfo: &TaskRelayInfo{},
	}

	taskErr := ValidateMultipartDirect(context, info)

	require.Nil(t, taskErr)
	storedReq, err := GetTaskRequest(context)
	require.NoError(t, err)
	require.Equal(t, []string{"https://example.com/first.png"}, storedReq.Images)
	require.Equal(t, constant.TaskActionGenerate, info.Action)
}

func TestValidateMultipartTaskRequestPreservesVideoProtocolFields(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{
		"model":             "video-model",
		"prompt":            "animate",
		"duration":          "5.5",
		"n":                 "2",
		"fps":               "24",
		"input_reference":   "https://example.test/first.png",
		"first_frame_image": "https://example.test/first.png",
		"last_frame_image":  "https://example.test/last.png",
		"resolution":        "720p",
		"aspect_ratio":      "16:9",
	} {
		require.NoError(t, writer.WriteField(key, value))
	}
	require.NoError(t, writer.WriteField("reference_images", "https://example.test/ref.png"))
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/v1/videos", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	req, err := validateMultipartTaskRequest(context, &RelayInfo{}, constant.TaskActionGenerate)
	require.NoError(t, err)
	assert.Equal(t, "5.5", req.Seconds)
	assert.Equal(t, 2, req.N)
	assert.Equal(t, 24, req.FPS)
	assert.Equal(t, "https://example.test/first.png", req.InputReference)
	assert.Equal(t, []string{"https://example.test/ref.png"}, req.ReferenceImages)
	assert.Equal(t, "720p", req.Resolution)
}

// TestTaskDurationBounds guards the billing invariant that user-supplied
// video duration (a quota multiplier via OtherRatio "seconds") is bounded, so
// it can never overflow quota calculation into a negative charge.
func TestTaskDurationBounds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newContext := func(t *testing.T, body string) (*gin.Context, *RelayInfo) {
		request := httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = request
		return context, &RelayInfo{TaskRelayInfo: &TaskRelayInfo{}}
	}

	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "huge duration is rejected",
			body:    `{"model":"sora-2","prompt":"a cat","duration":9999999999}`,
			wantErr: true,
		},
		{
			name:    "huge seconds string is rejected",
			body:    `{"model":"sora-2","prompt":"a cat","seconds":"9999999999"}`,
			wantErr: true,
		},
		{
			name:    "negative duration is rejected",
			body:    `{"model":"sora-2","prompt":"a cat","duration":-8}`,
			wantErr: true,
		},
		{
			name: "normal duration is accepted",
			body: `{"model":"sora-2","prompt":"a cat","seconds":"8"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" (multipart direct)", func(t *testing.T) {
			context, info := newContext(t, tt.body)
			taskErr := ValidateMultipartDirect(context, info)
			if tt.wantErr {
				require.NotNil(t, taskErr)
				require.Equal(t, "invalid_seconds", taskErr.Code)
			} else {
				require.Nil(t, taskErr)
			}
		})
		t.Run(tt.name+" (basic task request)", func(t *testing.T) {
			context, info := newContext(t, tt.body)
			taskErr := ValidateBasicTaskRequest(context, info, constant.TaskActionGenerate)
			if tt.wantErr {
				require.NotNil(t, taskErr)
				require.Equal(t, "invalid_seconds", taskErr.Code)
			} else {
				require.Nil(t, taskErr)
			}
		})
	}
}

func TestTaskSubmitReqNormalizesVideoProtocolNumbersAndBillingFields(t *testing.T) {
	var req TaskSubmitReq
	require.NoError(t, rootcommon.Unmarshal([]byte(`{
		"model":"video-model",
		"prompt":"a moving scene",
		"duration": "5.5",
		"n": "2",
			"fps": "24",
			"seed": "42",
			"width": "1280",
			"height": 720,
			"reference_images":["https://example.test/a.png"],
		"metadata":{"sampleCount":3,"durationSeconds":7}
	}`), &req))

	assert.InDelta(t, 5.5, req.RequestedDurationSeconds(), 0.0001)
	assert.Equal(t, 2, req.RequestedOutputCount())
	assert.Equal(t, 24, req.FPS)
	assert.Equal(t, int64(42), req.Seed)
	assert.Equal(t, 1280, req.Width)
	assert.Equal(t, 720, req.Height)
	assert.Equal(t, []string{"https://example.test/a.png"}, req.ImageList())
	assert.Equal(t, map[string]float64{"seconds": 5.5, "count": 2}, TaskRequestBillingRatios(req))
}

func TestTaskSubmitReqRejectsInvalidSecondsAndBoundsBatchFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	newContext := func(body string) *gin.Context {
		request := httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = request
		return context
	}

	for _, test := range []struct {
		name string
		body string
		code string
	}{
		{name: "invalid seconds", body: `{"model":"sora-2","prompt":"x","seconds":"not-a-number"}`, code: "invalid_seconds"},
		{name: "batch count overflow", body: `{"model":"sora-2","prompt":"x","n":17}`, code: "invalid_n"},
		{name: "fps overflow", body: `{"model":"sora-2","prompt":"x","fps":241}`, code: "invalid_fps"},
		{name: "width overflow", body: `{"model":"sora-2","prompt":"x","width":16385}`, code: "invalid_width"},
	} {
		t.Run(test.name, func(t *testing.T) {
			info := &RelayInfo{TaskRelayInfo: &TaskRelayInfo{}}
			err := ValidateMultipartDirect(newContext(test.body), info)
			require.NotNil(t, err)
			assert.Equal(t, test.code, err.Code)
		})
	}
}

func TestTaskSubmitReqPreservesProviderSpecificRootFields(t *testing.T) {
	var req TaskSubmitReq
	require.NoError(t, rootcommon.Unmarshal([]byte(`{
		"model":"video-model",
		"prompt":"a moving scene",
		"audio_url":"https://example.test/voice.mp3",
		"watermark":false,
		"camera_motion":{"pan":"left"},
		"metadata":{"watermark":true,"custom":"kept"}
	}`), &req))

	assert.Equal(t, "https://example.test/voice.mp3", req.Metadata["audio_url"])
	assert.Equal(t, true, req.Metadata["watermark"], "explicit metadata should win over a root field")
	assert.Equal(t, map[string]interface{}{"pan": "left"}, req.Metadata["camera_motion"])
	assert.Equal(t, "kept", req.Metadata["custom"])
}
