package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalRelayRequestPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "playground image", path: "/pg/images/generations", want: "/v1/images/generations"},
		{name: "playground video", path: "/pg/videos/task_123", want: "/v1/videos/task_123"},
		{name: "public endpoint", path: "/v1/chat/completions", want: "/v1/chat/completions"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, CanonicalRelayRequestPath(test.path))
		})
	}
}
