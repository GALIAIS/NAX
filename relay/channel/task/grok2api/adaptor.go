package grok2api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

const ChannelName = "grok2api"

// TaskAdaptor implements Grok2API's asynchronous video API while keeping the
// public OpenAI-compatible task contract exposed by New API.  Grok2API uses
// POST /v1/videos/generations and GET /v1/videos/{request_id}; this is why the
// normal advanced-custom request adaptor cannot handle video tasks by itself.
type TaskAdaptor struct {
	taskcommon.BaseBilling
	baseURL        string
	customConfig   *dto.AdvancedCustomConfig
	customRoute    dto.AdvancedCustomRoute
	useCustomRoute bool
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.customConfig = nil
	a.customRoute = dto.AdvancedCustomRoute{}
	a.useCustomRoute = false
	if info != nil {
		a.baseURL = strings.TrimRight(info.ChannelBaseUrl, "/")
		if info.ChannelMeta != nil && info.ChannelOtherSettings.AdvancedCustom != nil {
			a.customConfig = info.ChannelOtherSettings.AdvancedCustom
			if route, ok := findVideoRoute(a.customConfig, info.RequestURLPath, info.UpstreamModelName); ok {
				a.customRoute = route
				a.useCustomRoute = true
			}
		}
	}
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *taskdto.TaskError {
	if a.customConfig != nil {
		requestPath := common.CanonicalRelayRequestPath(c.Request.URL.Path)
		if route, ok := findVideoRoute(a.customConfig, requestPath, info.OriginModelName); ok {
			a.customRoute = route
			a.useCustomRoute = true
			if strings.TrimSpace(route.Converter) != "" && strings.TrimSpace(route.Converter) != "none" {
				return service.TaskErrorWrapperLocal(fmt.Errorf("advanced custom video routes only support native forwarding"), "unsupported_converter", http.StatusBadRequest)
			}
		} else {
			return service.TaskErrorWrapperLocal(
				fmt.Errorf("advanced custom channel does not support video request path %s for model %s", requestPath, info.OriginModelName),
				"unsupported_path", http.StatusBadRequest,
			)
		}
		return relaycommon.ValidateMultipartDirect(c, info)
	}
	if info != nil && info.OriginModelName != "" && !strings.HasPrefix(info.OriginModelName, "grok-imagine-video") {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("grok2api video adaptor does not support model %s", info.OriginModelName),
			"unsupported_model", http.StatusBadRequest,
		)
	}
	return relaycommon.ValidateMultipartDirect(c, info)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if a.useCustomRoute {
		modelName := ""
		if info != nil {
			modelName = info.UpstreamModelName
		}
		apiKey := ""
		if info != nil {
			apiKey = info.ApiKey
		}
		return a.buildCustomRouteURL(a.customRoute.UpstreamPath, modelName, "", apiKey)
	}
	if a.baseURL == "" {
		return "", fmt.Errorf("grok2api channel base URL is required")
	}
	return a.baseURL + "/v1/videos/generations", nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	if a.useCustomRoute {
		if err := a.setupCustomRouteHeader(req, info); err != nil {
			return err
		}
		if contentType := c.GetHeader("Content-Type"); contentType != "" {
			req.Header.Set("Content-Type", contentType)
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	if a.useCustomRoute && strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		return relaycommon.RewriteMultipartTaskModel(c, info.UpstreamModelName)
	}
	if a.useCustomRoute && !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return nil, fmt.Errorf("get request body failed: %w", err)
		}
		return common.ReaderOnly(storage), nil
	}
	var input map[string]any
	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return nil, fmt.Errorf("get request body failed: %w", err)
		}
		body, err := storage.Bytes()
		if err != nil {
			return nil, fmt.Errorf("read request body failed: %w", err)
		}
		if err := common.Unmarshal(body, &input); err != nil {
			return nil, fmt.Errorf("decode video request failed: %w", err)
		}
		if a.useCustomRoute {
			input["model"] = info.UpstreamModelName
			encoded, err := common.Marshal(input)
			if err != nil {
				return nil, fmt.Errorf("encode advanced custom video request failed: %w", err)
			}
			return bytes.NewReader(encoded), nil
		}
	} else {
		req, err := relaycommon.GetTaskRequest(c)
		if err != nil {
			return nil, err
		}
		uploaded, imageErr := relaycommon.MultipartImageData(c)
		if imageErr != nil {
			return nil, imageErr
		}
		if len(uploaded) > 0 {
			req.Images = append(uploaded, req.ImageList()...)
		}
		input = map[string]any{
			"prompt":            req.Prompt,
			"model":             req.Model,
			"size":              req.Size,
			"duration":          req.Duration,
			"seconds":           req.Seconds,
			"resolution":        req.Resolution,
			"aspect_ratio":      req.AspectRatio,
			"quality":           req.Quality,
			"width":             req.Width,
			"height":            req.Height,
			"fps":               req.FPS,
			"n":                 req.N,
			"seed":              req.Seed,
			"image":             req.Image,
			"input_reference":   req.InputReference,
			"reference_images":  req.ReferenceImages,
			"first_frame_image": req.FirstFrameImage,
			"last_frame_image":  req.LastFrameImage,
		}
		if len(req.Metadata) > 0 {
			input["metadata"] = req.Metadata
		}
	}

	body := make(map[string]any, 6)
	body["model"] = info.UpstreamModelName
	body["prompt"] = firstString(input, "prompt")

	if duration := firstInt(input, "duration", "seconds"); duration > 0 {
		body["duration"] = duration
	} else {
		body["duration"] = 6
	}

	aspectRatio := firstString(input, "aspect_ratio")
	resolution := firstString(input, "resolution")
	size := firstString(input, "size")
	if aspectRatio == "" {
		aspectRatio = aspectRatioFromSize(size)
	}
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	if resolution == "" {
		resolution = resolutionFromSize(size)
	}
	if resolution == "" {
		resolution = "720p"
	}
	body["aspect_ratio"] = aspectRatio
	body["resolution"] = resolution
	if quality := firstString(input, "quality"); quality != "" {
		body["quality"] = quality
	}
	if fps := firstInt(input, "fps"); fps > 0 {
		body["fps"] = fps
	}
	if seed, ok := firstInt64(input, "seed"); ok {
		body["seed"] = seed
	}
	if count := firstInt(input, "n"); count > 1 {
		body["n"] = count
	}

	if image := imageValue(input); image != nil {
		body["image"] = image
	}
	if refs := referenceImageValues(input["reference_images"]); len(refs) > 0 {
		body["reference_images"] = refs
	}
	for _, key := range []string{"first_frame_image", "last_frame_image"} {
		if value := firstString(input, key); value != "" {
			body[key] = value
		}
	}
	if images, ok := input["images"].([]string); ok && len(images) > 1 {
		body["last_frame_image"] = images[1]
	}
	// Keep provider-specific options instead of silently dropping fields that
	// are not part of the portable video DTO (camera motion, audio, watermark,
	// callback URL, width/height, and gateway-specific controls). Canonical
	// fields above remain normalized and cannot be overridden by this pass.
	for key, value := range input {
		switch key {
		case "model", "prompt", "duration", "seconds", "size", "resolution", "aspect_ratio", "quality", "fps", "n", "seed", "image", "images", "input_reference", "reference_images", "first_frame_image", "last_frame_image":
			continue
		}
		if value != nil {
			body[key] = value
		}
	}

	encoded, err := common.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode Grok2API video request failed: %w", err)
	}
	return bytes.NewReader(encoded), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *taskdto.TaskError) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	upstreamID := extractTaskID(body)
	if upstreamID == "" {
		return "", body, service.TaskErrorWrapper(fmt.Errorf("task id is empty in upstream response"), "invalid_response", http.StatusBadGateway)
	}

	video := dto.NewOpenAIVideo()
	video.ID = info.PublicTaskID
	video.TaskID = info.PublicTaskID
	video.Model = info.OriginModelName
	video.CreatedAt = time.Now().Unix()
	c.JSON(http.StatusOK, video)
	return upstreamID, body, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	modelName, _ := body["model"].(string)
	uri := ""
	if a.customConfig != nil {
		route, routeOK := findVideoRoute(a.customConfig, "", modelName)
		if !routeOK {
			return nil, fmt.Errorf("advanced custom video fetch route not found for model %s", modelName)
		}
		a.customRoute = route
		a.useCustomRoute = true
		fetchPath := strings.TrimSpace(route.FetchPath)
		if fetchPath == "" {
			fetchPath = deriveTaskPath(route.UpstreamPath, taskID)
		}
		var err error
		uri, err = a.buildCustomRouteURL(fetchPath, modelName, taskID, key)
		if err != nil {
			return nil, err
		}
	} else {
		uri = strings.TrimRight(baseURL, "/") + "/v1/videos/" + url.PathEscape(taskID)
	}
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	if a.useCustomRoute {
		if err := a.setupCustomRouteHeader(req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ApiKey: key}}); err != nil {
			return nil, err
		}
	} else {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) CancelTask(baseURL, key, taskID, proxy string) error {
	if strings.TrimSpace(taskID) == "" {
		return fmt.Errorf("invalid task_id")
	}
	uri := ""
	if a.customConfig != nil {
		route := a.customRoute
		routeOK := a.useCustomRoute && strings.TrimSpace(route.UpstreamPath) != ""
		if !routeOK {
			route, routeOK = findVideoRoute(a.customConfig, "", "")
		}
		if !routeOK {
			route, routeOK = findAnyVideoRoute(a.customConfig)
		}
		if !routeOK {
			return fmt.Errorf("advanced custom video cancel route not found")
		}
		a.customRoute = route
		a.useCustomRoute = true
		cancelPath := strings.TrimSpace(route.CancelPath)
		if cancelPath == "" {
			cancelPath = deriveTaskPath(route.UpstreamPath, taskID)
		}
		var err error
		uri, err = a.buildCustomRouteURL(cancelPath, "", taskID, key)
		if err != nil {
			return err
		}
	} else {
		uri = strings.TrimRight(baseURL, "/") + "/v1/videos/" + url.PathEscape(taskID)
	}
	req, err := http.NewRequest(http.MethodDelete, uri, nil)
	if err != nil {
		return err
	}
	if a.useCustomRoute {
		if err := a.setupCustomRouteHeader(req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ApiKey: key}}); err != nil {
			return err
		}
	} else {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("upstream cancel returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func (a *TaskAdaptor) ParseTaskResult(body []byte) (*relaycommon.TaskInfo, error) {
	parsed, err := parseGenericVideoPayload(body)
	if err != nil {
		return nil, fmt.Errorf("decode video status failed: %w", err)
	}

	result := &relaycommon.TaskInfo{Code: 0}
	switch strings.ToLower(parsed.Status) {
	case "queued", "pending":
		result.Status = model.TaskStatusQueued
	case "created", "submitted":
		result.Status = model.TaskStatusSubmitted
	case "processing", "in_progress", "running", "generating":
		result.Status = model.TaskStatusInProgress
	case "done", "completed", "success", "succeeded":
		result.Status = model.TaskStatusSuccess
		result.Url = parsed.URL
		result.DurationSeconds = parsed.Duration
		result.OutputCount = parsed.Count
	case "failed", "error", "cancelled", "canceled":
		result.Status = model.TaskStatusFailure
		result.Reason = parsed.Reason
		if result.Reason == "" {
			result.Reason = "video generation failed"
		}
	default:
		return nil, fmt.Errorf("unknown Grok2API video status: %s", parsed.Status)
	}
	if parsed.Progress > 0 {
		result.Progress = strconv.Itoa(parsed.Progress) + "%"
	}
	applyGenericTiming(result, parsed)
	return result, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{"grok-imagine-video"}
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

type genericVideoPayload struct {
	Status      string
	Progress    int
	URL         string
	Duration    float64
	Count       int
	Reason      string
	CreatedAt   float64
	StartedAt   float64
	CompletedAt float64
}

func parseGenericVideoPayload(body []byte) (genericVideoPayload, error) {
	var root map[string]any
	if err := common.Unmarshal(body, &root); err != nil {
		return genericVideoPayload{}, err
	}
	return parseVideoMap(root), nil
}

func parseVideoMap(root map[string]any) genericVideoPayload {
	result := genericVideoPayload{Count: 1}
	if root == nil {
		return result
	}
	result.Status = firstStringValue(root, "status", "state", "task_status")
	result.Progress = intValue(root, "progress", "percent", "percentage")
	result.URL = mediaURLValue(root, "url", "video_url", "output", "video")
	result.Duration = numberValue(root, "duration", "duration_seconds", "video_duration", "seconds")
	result.Count = intValue(root, "count", "output_count", "n", "sample_count")
	result.CreatedAt = timestampValue(root, "created_at", "createdAt", "submit_time", "submitted_at")
	result.StartedAt = timestampValue(root, "started_at", "startedAt", "processing_started_at")
	result.CompletedAt = timestampValue(root, "completed_at", "completedAt", "finished_at", "updated_at")
	if outputs := sliceValue(root, "outputs", "videos", "creations"); len(outputs) > 0 {
		result.Count = len(outputs)
		if result.URL == "" {
			for _, item := range outputs {
				if value, ok := item.(map[string]any); ok {
					if result.URL = mediaURLValue(value, "url", "video_url", "output", "video"); result.URL != "" {
						break
					}
				}
				if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
					result.URL = strings.TrimSpace(value)
					break
				}
			}
		}
	}
	if result.Count <= 0 {
		result.Count = 1
	}
	if errorValue, ok := root["error"]; ok {
		switch value := errorValue.(type) {
		case string:
			result.Reason = value
		case map[string]any:
			result.Reason = firstStringValue(value, "message", "detail", "code")
		}
	}
	if result.Reason == "" {
		result.Reason = firstStringValue(root, "message", "detail", "reason")
	}
	for _, nestedKey := range []string{"data", "response", "result", "prediction"} {
		nested, ok := root[nestedKey].(map[string]any)
		if !ok {
			continue
		}
		nestedResult := parseVideoMap(nested)
		if result.Status == "" {
			result.Status = nestedResult.Status
		}
		if result.Progress == 0 {
			result.Progress = nestedResult.Progress
		}
		if result.URL == "" {
			result.URL = nestedResult.URL
		}
		if result.Duration == 0 {
			result.Duration = nestedResult.Duration
		}
		if result.Count == 1 && nestedResult.Count > 1 {
			result.Count = nestedResult.Count
		}
		if result.Reason == "" {
			result.Reason = nestedResult.Reason
		}
		if result.CreatedAt == 0 {
			result.CreatedAt = nestedResult.CreatedAt
		}
		if result.StartedAt == 0 {
			result.StartedAt = nestedResult.StartedAt
		}
		if result.CompletedAt == 0 {
			result.CompletedAt = nestedResult.CompletedAt
		}
	}
	return result
}

func applyGenericTiming(result *relaycommon.TaskInfo, payload genericVideoPayload) {
	if result == nil || payload.CreatedAt <= 0 {
		return
	}
	if payload.StartedAt > payload.CreatedAt {
		result.QueueSeconds = payload.StartedAt - payload.CreatedAt
	}
	if payload.CompletedAt > payload.CreatedAt {
		result.TotalSeconds = payload.CompletedAt - payload.CreatedAt
		if payload.StartedAt > payload.CreatedAt && payload.CompletedAt > payload.StartedAt {
			result.ProcessingSeconds = payload.CompletedAt - payload.StartedAt
		}
	}
}

func extractTaskID(body []byte) string {
	var root map[string]any
	if common.Unmarshal(body, &root) != nil {
		return ""
	}
	return extractTaskIDFromMap(root)
}

func extractTaskIDFromMap(root map[string]any) string {
	if root == nil {
		return ""
	}
	for _, key := range []string{"request_id", "prediction_id", "task_id", "id"} {
		if value := stringValue(root[key]); value != "" {
			return value
		}
	}
	for _, key := range []string{"data", "response", "result", "prediction"} {
		if nested, ok := root[key].(map[string]any); ok {
			if value := extractTaskIDFromMap(nested); value != "" {
				return value
			}
		}
	}
	return ""
}

func firstStringValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	if value, ok := value.(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func numberValue(values map[string]any, keys ...string) float64 {
	for _, key := range keys {
		switch value := values[key].(type) {
		case float64:
			if value > 0 {
				return value
			}
		case int:
			if value > 0 {
				return float64(value)
			}
		case string:
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && parsed > 0 {
				return parsed
			}
		}
	}
	return 0
}

func timestampValue(values map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			if typed > 0 {
				if typed > 1_000_000_000_000 {
					return typed / 1000
				}
				return typed
			}
		case int:
			if typed > 0 {
				return float64(typed)
			}
		case string:
			text := strings.TrimSpace(typed)
			if parsed, err := strconv.ParseFloat(text, 64); err == nil && parsed > 0 {
				if parsed > 1_000_000_000_000 {
					return parsed / 1000
				}
				return parsed
			}
			for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
				if parsed, err := time.Parse(layout, text); err == nil {
					return float64(parsed.UnixNano()) / float64(time.Second)
				}
			}
		}
	}
	return 0
}

