package controller

import (
	"net/http/httptest"
	"testing"

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
