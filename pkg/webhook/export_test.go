package webhook

import "net/http"

// SetHTTPClientForTest swaps the delivery client so tests can reach a local
// httptest server, which the SSRF-guarded client refuses. It returns a restore
// function.
func SetHTTPClientForTest(c *http.Client) func() {
	old := secureHTTPClient
	secureHTTPClient = c
	return func() { secureHTTPClient = old }
}
