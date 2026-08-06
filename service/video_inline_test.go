package service

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractInlineVideoDataURLFromVertexResponse(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("video-bytes"))
	body := []byte(`{"name":"operations/1","done":true,"response":{"videos":[{"mimeType":"video/webm","bytesBase64Encoded":"` + encoded + `"}]}}`)

	result := extractInlineVideoDataURL(body)
	require.Equal(t, "data:video/webm;base64,"+encoded, result)
}

func TestExtractInlineVideoDataURLIgnoresRemoteURL(t *testing.T) {
	body := []byte(`{"response":{"videos":[{"uri":"https://storage.example/video.mp4"}]}}`)
	require.Empty(t, extractInlineVideoDataURL(body))
}

func TestExtractInlineVideoDataURLAcceptsRawBase64DataURL(t *testing.T) {
	encoded := base64.RawStdEncoding.EncodeToString([]byte("video-bytes"))
	body := []byte(`{"response":{"video":"data:video/mp4;base64,` + encoded + `"}}`)
	require.Equal(t, "data:video/mp4;base64,"+encoded, extractInlineVideoDataURL(body))
}

func TestRedactVideoResponseBodyRemovesInlineDataFields(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 300)))
	redacted := string(RedactVideoResponseBody([]byte(`{"response":{"video":"data:video/mp4;base64,` + encoded + `","bytesBase64Encoded":"` + encoded + `"}}`)))
	require.NotContains(t, redacted, encoded)
	require.Contains(t, redacted, `"video":"data:video/mp4;base64,`)
}
