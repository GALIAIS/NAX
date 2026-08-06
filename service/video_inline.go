package service

import (
	"encoding/base64"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// maxInlineVideoBase64Chars bounds the amount of provider data copied into a
// task's private JSON. The upstream response has already been read into
// memory, but an unbounded value here could make a single task row exhaust the
// database or the polling worker. Larger media remains resolvable by the
// provider-specific fetcher when the content endpoint is requested.
const maxInlineVideoBase64Chars = 128 << 20

// extractInlineVideoDataURL extracts the inline video representation used by
// Vertex/Gemini operation APIs. Those APIs may return one of several shapes:
// response.videos[].bytesBase64Encoded, response.bytesBase64Encoded, or a raw
// response.video string. The result is kept in TaskPrivateData rather than
// Task.Data so redacted task history never exposes a large base64 payload.
func extractInlineVideoDataURL(body []byte) string {
	var root map[string]any
	if len(body) == 0 || common.Unmarshal(body, &root) != nil {
		return ""
	}
	return findInlineVideoDataURL(root, 0)
}

// ExtractInlineVideoDataURL is the relay/controller boundary for providers
// that return video bytes inline. The lower-case implementation remains the
// canonical helper used by the polling service and is intentionally exposed
// only as a narrow parser API.
func ExtractInlineVideoDataURL(body []byte) string {
	return extractInlineVideoDataURL(body)
}

func findInlineVideoDataURL(value any, depth int) string {
	if depth > 12 {
		return ""
	}
	switch node := value.(type) {
	case map[string]any:
		if result := inlineVideoFromMap(node); result != "" {
			return result
		}
		for _, child := range node {
			if result := findInlineVideoDataURL(child, depth+1); result != "" {
				return result
			}
		}
	case []any:
		for _, child := range node {
			if result := findInlineVideoDataURL(child, depth+1); result != "" {
				return result
			}
		}
	}
	return ""
}

func inlineVideoFromMap(value map[string]any) string {
	if value == nil {
		return ""
	}
	mimeType := stringValue(value, "mimeType", "mime_type", "contentType", "content_type")
	encoding := stringValue(value, "encoding", "format", "file_type", "fileType")
	for _, key := range []string{
		"bytesBase64Encoded",
		"bytes_base64_encoded",
		"video_base64",
		"videoBase64",
		"base64_data",
		"base64Data",
	} {
		if encoded, ok := value[key].(string); ok {
			if result := makeInlineVideoDataURL(encoded, mimeType, encoding); result != "" {
				return result
			}
		}
	}
	// Some gateways use `video` or `data` for the payload. Only strings are
	// considered here; nested objects are traversed by findInlineVideoDataURL.
	for _, key := range []string{"video", "data"} {
		if raw, ok := value[key].(string); ok {
			if result := makeInlineVideoDataURL(raw, mimeType, encoding); result != "" {
				return result
			}
		}
	}
	return ""
}

func makeInlineVideoDataURL(raw, mimeType, encoding string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(raw), "data:") {
		if strings.Contains(strings.ToLower(raw[:inlineMinInt(len(raw), 256)]), ";base64,") && len(raw) <= maxInlineVideoBase64Chars+256 {
			return raw
		}
		return ""
	}
	if strings.HasPrefix(strings.ToLower(raw), "http://") || strings.HasPrefix(strings.ToLower(raw), "https://") {
		return ""
	}
	if len(raw) > maxInlineVideoBase64Chars {
		return ""
	}
	// Validate before persisting. Both padded and raw base64 are emitted by
	// different Google-compatible gateways; the original spelling is retained
	// so the content endpoint can decode it without another transformation.
	if _, err := base64.StdEncoding.DecodeString(raw); err != nil {
		if _, rawErr := base64.RawStdEncoding.DecodeString(raw); rawErr != nil {
			return ""
		}
	}
	return "data:" + videoMimeType(mimeType, encoding) + ";base64," + raw
}

func videoMimeType(mimeType, encoding string) string {
	mimeType = strings.TrimSpace(mimeType)
	if strings.Contains(mimeType, "/") {
		return mimeType
	}
	encoding = strings.TrimSpace(strings.ToLower(encoding))
	if strings.Contains(encoding, "/") {
		return encoding
	}
	switch encoding {
	case "webm":
		return "video/webm"
	case "mov", "quicktime":
		return "video/quicktime"
	default:
		return "video/mp4"
	}
}

func stringValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func inlineMinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
