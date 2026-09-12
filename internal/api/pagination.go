package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// linkRE matches RFC 5988 Link header values like `<url>; rel="next"`.
var linkRE = regexp.MustCompile(`<([^>]+)>;\s*rel="([^"]+)"`)

// nextLink extracts the "next" URL from a Link header.
func nextLink(header string) string {
	matches := linkRE.FindAllStringSubmatch(header, -1)
	for _, m := range matches {
		if len(m) == 3 && m[2] == "next" {
			return m[1]
		}
	}
	return ""
}

// fetchAllPages performs a paginated GET and accumulates all results into target.
// target must be a pointer to a slice of the response type.
func (c *httpCanvasClient) fetchAllPages(ctx context.Context, endpoint string, target any) error {
	currentURL := c.baseURL + "/api/v1" + endpoint

	var allResults []json.RawMessage

	for currentURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, currentURL, nil)
		if err != nil {
			return err
		}

		resp, err := c.do(req)
		if err != nil {
			return err
		}

		var page []json.RawMessage
		if err := decodeJSON(resp, &page); err != nil {
			return err
		}

		allResults = append(allResults, page...)

		// Resolve next link (may be relative) against the current request URL.
		rawNext := nextLink(resp.Header.Get("Link"))
		if rawNext == "" {
			break
		}
		resolved, err := resolveURL(currentURL, rawNext)
		if err != nil {
			return err
		}
		currentURL = resolved
	}

	// Re-marshal and unmarshal into target type.
	merged, err := json.Marshal(allResults)
	if err != nil {
		return err
	}
	return json.Unmarshal(merged, target)
}

// resolveURL resolves a potentially relative URL against a base URL.
func resolveURL(base, ref string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(refURL).String(), nil
}

// paginatedGet performs a GET and returns the decoded response alongside
// a next-link callback. For endpoints that may return a list or dict.
func (c *httpCanvasClient) paginatedGet(ctx context.Context, endpoint string) (*http.Response, error) {
	url := c.baseURL + "/api/v1" + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// buildQuery builds a query string from key-value pairs.
func buildQuery(pairs ...string) string {
	if len(pairs)%2 != 0 {
		panic("buildQuery: odd number of arguments")
	}
	var sb strings.Builder
	for i := 0; i < len(pairs); i += 2 {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(pairs[i])
		sb.WriteByte('=')
		sb.WriteString(pairs[i+1])
	}
	return sb.String()
}
