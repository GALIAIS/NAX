package service

import (
	"math"
	"time"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// BuildTaskTimingSnapshot captures the request-side duration before the task
// is persisted. It is intentionally independent of any provider protocol.
func BuildTaskTimingSnapshot(req relaycommon.TaskSubmitReq) *dto.TaskTiming {
	timing := &dto.TaskTiming{}
	if seconds := req.RequestedDurationSeconds(); seconds > 0 && seconds <= relaycommon.MaxTaskDurationSeconds {
		timing.RequestedDurationSeconds = seconds
	}
	if count := req.RequestedOutputCount(); count > 1 && count <= relaycommon.MaxTaskOutputCount {
		timing.RequestedOutputCount = count
	}
	return timing
}

// ApplyTaskTimingBillingRatios fills provider defaults that are only known
// after model mapping (for example a provider defaulting to five seconds).
// It never overwrites an explicit value parsed from the client's request.
func ApplyTaskTimingBillingRatios(timing *dto.TaskTiming, ratios map[string]float64) {
	if timing == nil || len(ratios) == 0 {
		return
	}
	if timing.RequestedDurationSeconds <= 0 {
		if seconds := ratios["seconds"]; seconds > 0 && seconds <= relaycommon.MaxTaskDurationSeconds {
			timing.RequestedDurationSeconds = seconds
		}
	}
	if timing.RequestedOutputCount <= 0 {
		if count := ratios["count"]; count >= 1 && count <= relaycommon.MaxTaskOutputCount {
			timing.RequestedOutputCount = int(count)
		}
	}
}

// UpdateTaskTiming updates lifecycle metrics after a polling response. Task
// timestamps are stored in seconds for database compatibility; the derived
// fields are therefore deterministic for SQLite, MySQL, and PostgreSQL.
func UpdateTaskTiming(task *model.Task, result *relaycommon.TaskInfo, now int64) {
	if task == nil {
		return
	}
	if now <= 0 {
		now = time.Now().Unix()
	}
	if task.PrivateData.Timing == nil {
		task.PrivateData.Timing = &dto.TaskTiming{}
	}
	timing := task.PrivateData.Timing
	if task.PrivateData.BillingContext != nil {
		ApplyTaskTimingBillingRatios(timing, task.PrivateData.BillingContext.OtherRatios)
	}
	timing.PollCount++
	timing.LastPolledAt = now

	if result != nil && result.DurationSeconds > 0 && result.DurationSeconds <= relaycommon.MaxTaskDurationSeconds {
		timing.ActualDurationSeconds = result.DurationSeconds
	}
	if result != nil && result.OutputCount > 0 && result.OutputCount <= relaycommon.MaxTaskOutputCount {
		timing.ActualOutputCount = result.OutputCount
	}
	if result != nil && result.QueueSeconds > 0 {
		timing.QueueSeconds = result.QueueSeconds
	}
	if result != nil && result.ProcessingSeconds > 0 {
		timing.ProcessingSeconds = result.ProcessingSeconds
	}
	if result != nil && result.TotalSeconds > 0 {
		timing.TotalSeconds = result.TotalSeconds
	}

	if task.Status == model.TaskStatusInProgress && task.StartTime == 0 {
		if task.SubmitTime > 0 {
			task.StartTime = task.SubmitTime
		} else {
			task.StartTime = now
		}
	}
	if task.Status == model.TaskStatusSuccess || task.Status == model.TaskStatusFailure {
		if task.StartTime == 0 {
			// A provider can return a terminal result on the first poll, without
			// ever exposing an explicit processing state. Mark processing as
			// starting at that observation so the measured queue time remains
			// visible instead of being silently reported as zero.
			task.StartTime = now
		}
		if task.FinishTime == 0 {
			task.FinishTime = now
		}
	}

	if task.SubmitTime > 0 && task.StartTime > 0 && timing.QueueSeconds <= 0 {
		timing.QueueSeconds = nonNegativeSeconds(task.StartTime - task.SubmitTime)
	}
	if task.StartTime > 0 && timing.ProcessingSeconds <= 0 {
		end := now
		if task.FinishTime > 0 {
			end = task.FinishTime
		}
		timing.ProcessingSeconds = nonNegativeSeconds(end - task.StartTime)
	}
	if task.SubmitTime > 0 && timing.TotalSeconds <= 0 {
		end := now
		if task.FinishTime > 0 {
			end = task.FinishTime
		}
		timing.TotalSeconds = nonNegativeSeconds(end - task.SubmitTime)
	}
}

func nonNegativeSeconds(value int64) float64 {
	if value <= 0 {
		return 0
	}
	seconds := float64(value)
	if math.IsInf(seconds, 0) || math.IsNaN(seconds) {
		return 0
	}
	return seconds
}
