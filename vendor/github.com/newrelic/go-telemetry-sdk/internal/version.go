package internal

import "net/http"

const (
	major = "0"
	minor = "1"
	patch = "0"

	// Version is the full string version of this SDK.
	Version = major + "." + minor + "." + patch
)

// AddUserAgentHeader adds a User-Agent header with the SDK's version.
func AddUserAgentHeader(h http.Header) {
	h.Add("User-Agent", "NewRelic-Go-TelemetrySDK/"+Version)
}
