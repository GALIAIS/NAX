package common

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

const maxMultipartVideoImageBytes = 20 * 1024 * 1024

// MultipartImageData returns uploaded image/frame fields as data URIs in
// request order. It lets JSON-only providers (Veo, Jimeng, and similar
// APIs) consume the same multipart inputs accepted by OpenAI's video API.
// The helper is intentionally bounded to prevent an image upload from being
// expanded without limit while converting it to base64.
func MultipartImageData(c *gin.Context, fields ...string) ([]string, error) {
	if c == nil {
		return nil, nil
	}
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		return nil, nil
	}
	form, err := c.MultipartForm()
	if err != nil || form == nil {
		return nil, err
	}
	if len(fields) == 0 {
		fields = []string{"input_reference", "image", "images", "first_frame_image", "last_frame_image", "reference_images"}
	}
	var result []string
	for _, field := range fields {
		for _, header := range form.File[field] {
			if header.Size > maxMultipartVideoImageBytes {
				return nil, fmt.Errorf("multipart image %s exceeds %d bytes", header.Filename, maxMultipartVideoImageBytes)
			}
			file, openErr := header.Open()
			if openErr != nil {
				return nil, openErr
			}
			data, readErr := io.ReadAll(io.LimitReader(file, maxMultipartVideoImageBytes+1))
			_ = file.Close()
			if readErr != nil {
				return nil, readErr
			}
			if len(data) > maxMultipartVideoImageBytes {
				return nil, fmt.Errorf("multipart image %s exceeds %d bytes", header.Filename, maxMultipartVideoImageBytes)
			}
			mimeType := header.Header.Get("Content-Type")
			if mimeType == "" || mimeType == "application/octet-stream" {
				mimeType = http.DetectContentType(data)
			}
			result = append(result, "data:"+mimeType+";base64,"+base64.StdEncoding.EncodeToString(data))
		}
	}
	return result, nil
}

// RewriteMultipartTaskModel rebuilds a multipart video request while
// replacing its model form field. It preserves all scalar fields and uploaded
// files, which is required when an advanced-custom/Replicate route maps the
// client model to a different upstream model.
func RewriteMultipartTaskModel(c *gin.Context, model string) (io.Reader, error) {
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	if err := writer.WriteField("model", model); err != nil {
		return nil, err
	}
	for key, values := range form.Value {
		if key == "model" {
			continue
		}
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				return nil, err
			}
		}
	}
	for field, files := range form.File {
		for _, header := range files {
			file, err := header.Open()
			if err != nil {
				return nil, err
			}
			contentType := header.Header.Get("Content-Type")
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			partHeader := make(textproto.MIMEHeader)
			partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, field, header.Filename))
			partHeader.Set("Content-Type", contentType)
			part, createErr := writer.CreatePart(partHeader)
			if createErr == nil {
				_, createErr = io.Copy(part, file)
			}
			_ = file.Close()
			if createErr != nil {
				return nil, createErr
			}
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return &buffer, nil
}

type HasPrompt interface {
	GetPrompt() string
}

type HasImage interface {
	HasImage() bool
}

func GetFullRequestURL(baseURL string, requestURL string, channelType int) string {
	fullRequestURL := fmt.Sprintf("%s%s", baseURL, requestURL)

	if strings.HasPrefix(baseURL, "https://gateway.ai.cloudflare.com") {
		switch channelType {
		case constant.ChannelTypeOpenAI:
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/v1"))
		case constant.ChannelTypeAzure:
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/openai/deployments"))
		}
	}
	return fullRequestURL
}

func SanitizeURLForLog(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	query := parsedURL.Query()
	if len(query) == 0 {
		return rawURL
	}

	changed := false
	for key := range query {
		if isSensitiveURLQueryKey(key) {
			query.Set(key, "***masked***")
			changed = true
		}
	}
	if !changed {
		return rawURL
	}

	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

func isSensitiveURLQueryKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	switch normalized {
	case "key",
		"api_key",
		"api-key",
		"apikey",
		"x-api-key",
		"access_token",
		"refresh_token",
		"id_token",
		"token",
		"authorization",
		"auth",
		"client_secret",
		"secret",
		"password",
		"passwd",
		"signature",
		"sig",
		"awsaccesskeyid",
		"x-amz-credential",
		"x-amz-security-token",
		"x-amz-signature":
		return true
	}
	return strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "signature")
}

