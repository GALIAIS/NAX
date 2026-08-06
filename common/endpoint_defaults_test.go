package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestDefaultEndpointInfoIncludesMediaGeneration(t *testing.T) {
	image, ok := GetDefaultEndpointInfo(constant.EndpointTypeImageGeneration)
	require.True(t, ok)
	require.Equal(t, EndpointInfo{Path: "/v1/images/generations", Method: "POST"}, image)

	video, ok := GetDefaultEndpointInfo(constant.EndpointTypeOpenAIVideo)
	require.True(t, ok)
	require.Equal(t, EndpointInfo{Path: "/v1/videos", Method: "POST"}, video)
}
