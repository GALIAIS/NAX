package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

// videoProxyError returns a standardized OpenAI-style error response.
func videoProxyError(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errType,
		},
	})
}

func VideoProxy(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", "task_id is required")
		return
	}

	userID := c.GetInt("id")
	task, exists, err := model.GetByTaskId(userID, taskID)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to query task %s: %s", taskID, err.Error()))
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to query task")
		return
	}
	if !exists || task == nil {
		videoProxyError(c, http.StatusNotFound, "invalid_request_error", "Task not found")
		return
	}

	if task.Status != model.TaskStatusSuccess {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error",
			fmt.Sprintf("Task is not completed yet, current status: %s", task.Status))
		return
	}

	channel, err := model.CacheGetChannel(task.ChannelId)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to get channel for task %s: %s", taskID, err.Error()))
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to retrieve channel information")
		return
	}
	baseURL := channel.GetBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	var videoURL string
	trustedChannelTarget := false
	proxy := channel.GetSetting().Proxy
	client := service.GetSSRFProtectedHTTPClient()
	if proxy != "" {
		// 渠道代理路径的连接由代理侧建立，无法做拨号时逐 IP 校验，
		// 因此后面对 videoURL 保留请求前的一次性 SSRF 校验。
		client, err = service.GetHttpClientWithProxy(proxy)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to create proxy client for task %s: %s", taskID, err.Error()))
			videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create proxy client")
			return
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	method := c.Request.Method
	if method != http.MethodGet && method != http.MethodHead {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, "", nil)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to create request: %s", err.Error()))
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create proxy request")
		return
	}
	for _, header := range []string{"Range", "If-Range", "Accept", "If-None-Match", "If-Modified-Since"} {
		if value := c.GetHeader(header); value != "" {
			req.Header.Set(header, value)
		}
	}

	switch channel.Type {
	case constant.ChannelTypeGemini:
		apiKey := task.PrivateData.Key
		if inlineResult := strings.TrimSpace(task.PrivateData.InlineResult); strings.HasPrefix(strings.ToLower(inlineResult), "data:") {
			videoURL = inlineResult
		} else {
			if apiKey == "" {
				logger.LogError(c.Request.Context(), fmt.Sprintf("Missing stored API key for Gemini task %s", taskID))
				videoProxyError(c, http.StatusInternalServerError, "server_error", "API key not stored for task")
				return
			}
			videoURL, err = getGeminiVideoURL(channel, task, apiKey)
			if err != nil {
				logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to resolve Gemini video URL for task %s: %s", taskID, err.Error()))
				videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to resolve Gemini video URL")
				return
			}
			req.Header.Set("x-goog-api-key", apiKey)
		}
	case constant.ChannelTypeVertexAi:
		if inlineResult := strings.TrimSpace(task.PrivateData.InlineResult); strings.HasPrefix(strings.ToLower(inlineResult), "data:") {
			videoURL = inlineResult
		} else {
			videoURL, err = getVertexVideoURL(channel, task)
			if err != nil {
				logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to resolve Vertex video URL for task %s: %s", taskID, err.Error()))
				videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to resolve Vertex video URL")
				return
			}
		}
	case constant.ChannelTypeOpenAI, constant.ChannelTypeSora:
		videoURL = fmt.Sprintf("%s/v1/videos/%s/content", baseURL, task.GetUpstreamTaskID())
		req.Header.Set("Authorization", "Bearer "+channel.Key)
	case constant.ChannelTypeAdvancedCustom:
		// Advanced custom video adaptors may return a provider media URL rather
		// than a New API proxy URL. Apply the configured route authentication
		// when the provider requires a header/query key; signed media URLs can
		// remain unauthenticated.
		videoURL = task.GetResultURL()
		videoURL, trustedChannelTarget = resolveAdvancedCustomVideoContentTarget(channel, task, videoURL)
		videoURL = applyAdvancedCustomVideoAuth(channel, task, req, videoURL)
	default:
		// Video URL is stored in PrivateData.ResultURL (fallback to FailReason for old data)
		videoURL = task.GetResultURL()
	}

	videoURL = strings.TrimSpace(videoURL)
	if videoURL == "" {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Video URL is empty for task %s", taskID))
		videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch video content")
		return
	}
	videoURL = resolveVideoURL(baseURL, videoURL)
	if isTaskProxyContentURL(videoURL, taskID) {
		// A provider without a resolvable media URL must not be fetched through
		// this endpoint again: doing so would recurse into /content until the
		// request times out. Gemini/Vertex resolve their URL before this guard.
		videoProxyError(c, http.StatusBadGateway, "server_error", "Provider did not return a downloadable video URL")
		return
	}

	if strings.HasPrefix(videoURL, "data:") {
		if err := writeVideoDataURL(c, videoURL); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to decode video data URL for task %s: %s", taskID, err.Error()))
			videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch video content")
		}
		return
	}

	var validateErr error
	if trustedChannelTarget {
		// This target is derived exclusively from the administrator-configured
		// channel base URL and the task's stored upstream ID. It follows the same
		// trust boundary as ordinary channel relay requests and may legitimately
		// point at an internal service name in a Compose network.
		client, err = service.GetHttpClientWithProxy(proxy)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to create channel client for task %s: %s", taskID, err.Error()))
			videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create proxy client")
			return
		}
	} else if proxy == "" {
		validateErr = service.ValidateSSRFProtectedFetchURL(videoURL)
	} else {
		fetchSetting := system_setting.GetFetchSetting()
		validateErr = common.ValidateURLWithFetchSetting(videoURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain)
	}
	if validateErr != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Video URL blocked for task %s: %v", taskID, validateErr))
		videoProxyError(c, http.StatusForbidden, "server_error", fmt.Sprintf("request blocked: %v", validateErr))
		return
	}

	req.URL, err = url.Parse(videoURL)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to parse URL %s: %s", videoURL, err.Error()))
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create proxy request")
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to fetch video from %s: %s", videoURL, err.Error()))
		videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch video content")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusNotModified {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Upstream returned status %d for %s", resp.StatusCode, videoURL))
		videoProxyError(c, http.StatusBadGateway, "server_error",
			fmt.Sprintf("Upstream service returned status %d", resp.StatusCode))
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	c.Writer.Header().Set("Cache-Control", "public, max-age=86400")
	c.Writer.WriteHeader(resp.StatusCode)
	if method == http.MethodHead || resp.StatusCode == http.StatusNotModified {
		return
	}
	if _, err = io.Copy(c.Writer, resp.Body); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to stream video content: %s", err.Error()))
	}
}

