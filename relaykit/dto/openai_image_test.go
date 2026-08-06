package dto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImageRequestPreservesGrokImageControls(t *testing.T) {
	var request ImageRequest
	require.NoError(t, request.UnmarshalJSON([]byte(`{
		"model":"grok-imagine-image-quality",
		"prompt":"a city at night",
		"n":2,
		"aspect_ratio":"9:16",
		"resolution":"2k",
		"response_format":"url",
		"stream":false,
		"group":"default"
	}`)))

	require.Equal(t, "9:16", request.AspectRatio)
	require.Equal(t, "2k", request.Resolution)

	payload, err := request.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"model":"grok-imagine-image-quality",
		"prompt":"a city at night",
		"n":2,
		"aspect_ratio":"9:16",
		"resolution":"2k",
		"response_format":"url",
		"stream":false
	}`, string(payload))
}
