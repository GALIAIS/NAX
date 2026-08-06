package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseSingleVideoRange(t *testing.T) {
	start, end, err := parseSingleVideoRange("bytes=2-5", 10)
	require.NoError(t, err)
	require.Equal(t, 2, start)
	require.Equal(t, 5, end)

	start, end, err = parseSingleVideoRange("bytes=-3", 10)
	require.NoError(t, err)
	require.Equal(t, 7, start)
	require.Equal(t, 9, end)

	_, _, err = parseSingleVideoRange("bytes=10-", 10)
	require.Error(t, err)
}

func TestWriteVideoDataURLHonorsRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest("GET", "/v1/videos/task/content", nil)
	req.Header.Set("Range", "bytes=1-3")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req

	require.NoError(t, writeVideoDataURL(c, "data:video/mp4;base64,AAECAwQFBg=="))
	require.Equal(t, 206, recorder.Code)
	require.Equal(t, []byte{1, 2, 3}, recorder.Body.Bytes())
	require.Equal(t, "bytes 1-3/7", recorder.Header().Get("Content-Range"))
}

func TestResolveVideoURL(t *testing.T) {
	require.Equal(t,
		"https://provider.example/media/video.mp4",
		resolveVideoURL("https://provider.example", "/media/video.mp4"),
	)
	require.Equal(t,
		"https://provider.example/media/video.mp4",
		resolveVideoURL("https://provider.example/api/v1", "../media/video.mp4"),
	)
	require.Equal(t,
		"https://signed.example/video.mp4?token=abc",
		resolveVideoURL("https://provider.example", "https://signed.example/video.mp4?token=abc"),
	)
}

func TestResolveAdvancedCustomVideoContentTargetUsesChannelBase(t *testing.T) {
	baseURL := "http://grok2api:8000"
	channel := &model.Channel{Type: constant.ChannelTypeAdvancedCustom, BaseURL: &baseURL}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{Routes: []dto.AdvancedCustomRoute{{
			IncomingPath: "/v1/videos/generations",
			UpstreamPath: "/v1/videos/generations",
			Models:       []string{"grok-imagine-video"},
		}}},
	})
	task := &model.Task{
		Properties:  model.Properties{OriginModelName: "grok-imagine-video"},
		PrivateData: model.TaskPrivateData{UpstreamTaskID: "video_demo"},
	}

	target, trusted := resolveAdvancedCustomVideoContentTarget(
		channel,
		task,
		"https://public.example/v1/videos/video_demo/content",
	)

	require.True(t, trusted)
	require.Equal(t, "http://grok2api:8000/v1/videos/video_demo/content", target)
}

func TestResolveAdvancedCustomVideoContentTargetKeepsCDNURL(t *testing.T) {
	baseURL := "http://grok2api:8000"
	channel := &model.Channel{Type: constant.ChannelTypeAdvancedCustom, BaseURL: &baseURL}
	task := &model.Task{PrivateData: model.TaskPrivateData{UpstreamTaskID: "video_demo"}}

	target, trusted := resolveAdvancedCustomVideoContentTarget(
		channel,
		task,
		"https://cdn.example/video.mp4",
	)

	require.False(t, trusted)
	require.Equal(t, "https://cdn.example/video.mp4", target)
}