func resolveVideoURL(baseURL, videoURL string) string {
	videoURL = strings.TrimSpace(videoURL)
	if videoURL == "" {
		return ""
	}
	parsed, err := url.Parse(videoURL)
	if err != nil || parsed.IsAbs() || baseURL == "" {
		return videoURL
	}
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || !base.IsAbs() {
		return videoURL
	}
	return base.ResolveReference(parsed).String()
}

func resolveAdvancedCustomVideoContentTarget(channelModel *model.Channel, task *model.Task, videoURL string) (string, bool) {
	if channelModel == nil || task == nil {
		return videoURL, false
	}
	upstreamTaskID := strings.TrimSpace(task.GetUpstreamTaskID())
	if upstreamTaskID == "" {
		return videoURL, false
	}

	result, err := url.Parse(strings.TrimSpace(videoURL))
	if err != nil || result.Path != "/v1/videos/"+upstreamTaskID+"/content" {
		return videoURL, false
	}

	config := channelModel.GetOtherSettings().AdvancedCustom
	if config == nil {
		return videoURL, false
	}
	modelName := task.Properties.OriginModelName
	var route dto.AdvancedCustomRoute
	var found bool
	for _, path := range []string{"/v1/videos", "/v1/videos/generations", "/v1/video/generations"} {
		if route, found = config.MatchPathForModel(path, modelName); found {
			break
		}
	}
	if !found {
		return videoURL, false
	}

	routeTarget := strings.TrimSpace(route.UpstreamPath)
	var target *url.URL
	if strings.HasPrefix(routeTarget, "/") && !strings.HasPrefix(routeTarget, "//") {
		target, err = url.Parse(strings.TrimSpace(channelModel.GetBaseURL()))
		if err != nil || target.Scheme == "" || target.Host == "" {
			return videoURL, false
		}
		contentPath := deriveAdvancedCustomVideoContentPath(routeTarget, upstreamTaskID)
		target.Path = strings.TrimRight(target.Path, "/") + "/" + strings.TrimLeft(contentPath, "/")
	} else {
		target, err = url.Parse(routeTarget)
		if err != nil || target.Scheme == "" || target.Host == "" {
			return videoURL, false
		}
		target.Path = deriveAdvancedCustomVideoContentPath(target.Path, upstreamTaskID)
	}
	if !strings.EqualFold(target.Scheme, "http") && !strings.EqualFold(target.Scheme, "https") {
		return videoURL, false
	}
	target.RawPath = ""
	target.RawQuery = ""
	target.Fragment = ""
	return target.String(), true
}

func deriveAdvancedCustomVideoContentPath(submitPath, taskID string) string {
	path := strings.TrimRight(strings.TrimSpace(submitPath), "/")
	if strings.HasSuffix(path, "/generations") {
		path = strings.TrimSuffix(path, "/generations")
	}
	return path + "/" + url.PathEscape(taskID) + "/content"
}