func intValue(values map[string]any, keys ...string) int {
	return int(numberValue(values, keys...))
}

func sliceValue(values map[string]any, keys ...string) []any {
	for _, key := range keys {
		if value, ok := values[key].([]any); ok {
			return value
		}
	}
	return nil
}

func mediaURLValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value := values[key]
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case map[string]any:
			if nested := firstStringValue(typed, "url", "download_url", "uri"); nested != "" {
				return nested
			}
		}
	}
	return ""
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	seconds := req.RequestedDurationSeconds()
	if seconds <= 0 {
		seconds = 6
	}
	if seconds > relaycommon.MaxTaskDurationSeconds {
		seconds = relaycommon.MaxTaskDurationSeconds
	}
	return map[string]float64{"seconds": seconds}
}

func findVideoRoute(config *dto.AdvancedCustomConfig, incomingPath, model string) (dto.AdvancedCustomRoute, bool) {
	if config == nil {
		return dto.AdvancedCustomRoute{}, false
	}
	paths := []string{}
	if strings.TrimSpace(incomingPath) != "" {
		path := strings.Split(strings.TrimSpace(incomingPath), "?")[0]
		paths = append(paths, common.CanonicalRelayRequestPath(path))
	}
	paths = append(paths, "/v1/videos", "/v1/videos/generations", "/v1/video/generations")
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		if route, ok := config.MatchPathForModel(path, model); ok {
			return route, true
		}
	}
	return dto.AdvancedCustomRoute{}, false
}

