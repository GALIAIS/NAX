package gemini

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const maxVeoImageSize = 20 * 1024 * 1024

// ExtractMultipartImage reads the first uploaded image/frame field from a
// multipart request and returns a VeoImageInput. Returns nil if no file is
// present or the bounded conversion fails.
func ExtractMultipartImage(c *gin.Context, info *relaycommon.RelayInfo) *VeoImageInput {
	values, err := relaycommon.MultipartImageData(c, "input_reference", "image", "images", "first_frame_image", "last_frame_image", "reference_images")
	if err != nil || len(values) == 0 {
		return nil
	}
	if info != nil {
		info.Action = constant.TaskActionGenerate
	}
	return ParseImageInput(values[0])
}

// ParseImageInput parses an image string (data URI or raw base64) into a
// VeoImageInput. HTTP(S) inputs are fetched through the gateway's SSRF-safe
// client because Veo accepts image bytes rather than arbitrary remote URLs.
func ParseImageInput(imageStr string) *VeoImageInput {
	imageStr = strings.TrimSpace(imageStr)
	if imageStr == "" {
		return nil
	}

	if strings.HasPrefix(imageStr, "data:") {
		return parseDataURI(imageStr)
	}
	if strings.HasPrefix(strings.ToLower(imageStr), "http://") || strings.HasPrefix(strings.ToLower(imageStr), "https://") {
		if err := service.ValidateSSRFProtectedFetchURL(imageStr); err != nil {
			return nil
		}
		resp, err := service.GetSSRFProtectedHTTPClient().Get(imageStr)
		if err != nil {
			return nil
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxVeoImageSize+1))
		if err != nil || len(data) == 0 || len(data) > maxVeoImageSize {
			return nil
		}
		mimeType := resp.Header.Get("Content-Type")
		if index := strings.IndexByte(mimeType, ';'); index >= 0 {
			mimeType = strings.TrimSpace(mimeType[:index])
		}
		if mimeType == "" {
			mimeType = http.DetectContentType(data)
		}
		return &VeoImageInput{
			BytesBase64Encoded: base64.StdEncoding.EncodeToString(data),
			MimeType:           mimeType,
		}
	}

	raw, err := base64.StdEncoding.DecodeString(imageStr)
	if err != nil {
		return nil
	}
	return &VeoImageInput{
		BytesBase64Encoded: imageStr,
		MimeType:           http.DetectContentType(raw),
	}
}

func parseDataURI(uri string) *VeoImageInput {
	// data:image/png;base64,iVBOR...
	rest := uri[len("data:"):]
	idx := strings.Index(rest, ",")
	if idx < 0 {
		return nil
	}
	meta := rest[:idx]
	b64 := rest[idx+1:]
	if b64 == "" {
		return nil
	}

	mimeType := "application/octet-stream"
	parts := strings.SplitN(meta, ";", 2)
	if len(parts) >= 1 && parts[0] != "" {
		mimeType = parts[0]
	}

	return &VeoImageInput{
		BytesBase64Encoded: b64,
		MimeType:           mimeType,
	}
}