func GetAPIVersion(c *gin.Context) string {
	query := c.Request.URL.Query()
	apiVersion := query.Get("api-version")
	if apiVersion == "" {
		apiVersion = c.GetString("api_version")
	}
	return apiVersion
}

func createTaskError(err error, code string, statusCode int, localError bool) *dto.TaskError {
	return &dto.TaskError{
		Code:       code,
		Message:    err.Error(),
		StatusCode: statusCode,
		LocalError: localError,
		Error:      err,
	}
}

func storeTaskRequest(c *gin.Context, info *RelayInfo, action string, requestObj TaskSubmitReq) {
	info.Action = action
	c.Set("task_request", requestObj)
}
func GetTaskRequest(c *gin.Context) (TaskSubmitReq, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return TaskSubmitReq{}, fmt.Errorf("request not found in context")
	}
	req, ok := v.(TaskSubmitReq)
	if !ok {
		return TaskSubmitReq{}, fmt.Errorf("invalid task request type")
	}
	return req, nil
}

func validatePrompt(prompt string) *dto.TaskError {
	if strings.TrimSpace(prompt) == "" {
		return createTaskError(fmt.Errorf("prompt is required"), "invalid_request", http.StatusBadRequest, true)
	}
	return nil
}

// MaxTaskDurationSeconds caps user-supplied video duration. Duration is used
// as a billing multiplier (OtherRatio "seconds"); an unbounded value could
// overflow quota calculation into a negative charge.
const MaxTaskDurationSeconds = 3600

func validateTaskDurationBounds(req TaskSubmitReq) *dto.TaskError {
	if strings.TrimSpace(req.Seconds) != "" {
		if _, ok := parseTaskNumber([]byte(req.Seconds)); !ok {
			return createTaskError(fmt.Errorf("seconds must be a number"), "invalid_seconds", http.StatusBadRequest, true)
		}
	}
	seconds := req.RequestedDurationSeconds()
	if seconds < 0 || seconds > MaxTaskDurationSeconds {
		return createTaskError(fmt.Errorf("seconds must be between 0 and %d", MaxTaskDurationSeconds), "invalid_seconds", http.StatusBadRequest, true)
	}
	requestedCount := req.RequestedOutputCount()
	if req.N < 0 || requestedCount > MaxTaskOutputCount {
		return createTaskError(fmt.Errorf("n must be between 0 and %d", MaxTaskOutputCount), "invalid_n", http.StatusBadRequest, true)
	}
	if req.FPS < 0 || req.FPS > MaxTaskFPS {
		return createTaskError(fmt.Errorf("fps must be between 0 and %d", MaxTaskFPS), "invalid_fps", http.StatusBadRequest, true)
	}
	if req.Width < 0 || req.Width > MaxTaskDimension {
		return createTaskError(fmt.Errorf("width must be between 0 and %d", MaxTaskDimension), "invalid_width", http.StatusBadRequest, true)
	}
	if req.Height < 0 || req.Height > MaxTaskDimension {
		return createTaskError(fmt.Errorf("height must be between 0 and %d", MaxTaskDimension), "invalid_height", http.StatusBadRequest, true)
	}
	return nil
}

