package cache

import "canvaslms-gui/internal/api"

// LiveClient bypasses read caches for correctness-sensitive workflows such as
// grade uploads: each attempt must fetch the current, fully paginated roster.
func LiveClient(client api.CanvasClient) api.CanvasClient {
	if cached, ok := client.(*CachedCanvasClient); ok {
		return cached.real
	}
	return client
}
