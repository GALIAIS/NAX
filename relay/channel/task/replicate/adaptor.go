package replicate

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

const ChannelName = "replicate-video"

type TaskAdaptor struct {
	taskcommon.BaseBilling
	baseURL            string
	version            string
	usePredictionRoute bool
}

type predictionResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	Output      any    `json:"output"`
	Error       any    `json:"error"`
	URLs        struct {
		Get    string `json:"get"`
		Cancel string `json:"cancel"`
	} `json:"urls"`
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.version = ""
	a.usePredictionRoute = false
	if info != nil {
		a.baseURL = strings.TrimRight(info.ChannelBaseUrl, "/")
	}
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *taskdto.TaskError {
	return relaycommon.ValidateMultipartDirect(c, info)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if a.baseURL == "" {
		return "", fmt.Errorf("replicate channel base URL is required")
	}
	modelName := strings.TrimSpace(info.UpstreamModelName)
	if modelName == "" {
		return "", fmt.Errorf("replicate model is required")
	}
	owner, modelSlug, version := parseModelReference(modelName)
	a.version = version
	a.usePredictionRoute = version != ""
	if owner != "" {
		if a.usePredictionRoute {
			return a.baseURL + "/v1/predictions", nil
		}
		return a.baseURL + "/v1/models/" + url.PathEscape(owner) + "/" + url.PathEscape(modelSlug) + "/predictions", nil
	}
	return a.baseURL + "/v1/predictions", nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	if uploaded, imageErr := relaycommon.MultipartImageData(c); imageErr != nil {
		return nil, imageErr
	} else if len(uploaded) > 0 {
		req.Images = append(uploaded, req.Images...)
	}
	// BuildRequestBody runs before DoRequest invokes BuildRequestURL. Parse the
	// model reference here as well so versioned models include the required
	// top-level `version` field in the same request that selects /v1/predictions.
	_, _, a.version = parseModelReference(strings.TrimSpace(info.UpstreamModelName))
	a.usePredictionRoute = a.version != ""
	input := map[string]any{"prompt": req.Prompt}
	if images := req.ImageList(); len(images) > 0 {
		input["image"] = images[0]
		if len(images) > 1 {
			input["images"] = images
		}
	}
	if seconds := req.RequestedDurationSeconds(); seconds > 0 {
		input["duration"] = seconds
	}
	if req.Size != "" {
		input["size"] = req.Size
	}
	if req.Resolution != "" {
		input["resolution"] = req.Resolution
	}
	if req.AspectRatio != "" {
		input["aspect_ratio"] = req.AspectRatio
	}
	if req.Quality != "" {
		input["quality"] = req.Quality
	}
	if req.Width > 0 {
		input["width"] = req.Width
	}
	if req.Height > 0 {
		input["height"] = req.Height
	}
	if req.FPS > 0 {
		input["fps"] = req.FPS
	}
	if req.N > 0 {
		input["num_outputs"] = req.N
	}
	if req.Seed != 0 {
		input["seed"] = req.Seed
	}
	for key, value := range req.Metadata {
		if isReservedInputKey(key) {
			continue
		}
		input[key] = value
	}
	body := map[string]any{"input": input}
	if a.version != "" {
		body["version"] = a.version
	}
	encoded, err := common.Marshal(body)
	if err != nil {
		return nil, err
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
	var parsed predictionResponse
	if err := common.Unmarshal(body, &parsed); err != nil || strings.TrimSpace(parsed.ID) == "" {
		return "", body, service.TaskErrorWrapper(fmt.Errorf("invalid Replicate prediction response"), "invalid_response", http.StatusBadGateway)
	}
	video := dto.NewOpenAIVideo()
	video.ID = info.PublicTaskID
	video.TaskID = info.PublicTaskID
	video.Model = info.OriginModelName
	video.CreatedAt = time.Now().Unix()
	c.JSON(http.StatusOK, video)
	return parsed.ID, body, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	uri := strings.TrimRight(baseURL, "/") + "/v1/predictions/" + url.PathEscape(taskID)
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func (a *TaskAdaptor) CancelTask(baseURL, key, taskID, proxy string) error {
	uri := strings.TrimRight(baseURL, "/") + "/v1/predictions/" + url.PathEscape(taskID) + "/cancel"
	req, err := http.NewRequest(http.MethodPost, uri, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
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
	var parsed predictionResponse
	if err := common.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	result := &relaycommon.TaskInfo{Code: 0}
	switch strings.ToLower(strings.TrimSpace(parsed.Status)) {
	case "starting", "processing", "running":
		result.Status = model.TaskStatusInProgress
	case "succeeded", "completed", "success":
		result.Status = model.TaskStatusSuccess
		result.Url, result.DurationSeconds, result.OutputCount = parseOutput(parsed.Output)
	case "failed", "canceled", "cancelled", "error":
		result.Status = model.TaskStatusFailure
		result.Reason = errorString(parsed.Error)
		if result.Reason == "" {
			result.Reason = "video generation failed"
		}
	default:
		return nil, fmt.Errorf("unknown Replicate prediction status: %s", parsed.Status)
	}
	applyPredictionTiming(result, parsed)
	if result.Status == model.TaskStatusSuccess && result.OutputCount == 0 {
		result.OutputCount = 1
	}
	return result, nil
}

func applyPredictionTiming(result *relaycommon.TaskInfo, prediction predictionResponse) {
	if result == nil {
		return
	}
	created := parseProviderTime(prediction.CreatedAt)
	started := parseProviderTime(prediction.StartedAt)
	completed := parseProviderTime(prediction.CompletedAt)
	if created.IsZero() {
		return
	}
	if !started.IsZero() && started.After(created) {
		result.QueueSeconds = started.Sub(created).Seconds()
	}
	if !completed.IsZero() && completed.After(created) {
		result.TotalSeconds = completed.Sub(created).Seconds()
		if !started.IsZero() && completed.After(started) {
			result.ProcessingSeconds = completed.Sub(started).Seconds()
		}
	}
}

func parseProviderTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999 -0700 MST", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if seconds > 1_000_000_000_000 {
			return time.UnixMilli(seconds)
		}
		return time.Unix(seconds, 0)
	}
	return time.Time{}
}

