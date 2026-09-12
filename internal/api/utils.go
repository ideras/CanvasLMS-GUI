package api

import (
	"encoding/json"
	"io"
	"net/http"
)

// readResponseBody reads the full response body and closes it.
func readResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// jsonUnmarshal is a thin wrapper around json.Unmarshal.
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// jsonDecodeReader decodes JSON from an io.Reader.
func jsonDecodeReader(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}