func validateMultipartTaskRequest(c *gin.Context, info *RelayInfo, action string) (TaskSubmitReq, error) {
	var req TaskSubmitReq
	if _, err := c.MultipartForm(); err != nil {
		return req, err
	}

	formData := c.Request.PostForm
	req = TaskSubmitReq{
		Prompt:          formData.Get("prompt"),
		Model:           formData.Get("model"),
		Mode:            formData.Get("mode"),
		Image:           formData.Get("image"),
		Size:            formData.Get("size"),
		Resolution:      formData.Get("resolution"),
		AspectRatio:     formData.Get("aspect_ratio"),
		Quality:         formData.Get("quality"),
		Width:           parseMultipartInt(formData.Get("width"), MaxTaskDimension),
		Height:          parseMultipartInt(formData.Get("height"), MaxTaskDimension),
		ResponseFormat:  formData.Get("response_format"),
		User:            formData.Get("user"),
		InputReference:  formData.Get("input_reference"),
		FirstFrameImage: formData.Get("first_frame_image"),
		LastFrameImage:  formData.Get("last_frame_image"),
		ReferenceImages: append([]string(nil), formData["reference_images"]...),
		Metadata:        make(map[string]interface{}),
	}

	if durationStr := formData.Get("seconds"); durationStr != "" {
		req.Seconds = durationStr
	}
	if durationStr := formData.Get("duration"); durationStr != "" {
		if duration, err := strconv.ParseFloat(durationStr, 64); err == nil {
			req.DurationSeconds = duration
			if req.Seconds == "" {
				req.Seconds = strconv.FormatFloat(duration, 'f', -1, 64)
			}
			if duration > 0 && duration <= MaxTaskDurationSeconds {
				req.Duration = int(math.Ceil(duration))
			}
		} else {
			req.DurationSeconds = -1
		}
	}
	if countStr := formData.Get("n"); countStr != "" {
		if count, err := strconv.Atoi(countStr); err == nil {
			req.N = count
		} else {
			req.N = MaxTaskOutputCount + 1
		}
	}
	if fpsStr := formData.Get("fps"); fpsStr != "" {
		if fps, err := strconv.Atoi(fpsStr); err == nil {
			req.FPS = fps
		} else {
			req.FPS = MaxTaskFPS + 1
		}
	}
	if seedStr := formData.Get("seed"); seedStr != "" {
		if seed, err := strconv.ParseInt(seedStr, 10, 64); err == nil {
			req.Seed = seed
		}
	}

	if images := formData["images"]; len(images) > 0 {
		req.Images = append([]string(nil), images...)
	}

	for key, values := range formData {
		if len(values) > 0 && !isKnownTaskField(key) {
			if key == "metadata" {
				var metadata map[string]interface{}
				if err := common.Unmarshal([]byte(values[0]), &metadata); err == nil {
					for metadataKey, metadataValue := range metadata {
						req.Metadata[metadataKey] = metadataValue
					}
				}
				continue
			}
			if intVal, err := strconv.Atoi(values[0]); err == nil {
				req.Metadata[key] = intVal
			} else if floatVal, err := strconv.ParseFloat(values[0], 64); err == nil {
				req.Metadata[key] = floatVal
			} else {
				req.Metadata[key] = values[0]
			}
		}
	}
	return req, nil
}

func parseMultipartInt(value string, max int) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return max + 1
	}
	if parsed < 0 {
		return -1
	}
	if parsed > max {
		return max + 1
	}
	return parsed
}

