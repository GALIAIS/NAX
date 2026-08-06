package common

import "strings"

// CanonicalRelayRequestPath maps dashboard playground relay paths to the
// OpenAI-compatible /v1 path used by channel routing and provider adaptors.
// Playground endpoints intentionally live below /pg so browser sessions can
// bypass API-token quota accounting, but a channel's configured route remains
// expressed in its public /v1 form.
func CanonicalRelayRequestPath(path string) string {
	if path == "/pg" {
		return "/v1"
	}
	if strings.HasPrefix(path, "/pg/") {
		return "/v1" + strings.TrimPrefix(path, "/pg")
	}
	return path
}