func findAnyVideoRoute(config *dto.AdvancedCustomConfig) (dto.AdvancedCustomRoute, bool) {
	if config == nil {
		return dto.AdvancedCustomRoute{}, false
	}
	for _, route := range config.Routes {
		path := strings.TrimSpace(route.IncomingPath)
		if path == "/v1/videos" || path == "/v1/videos/generations" || path == "/v1/video/generations" {
			return route, true
		}
	}
	return dto.AdvancedCustomRoute{}, false
}

func (a *TaskAdaptor) buildCustomRouteURL(upstreamPath, model, taskID, apiKey string) (string, error) {
	path := strings.TrimSpace(upstreamPath)
	if path == "" {
		return "", fmt.Errorf("advanced custom video upstream path is required")
	}
	path = strings.ReplaceAll(path, "{model}", url.PathEscape(strings.TrimSpace(model)))
	path = strings.ReplaceAll(path, "{task_id}", url.PathEscape(strings.TrimSpace(taskID)))
	path = strings.ReplaceAll(path, "{id}", url.PathEscape(strings.TrimSpace(taskID)))
	if strings.HasPrefix(path, "/") {
		if a.baseURL == "" {
			return "", fmt.Errorf("advanced custom video channel base URL is required")
		}
		base, err := url.Parse(a.baseURL)
		if err != nil || base.Scheme == "" || base.Host == "" {
			return "", fmt.Errorf("invalid advanced custom channel base URL")
		}
		parsedPath, err := url.Parse(path)
		if err != nil {
			return "", err
		}
		base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(parsedPath.Path, "/")
		base.RawPath = ""
		base.RawQuery = parsedPath.RawQuery
		base.Fragment = parsedPath.Fragment
		if a.customRoute.Auth != nil && strings.TrimSpace(a.customRoute.Auth.Type) == dto.AdvancedCustomAuthTypeQuery {
			query := base.Query()
			query.Set(strings.TrimSpace(a.customRoute.Auth.Name), applyAuthTemplate(a.customRoute.Auth.Value, apiKey))
			base.RawQuery = query.Encode()
		}
		return base.String(), nil
	}
	parsed, err := url.Parse(path)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("advanced custom video upstream path must be a full URL or an absolute path")
	}
	if a.customRoute.Auth != nil && strings.TrimSpace(a.customRoute.Auth.Type) == dto.AdvancedCustomAuthTypeQuery {
		query := parsed.Query()
		query.Set(strings.TrimSpace(a.customRoute.Auth.Name), applyAuthTemplate(a.customRoute.Auth.Value, apiKey))
		parsed.RawQuery = query.Encode()
	}
	return parsed.String(), nil
}

