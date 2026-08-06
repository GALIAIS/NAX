package replicate

import (
	"io"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseOutputSupportsReplicateVideoShapes(t *testing.T) {
	urlValue, duration, count := parseOutput([]any{
		"https://cdn.example/video.mp4",
		"https://cdn.example/video-2.mp4",
	})
	require.Equal(t, "https://cdn.example/video.mp4", urlValue)
	require.Equal(t, float64(0), duration)
	require.Equal(t, 2, count)

	urlValue, duration, count = parseOutput(map[string]any{
		"video_url": "https://cdn.example/video.mp4",
		"duration":  8.5,
	})
	require.Equal(t, "https://cdn.example/video.mp4", urlValue)
	require.Equal(t, 8.5, duration)
	require.Equal(t, 1, count)
}

func TestErrorStringSupportsReplicateErrorObjects(t *testing.T) {
	require.Equal(t, "content policy", errorString(map[string]any{"message": "content policy"}))
	require.Equal(t, "cancelled", errorString("cancelled"))
}

func TestParseTaskResultIncludesProviderTiming(t *testing.T) {
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"status":"succeeded",
		"created_at":"2026-08-06T00:00:00Z",
		"started_at":"2026-08-06T00:00:02Z",
		"completed_at":"2026-08-06T00:00:07Z",
		"output":"https://cdn.example/video.mp4"
	}`))
	require.NoError(t, err)
	require.Equal(t, 2.0, result.QueueSeconds)
	require.Equal(t, 5.0, result.ProcessingSeconds)
	require.Equal(t, 7.0, result.TotalSeconds)
}

func TestVersionedModelUsesModelRouteAndPreservesVersionInput(t *testing.T) {
	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
	// RelayTaskSubmit builds the body before DoRequest builds the URL. Keep that
	// order here so a versioned model cannot silently lose its top-level version.
	adaptor.baseURL = "https://api.replicate.com"
	info.UpstreamModelName = "owner/model:version-123"

	req := httptest.NewRequest("POST", "/v1/videos", nil)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	ctx.Set("task_request", relaycommon.TaskSubmitReq{Prompt: "animate"})
	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	encoded, err := io.ReadAll(body)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"version":"version-123"`)
	require.Contains(t, string(encoded), `"input":{"prompt":"animate"}`)
	urlValue, err := adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://api.replicate.com/v1/predictions", urlValue)
}