func applyAdvancedCustomVideoAuth(channelModel *model.Channel, task *model.Task, req *http.Request, videoURL string) string {
	if channelModel == nil || task == nil || req == nil {
		return videoURL
	}
	config := channelModel.GetOtherSettings().AdvancedCustom
	if config == nil {
		req.Header.Set("Authorization", "Bearer "+channelModel.Key)
		return videoURL
	}
	modelName := task.Properties.OriginModelName
	var route dto.AdvancedCustomRoute
	var found bool
	for _, path := range []string{"/v1/videos", "/v1/videos/generations", "/v1/video/generations"} {
		if route, found = config.MatchPathForModel(path, modelName); found {
			break
		}
	}
	if !found {
		req.Header.Set("Authorization", "Bearer "+channelModel.Key)
		return videoURL
	}
	auth := route.Auth
	if auth == nil {
		req.Header.Set("Authorization", "Bearer "+channelModel.Key)
		return videoURL
	}
	value := strings.ReplaceAll(auth.Value, "{api_key}", channelModel.Key)
	value = strings.ReplaceAll(value, "{key}", channelModel.Key)
	switch strings.TrimSpace(auth.Type) {
	case dto.AdvancedCustomAuthTypeHeader:
		req.Header.Set(strings.TrimSpace(auth.Name), value)
	case dto.AdvancedCustomAuthTypeQuery:
		parsed, err := url.Parse(videoURL)
		if err == nil {
			query := parsed.Query()
			query.Set(strings.TrimSpace(auth.Name), value)
			parsed.RawQuery = query.Encode()
			videoURL = parsed.String()
		}
	}
	return videoURL
}

func writeVideoDataURL(c *gin.Context, dataURL string) error {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid data url")
	}

	header := parts[0]
	payload := parts[1]
	if !strings.HasPrefix(header, "data:") || !strings.Contains(header, ";base64") {
		return fmt.Errorf("unsupported data url")
	}

	mimeType := strings.TrimPrefix(header, "data:")
	mimeType = strings.TrimSuffix(mimeType, ";base64")
	if mimeType == "" {
		mimeType = "video/mp4"
	}

	videoBytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		videoBytes, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return err
		}
	}

	return writeVideoBytes(c, videoBytes, mimeType)
}

func writeVideoBytes(c *gin.Context, videoBytes []byte, mimeType string) error {
	if c == nil {
		return fmt.Errorf("context is nil")
	}
	if mimeType == "" {
		mimeType = "video/mp4"
	}
	c.Writer.Header().Set("Content-Type", mimeType)
	c.Writer.Header().Set("Accept-Ranges", "bytes")
	c.Writer.Header().Set("Cache-Control", "public, max-age=86400")

	start, end := 0, len(videoBytes)-1
	status := http.StatusOK
	if rangeHeader := strings.TrimSpace(c.GetHeader("Range")); rangeHeader != "" {
		var err error
		start, end, err = parseSingleVideoRange(rangeHeader, len(videoBytes))
		if err != nil {
			c.Writer.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", len(videoBytes)))
			c.Writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return nil
		}
		status = http.StatusPartialContent
		c.Writer.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(videoBytes)))
	}
	contentLength := end - start + 1
	if contentLength < 0 {
		contentLength = 0
	}
	c.Writer.Header().Set("Content-Length", strconv.Itoa(contentLength))
	c.Writer.WriteHeader(status)
	if c.Request.Method == http.MethodHead || contentLength == 0 {
		return nil
	}
	_, err := c.Writer.Write(videoBytes[start : end+1])
	return err
}

func parseSingleVideoRange(value string, size int) (int, int, error) {
	if size <= 0 || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "bytes=") {
		return 0, 0, fmt.Errorf("invalid range")
	}
	spec := strings.TrimSpace(strings.TrimSpace(value)[len("bytes="):])
	if strings.Contains(spec, ",") {
		return 0, 0, fmt.Errorf("multiple ranges are not supported")
	}
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range")
	}
	if strings.TrimSpace(parts[0]) == "" {
		suffix, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || suffix <= 0 {
			return 0, 0, fmt.Errorf("invalid suffix range")
		}
		if suffix > size {
			suffix = size
		}
		return size - suffix, size - 1, nil
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || start < 0 || start >= size {
		return 0, 0, fmt.Errorf("invalid range start")
	}
	end := size - 1
	if strings.TrimSpace(parts[1]) != "" {
		end, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || end < start {
			return 0, 0, fmt.Errorf("invalid range end")
		}
		if end >= size {
			end = size - 1
		}
	}
	return start, end, nil
}
