package dto

import (
	"encoding/json"
)

type TaskError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Data       any    `json:"data"`
	StatusCode int    `json:"-"`
	LocalError bool   `json:"-"`
	Error      error  `json:"-"`
}

type TaskData interface {
	SunoDataResponse | []SunoDataResponse | string | any
}

const TaskSuccessCode = "success"

type TaskResponse[T TaskData] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func (t *TaskResponse[T]) IsSuccess() bool {
	return t.Code == TaskSuccessCode
}

type TaskDto struct {
	ID         int64               `json:"id"`
	CreatedAt  int64               `json:"created_at"`
	UpdatedAt  int64               `json:"updated_at"`
	TaskID     string              `json:"task_id"`
	Platform   string              `json:"platform"`
	UserId     int                 `json:"user_id"`
	Group      string              `json:"group"`
	ChannelId  int                 `json:"channel_id"`
	Quota      int                 `json:"quota"`
	Billing    *TaskBillingSummary `json:"billing,omitempty"`
	Action     string              `json:"action"`
	Status     string              `json:"status"`
	FailReason string              `json:"fail_reason"`
	ResultURL  string              `json:"result_url,omitempty"` // 任务结果 URL（视频地址等）
	SubmitTime int64               `json:"submit_time"`
	StartTime  int64               `json:"start_time"`
	FinishTime int64               `json:"finish_time"`
	Progress   string              `json:"progress"`
	Timing     *TaskTiming         `json:"timing,omitempty"`
	Properties any                 `json:"properties"`
	Username   string              `json:"username,omitempty"`
	Data       json.RawMessage     `json:"data"`
}

// TaskTiming contains provider-independent timing information for an
// asynchronous task. All values are seconds and are derived from the task
// lifecycle timestamps, so older rows remain compatible without a migration.
type TaskTiming struct {
	RequestedDurationSeconds float64 `json:"requested_duration_seconds,omitempty"`
	ActualDurationSeconds    float64 `json:"actual_duration_seconds,omitempty"`
	RequestedOutputCount     int     `json:"requested_output_count,omitempty"`
	ActualOutputCount        int     `json:"actual_output_count,omitempty"`
	QueueSeconds             float64 `json:"queue_seconds,omitempty"`
	ProcessingSeconds        float64 `json:"processing_seconds,omitempty"`
	TotalSeconds             float64 `json:"total_seconds,omitempty"`
	PollCount                int     `json:"poll_count,omitempty"`
	LastPolledAt             int64   `json:"last_polled_at,omitempty"`
}

// TaskBillingSummary exposes the non-sensitive billing snapshot and the
// currently settled amount. It intentionally contains no provider key or
// token data and is safe for both user and administrator task views.
type TaskBillingSummary struct {
	Mode             string             `json:"mode,omitempty"`
	PreConsumedQuota int                `json:"pre_consumed_quota,omitempty"`
	ActualQuota      int                `json:"actual_quota,omitempty"`
	ModelPrice       float64            `json:"model_price,omitempty"`
	ModelRatio       float64            `json:"model_ratio,omitempty"`
	GroupRatio       float64            `json:"group_ratio,omitempty"`
	OtherRatios      map[string]float64 `json:"other_ratios,omitempty"`
	Settled          bool               `json:"settled,omitempty"`
}

type FetchReq struct {
	IDs []string `json:"ids"`
}
