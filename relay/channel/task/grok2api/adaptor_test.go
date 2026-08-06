package grok2api

import (
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/require"
)

func TestAdvancedCustomVideoRouteBuildsURLAndFetchPath(t *testing.T) {
	adaptor := &TaskAdaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ChannelType:    constant.ChannelTypeAdvancedCustom,
			ChannelBaseUrl: "https://provider.example",
			ApiKey:         "secret",
			ChannelOtherSettings: dto.ChannelOtherSettings{AdvancedCustom: &dto.AdvancedCustomConfig{
				Routes: []dto.AdvancedCustomRoute{{
					IncomingPath: "/v1/videos/generations",
					UpstreamPath: "/v1/videos/generations",
					FetchPath:    "/v1/videos/{task_id}",
					CancelPath:   "/v1/videos/{task_id}",
					Auth:         &dto.AdvancedCustomRouteAuth{Type: dto.AdvancedCustomAuthTypeQuery, Name: "key", Value: "{api_key}"},
				}},
			}},
		},
		OriginModelName: "grok-imagine-video",
	}
	info.RequestURLPath = "/v1/videos/generations"
	info.UpstreamModelName = "grok-imagine-video"
	adaptor.Init(info)

	requestURL, err := adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	parsed, err := url.Parse(requestURL)
	require.NoError(t, err)
	require.Equal(t, "/v1/videos/generations", parsed.Path)
	require.Equal(t, "secret", parsed.Query().Get("key"))

	fetchPath := deriveTaskPath(adaptor.customRoute.UpstreamPath, "task_123")
	require.Equal(t, "/v1/videos/task_123", fetchPath)
}

func TestParseGenericVideoPayloadSupportsNestedProviders(t *testing.T) {
	result, err := parseGenericVideoPayload([]byte(`{
		"data": {
			"status": "succeeded",
			"progress": 100,
			"videos": [{"url":"https://cdn.example/video.mp4"},{"url":"https://cdn.example/video-2.mp4"}],
			"duration_seconds": 7.5
		}
	}`))
	require.NoError(t, err)
	require.Equal(t, "succeeded", result.Status)
	require.Equal(t, 100, result.Progress)
	require.Equal(t, "https://cdn.example/video.mp4", result.URL)
	require.Equal(t, 7.5, result.Duration)
	require.Equal(t, 2, result.Count)
}

func TestExtractTaskIDSupportsCommonAsyncShapes(t *testing.T) {
	require.Equal(t, "req_123", extractTaskID([]byte(`{"data":{"request_id":"req_123"}}`)))
	require.Equal(t, "task_456", extractTaskID([]byte(`{"task_id":"task_456"}`)))
}

func TestParseGenericVideoPayloadIncludesProviderTiming(t *testing.T) {
	result, err := parseGenericVideoPayload([]byte(`{
		"status":"completed",
		"created_at":"2026-08-06T00:00:00Z",
		"started_at":"2026-08-06T00:00:03Z",
		"completed_at":"2026-08-06T00:00:11Z"
	}`))
	require.NoError(t, err)
	taskResult, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"status":"completed",
		"created_at":"2026-08-06T00:00:00Z",
		"started_at":"2026-08-06T00:00:03Z",
		"completed_at":"2026-08-06T00:00:11Z"
	}`))
	require.NoError(t, err)
	require.Equal(t, 3.0, taskResult.QueueSeconds)
	require.Equal(t, 8.0, taskResult.ProcessingSeconds)
	require.Equal(t, 11.0, taskResult.TotalSeconds)
	require.Equal(t, result.CompletedAt-result.CreatedAt, taskResult.TotalSeconds)
}
