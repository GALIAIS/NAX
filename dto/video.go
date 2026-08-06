package dto

// VideoRequest is the provider-neutral JSON shape used by the task routes.
// Provider-specific options remain available through Metadata, while these
// fields cover the common OpenAI/Kling/Veo/Wan/Hailuo conventions.
type VideoRequest struct {
	Model           string         `json:"model,omitempty" example:"kling-v1"`
	Prompt          string         `json:"prompt,omitempty" example:"宇航员站起身走了"`
	Image           string         `json:"image,omitempty"`
	Images          []string       `json:"images,omitempty"`
	InputReference  string         `json:"input_reference,omitempty"`
	ReferenceImages []string       `json:"reference_images,omitempty"`
	FirstFrameImage string         `json:"first_frame_image,omitempty"`
	LastFrameImage  string         `json:"last_frame_image,omitempty"`
	Duration        float64        `json:"duration,omitempty" example:"5.0"`
	Seconds         string         `json:"seconds,omitempty" example:"5"`
	Size            string         `json:"size,omitempty" example:"1280x720"`
	Resolution      string         `json:"resolution,omitempty" example:"720p"`
	AspectRatio     string         `json:"aspect_ratio,omitempty" example:"16:9"`
	Quality         string         `json:"quality,omitempty" example:"high"`
	Width           int            `json:"width,omitempty" example:"512"`
	Height          int            `json:"height,omitempty" example:"512"`
	Fps             int            `json:"fps,omitempty" example:"30"`
	Seed            int64          `json:"seed,omitempty" example:"20231234"`
	N               int            `json:"n,omitempty" example:"1"`
	ResponseFormat  string         `json:"response_format,omitempty" example:"url"`
	User            string         `json:"user,omitempty" example:"user-1234"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// VideoResponse is the provider-neutral submission response.
type VideoResponse struct {
	TaskId string `json:"task_id"`
	Status string `json:"status"`
}

// VideoTaskResponse is the provider-neutral task status response.
type VideoTaskResponse struct {
	TaskId   string              `json:"task_id" example:"abcd1234efgh"`
	Status   string              `json:"status" example:"succeeded"`
	Url      string              `json:"url,omitempty"`
	Format   string              `json:"format,omitempty" example:"mp4"`
	Metadata *VideoTaskMetadata  `json:"metadata,omitempty"`
	Timing   *TaskTiming         `json:"timing,omitempty"`
	Billing  *TaskBillingSummary `json:"billing,omitempty"`
	Error    *VideoTaskError     `json:"error,omitempty"`
}

// VideoTaskMetadata contains provider-reported output properties.
type VideoTaskMetadata struct {
	Duration float64 `json:"duration" example:"5.0"`
	Count    int     `json:"count,omitempty" example:"1"`
	Fps      int     `json:"fps" example:"30"`
	Width    int     `json:"width" example:"512"`
	Height   int     `json:"height" example:"512"`
	Seed     int     `json:"seed" example:"20231234"`
}

// VideoTaskError is the provider-neutral task error shape.
type VideoTaskError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
