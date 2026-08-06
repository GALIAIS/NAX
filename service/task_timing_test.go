package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAndUpdateTaskTimingSnapshot(t *testing.T) {
	req := relaycommon.TaskSubmitReq{DurationSeconds: 8, N: 2}
	timing := BuildTaskTimingSnapshot(req)
	require.NotNil(t, timing)
	assert.Equal(t, 8.0, timing.RequestedDurationSeconds)
	assert.Equal(t, 2, timing.RequestedOutputCount)

	task := &model.Task{
		SubmitTime: 100,
		Status:     model.TaskStatusInProgress,
		PrivateData: model.TaskPrivateData{
			Timing: timing,
		},
	}
	UpdateTaskTiming(task, &relaycommon.TaskInfo{
		DurationSeconds: 6,
		OutputCount:     1,
	}, 105)

	require.NotNil(t, task.PrivateData.Timing)
	assert.Equal(t, 1, task.PrivateData.Timing.PollCount)
	assert.Equal(t, 6.0, task.PrivateData.Timing.ActualDurationSeconds)
	assert.Equal(t, 1, task.PrivateData.Timing.ActualOutputCount)
	assert.Equal(t, 5.0, task.PrivateData.Timing.TotalSeconds)
	assert.Equal(t, 5.0, task.PrivateData.Timing.ProcessingSeconds)
}

func TestUpdateTaskTimingTerminalDerivesQueueAndFinishTime(t *testing.T) {
	task := &model.Task{
		SubmitTime: 100,
		StartTime:  103,
		Status:     model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			Timing: &dto.TaskTiming{},
		},
	}
	UpdateTaskTiming(task, nil, 110)

	assert.Equal(t, int64(110), task.FinishTime)
	assert.Equal(t, 3.0, task.PrivateData.Timing.QueueSeconds)
	assert.Equal(t, 7.0, task.PrivateData.Timing.ProcessingSeconds)
	assert.Equal(t, 10.0, task.PrivateData.Timing.TotalSeconds)
}

func TestUpdateTaskTimingTerminalWithoutProcessingPollKeepsQueueTime(t *testing.T) {
	task := &model.Task{
		SubmitTime: 100,
		Status:     model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			Timing: &dto.TaskTiming{},
		},
	}
	UpdateTaskTiming(task, nil, 110)

	assert.Equal(t, int64(110), task.StartTime)
	assert.Equal(t, 10.0, task.PrivateData.Timing.QueueSeconds)
	assert.Equal(t, 0.0, task.PrivateData.Timing.ProcessingSeconds)
	assert.Equal(t, 10.0, task.PrivateData.Timing.TotalSeconds)
}