func (a *TaskAdaptor) GetModelList() []string { return nil }
func (a *TaskAdaptor) GetChannelName() string { return ChannelName }

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	seconds := req.RequestedDurationSeconds()
	if seconds <= 0 {
		seconds = 5
	}
	if seconds > relaycommon.MaxTaskDurationSeconds {
		seconds = relaycommon.MaxTaskDurationSeconds
	}
	ratio := map[string]float64{"seconds": seconds}
	if count := req.RequestedOutputCount(); count > 1 && count <= relaycommon.MaxTaskOutputCount {
		ratio["count"] = float64(count)
	}
	return ratio
}

func parseModelReference(modelName string) (owner, modelSlug, version string) {
	parts := strings.SplitN(strings.TrimSpace(modelName), "/", 2)
	if len(parts) != 2 {
		return "", "", ""
	}
	modelSlug = parts[1]
	if versionParts := strings.SplitN(modelSlug, ":", 2); len(versionParts) == 2 {
		modelSlug = strings.TrimSpace(versionParts[0])
		version = strings.TrimSpace(versionParts[1])
	}
	return strings.TrimSpace(parts[0]), modelSlug, version
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var parsed predictionResponse
	_ = common.Unmarshal(task.Data, &parsed)
	video := dto.NewOpenAIVideo()
	video.ID = task.TaskID
	video.TaskID = task.TaskID
	video.Model = task.Properties.OriginModelName
	video.Status = task.Status.ToVideoStatus()
	video.CreatedAt = task.CreatedAt
	video.SetProgressStr(task.Progress)
	if task.Status == model.TaskStatusSuccess || task.Status == model.TaskStatusFailure {
		video.CompletedAt = task.FinishTime
	}
	if urlValue, duration, _ := parseOutput(parsed.Output); urlValue != "" {
		video.SetMetadata("url", urlValue)
		if duration > 0 {
			video.Seconds = strconv.FormatFloat(duration, 'f', -1, 64)
		}
	}
	if task.Status == model.TaskStatusFailure {
		video.Error = &dto.OpenAIVideoError{Message: task.FailReason, Code: "upstream_error"}
	}
	return common.Marshal(video)
}

func parseOutput(output any) (string, float64, int) {
	if value, ok := output.(string); ok {
		return strings.TrimSpace(value), 0, 1
	}
	if values, ok := output.([]any); ok {
		for _, item := range values {
			if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value), 0, len(values)
			}
		}
		return "", 0, len(values)
	}
	if value, ok := output.(map[string]any); ok {
		urlValue, _ := value["url"].(string)
		if urlValue == "" {
			urlValue, _ = value["video_url"].(string)
		}
		var duration float64
		switch raw := value["duration"].(type) {
		case float64:
			duration = raw
		case string:
			duration, _ = strconv.ParseFloat(raw, 64)
		}
		return strings.TrimSpace(urlValue), duration, 1
	}
	return "", 0, 0
}

func errorString(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	if object, ok := value.(map[string]any); ok {
		for _, key := range []string{"message", "detail", "code"} {
			if text, ok := object[key].(string); ok && strings.TrimSpace(text) != "" {
				return strings.TrimSpace(text)
			}
		}
	}
	return ""
}

func isReservedInputKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "model", "prompt", "image", "images", "duration", "seconds", "size", "resolution", "aspect_ratio", "quality", "width", "height", "fps", "n", "seed", "metadata":
		return true
	default:
		return false
	}
}
