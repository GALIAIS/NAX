package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// CancelVideoTask implements the OpenAI-compatible DELETE video operation.
// Providers expose different cancellation APIs, so cancellation is first
// applied atomically to the gateway task; the polling worker then ignores the
// terminal task and the reserved quota is refunded exactly once.
func CancelVideoTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", "task_id is required")
		return
	}
	task, exists, err := model.GetByTaskId(c.GetInt("id"), taskID)
	if err != nil {
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to query task")
		return
	}
	if !exists || task == nil {
		videoProxyError(c, http.StatusNotFound, "invalid_request_error", "Task not found")
		return
	}
	if task.Status == model.TaskStatusSuccess || task.Status == model.TaskStatusFailure {
		videoProxyError(c, http.StatusConflict, "invalid_request_error", "Task is already in a terminal state")
		return
	}
	previousStatus := task.Status
	task.Status = model.TaskStatusFailure
	task.Progress = "100%"
	task.FinishTime = time.Now().Unix()
	task.FailReason = "task cancelled by user"
	service.UpdateTaskTiming(task, nil, task.FinishTime)
	won, err := task.UpdateWithStatus(previousStatus)
	if err != nil {
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to cancel task")
		return
	}
	if !won {
		videoProxyError(c, http.StatusConflict, "invalid_request_error", "Task changed while cancelling")
		return
	}
	// The local CAS is deliberately performed before the provider call. If a
	// poller completed the task concurrently, we must not send a cancellation
	// request for a video that has already succeeded upstream. Provider cancel
	// endpoints are best-effort; the local terminal state/refund remains the
	// source of truth for the gateway.
	if err := relay.CancelUpstreamVideoTask(task); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("upstream cancellation failed for task %s: %v", taskID, err))
	}
	if task.Quota != 0 {
		service.RefundTaskQuota(c.Request.Context(), task, fmt.Sprintf("%s: %s", task.FailReason, taskID))
	}

	video := task.ToOpenAIVideo()
	video.Error = &dto.OpenAIVideoError{Message: task.FailReason, Code: "cancelled"}
	c.JSON(http.StatusOK, video)
}