func ValidateMultipartDirect(c *gin.Context, info *RelayInfo) *dto.TaskError {
	var prompt string
	var model string
	var seconds int
	var size string
	var hasInputReference bool

	var req TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return createTaskError(err, "invalid_json", http.StatusBadRequest, true)
	}

	prompt = req.Prompt
	model = req.Model
	size = req.Size
	seconds = int(req.RequestedDurationSeconds())
	if req.InputReference != "" {
		req.Images = []string{req.InputReference}
	} else if len(req.Images) == 0 && strings.TrimSpace(req.Image) != "" {
		// 兼容单图上传
		req.Images = []string{strings.TrimSpace(req.Image)}
	}

	if strings.TrimSpace(req.Model) == "" {
		return createTaskError(fmt.Errorf("model field is required"), "missing_model", http.StatusBadRequest, true)
	}

	if req.HasImage() || multipartInputReferenceExists(c) {
		hasInputReference = true
	}

	if taskErr := validatePrompt(prompt); taskErr != nil {
		return taskErr
	}

	if taskErr := validateTaskDurationBounds(req); taskErr != nil {
		return taskErr
	}

	action := constant.TaskActionTextGenerate
	if hasInputReference {
		action = constant.TaskActionGenerate
	}
	if strings.HasPrefix(model, "sora-2") {

		if size == "" {
			size = "720x1280"
		}

		if seconds <= 0 {
			seconds = 4
		}

		if model == "sora-2" && !lo.Contains([]string{"720x1280", "1280x720"}, size) {
			return createTaskError(fmt.Errorf("sora-2 size is invalid"), "invalid_size", http.StatusBadRequest, true)
		}
		if model == "sora-2-pro" && !lo.Contains([]string{"720x1280", "1280x720", "1792x1024", "1024x1792"}, size) {
			return createTaskError(fmt.Errorf("sora-2 size is invalid"), "invalid_size", http.StatusBadRequest, true)
		}
		// OtherRatios 已移到 Sora adaptor 的 EstimateBilling 中设置
	}

	storeTaskRequest(c, info, action, req)

	return nil
}

func isKnownTaskField(field string) bool {
	knownFields := map[string]bool{
		"prompt":            true,
		"model":             true,
		"mode":              true,
		"image":             true,
		"images":            true,
		"size":              true,
		"duration":          true,
		"seconds":           true,
		"resolution":        true,
		"aspect_ratio":      true,
		"quality":           true,
		"width":             true,
		"height":            true,
		"fps":               true,
		"n":                 true,
		"seed":              true,
		"response_format":   true,
		"user":              true,
		"reference_images":  true,
		"first_frame_image": true,
		"last_frame_image":  true,
		"input_reference":   true, // Sora 特有字段
	}
	return knownFields[field]
}

func multipartInputReferenceExists(c *gin.Context) bool {
	if c == nil || !strings.Contains(c.GetHeader("Content-Type"), "multipart/form-data") {
		return false
	}
	form, err := c.MultipartForm()
	if err != nil || form == nil {
		return false
	}
	for _, field := range []string{"input_reference", "image", "images", "first_frame_image", "last_frame_image"} {
		if len(form.File[field]) > 0 {
			return true
		}
	}
	return false
}

// TaskRequestBillingRatios adds safe, protocol-independent multipliers for
// explicit duration and batch-count parameters. Provider adaptors remain free
// to override these keys with their published resolution/input pricing.
func TaskRequestBillingRatios(req TaskSubmitReq) map[string]float64 {
	ratios := make(map[string]float64)
	if seconds := req.RequestedDurationSeconds(); seconds > 0 && seconds <= MaxTaskDurationSeconds {
		ratios["seconds"] = seconds
	}
	if count := req.RequestedOutputCount(); count > 1 && count <= MaxTaskOutputCount {
		ratios["count"] = float64(count)
	}
	if len(ratios) == 0 {
		return nil
	}
	return ratios
}

func ValidateBasicTaskRequest(c *gin.Context, info *RelayInfo, action string) *dto.TaskError {
	var err error
	contentType := c.GetHeader("Content-Type")
	var req TaskSubmitReq
	if strings.HasPrefix(contentType, "multipart/form-data") {
		req, err = validateMultipartTaskRequest(c, info, action)
		if err != nil {
			return createTaskError(err, "invalid_multipart_form", http.StatusBadRequest, true)
		}
	}
	// 为了metadata字段的兼容性，统一UnmarshalBodyReusable
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return createTaskError(err, "invalid_request", http.StatusBadRequest, true)
	}

	if taskErr := validatePrompt(req.Prompt); taskErr != nil {
		return taskErr
	}

	if taskErr := validateTaskDurationBounds(req); taskErr != nil {
		return taskErr
	}

	if len(req.Images) == 0 && strings.TrimSpace(req.Image) != "" {
		// 兼容单图上传
		req.Images = []string{req.Image}
	}

	storeTaskRequest(c, info, action, req)
	return nil
}