func (a *TaskAdaptor) setupCustomRouteHeader(req *http.Request, info *relaycommon.RelayInfo) error {
	if req == nil || info == nil {
		return fmt.Errorf("advanced custom video request context is missing")
	}
	auth := a.customRoute.Auth
	if auth == nil {
		req.Header.Set("Authorization", "Bearer "+info.ApiKey)
		return nil
	}
	switch strings.TrimSpace(auth.Type) {
	case "", dto.AdvancedCustomAuthTypeNone:
		return nil
	case dto.AdvancedCustomAuthTypeHeader:
		name := strings.TrimSpace(auth.Name)
		if name == "" {
			return fmt.Errorf("advanced custom video auth header name is required")
		}
		req.Header.Set(name, applyAuthTemplate(auth.Value, info.ApiKey))
		return nil
	case dto.AdvancedCustomAuthTypeQuery:
		return nil
	default:
		return fmt.Errorf("invalid advanced custom video auth type: %s", auth.Type)
	}
}

func applyAuthTemplate(value, key string) string {
	value = strings.ReplaceAll(value, "{api_key}", key)
	return strings.ReplaceAll(value, "{key}", key)
}

func deriveTaskPath(upstreamPath, taskID string) string {
	path := strings.TrimRight(strings.TrimSpace(upstreamPath), "/")
	if strings.Contains(path, "{task_id}") || strings.Contains(path, "{id}") {
		return strings.ReplaceAll(strings.ReplaceAll(path, "{task_id}", url.PathEscape(taskID)), "{id}", url.PathEscape(taskID))
	}
	if strings.HasSuffix(path, "/generations") {
		return strings.TrimSuffix(path, "/generations") + "/" + url.PathEscape(taskID)
	}
	if strings.HasSuffix(path, "/videos") {
		return path + "/" + url.PathEscape(taskID)
	}
	return path + "/" + url.PathEscape(taskID)
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	parsed, _ := parseGenericVideoPayload(task.Data)

	video := dto.NewOpenAIVideo()
	video.ID = task.TaskID
	video.TaskID = task.TaskID
	video.Model = task.Properties.OriginModelName
	video.CreatedAt = task.CreatedAt
	video.SetProgressStr(task.Progress)
	video.Status = task.Status.ToVideoStatus()
	if task.Status == model.TaskStatusSuccess || task.Status == model.TaskStatusFailure {
		video.CompletedAt = task.UpdatedAt
	}
	if parsed.URL != "" {
		video.SetMetadata("url", parsed.URL)
	}
	if parsed.Duration > 0 {
		video.Seconds = strconv.FormatFloat(parsed.Duration, 'f', -1, 64)
	}
	if task.Status == model.TaskStatusFailure {
		video.Error = &dto.OpenAIVideoError{Message: task.FailReason, Code: "upstream_error"}
	}
	return common.Marshal(video)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
		if metadata, ok := values["metadata"].(map[string]any); ok {
			if value, ok := metadata[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func firstInt(values map[string]any, keys ...string) int {
	for _, key := range keys {
		var value any
		if v, ok := values[key]; ok {
			value = v
		} else if metadata, ok := values["metadata"].(map[string]any); ok {
			value = metadata[key]
		}
		switch v := value.(type) {
		case int:
			if v > 0 {
				return v
			}
		case int64:
			if v > 0 {
				return int(v)
			}
		case float64:
			if v > 0 {
				return int(v)
			}
		case string:
			if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
				return parsed
			}
		}
	}
	return 0
}

func firstInt64(values map[string]any, keys ...string) (int64, bool) {
	for _, key := range keys {
		var value any
		if v, ok := values[key]; ok {
			value = v
		} else if metadata, ok := values["metadata"].(map[string]any); ok {
			value = metadata[key]
		}
		switch v := value.(type) {
		case int:
			return int64(v), true
		case int64:
			return v, true
		case float64:
			return int64(v), true
		case string:
			parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func imageValue(values map[string]any) any {
	if image, ok := values["image"]; ok {
		switch value := image.(type) {
		case map[string]any:
			if url, ok := value["url"].(string); ok && strings.TrimSpace(url) != "" {
				return map[string]any{"url": strings.TrimSpace(url)}
			}
		case string:
			if strings.TrimSpace(value) != "" {
				return map[string]any{"url": strings.TrimSpace(value)}
			}
		}
	}
	if reference, ok := values["input_reference"].(string); ok && strings.TrimSpace(reference) != "" {
		return map[string]any{"url": strings.TrimSpace(reference)}
	}
	for _, key := range []string{"first_frame_image", "last_frame_image"} {
		if reference, ok := values[key].(string); ok && strings.TrimSpace(reference) != "" {
			return map[string]any{"url": strings.TrimSpace(reference)}
		}
	}
	if images, ok := values["images"].([]string); ok && len(images) > 0 {
		if strings.TrimSpace(images[0]) != "" {
			return map[string]any{"url": strings.TrimSpace(images[0])}
		}
	}
	if images, ok := values["images"].([]any); ok && len(images) > 0 {
		if first, ok := images[0].(string); ok && strings.TrimSpace(first) != "" {
			return map[string]any{"url": strings.TrimSpace(first)}
		}
	}
	return nil
}

// referenceImageValues converts New API's provider-neutral string list to
// Grok2API's official [{"url":"..."}] video shape. Already-normalized
// objects remain supported for direct API clients.
func referenceImageValues(value any) []map[string]any {
	var raw []any
	switch values := value.(type) {
	case []string:
		raw = make([]any, 0, len(values))
		for _, item := range values {
			raw = append(raw, item)
		}
	case []any:
		raw = values
	default:
		return nil
	}

	result := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		switch typed := item.(type) {
		case string:
			if value := strings.TrimSpace(typed); value != "" {
				result = append(result, map[string]any{"url": value})
			}
		case map[string]any:
			if value, ok := typed["url"].(string); ok && strings.TrimSpace(value) != "" {
				result = append(result, map[string]any{"url": strings.TrimSpace(value)})
			}
		}
	}
	return result
}

func aspectRatioFromSize(size string) string {
	switch strings.TrimSpace(size) {
	case "720x1280", "1024x1792", "768x1365":
		return "9:16"
	case "1280x720", "1792x1024", "1365x768":
		return "16:9"
	default:
		return ""
	}
}

func resolutionFromSize(size string) string {
	size = strings.ToLower(strings.TrimSpace(size))
	if strings.HasSuffix(size, "p") && len(size) >= 2 {
		return size
	}
	return ""
}
