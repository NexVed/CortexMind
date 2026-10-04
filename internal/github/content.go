package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type APIError struct{ Status int }

func (e *APIError) Error() string { return fmt.Sprintf("GitHub API returned HTTP %d", e.Status) }

func EscapePath(value string) string {
	parts := strings.Split(value, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

// RepositoryResource always requests api.github.com; credentials never go to a supplied URL.
func (c Client) RepositoryResource(ctx context.Context, token, fullName, suffix, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+"/repos/"+EscapePath(fullName)+suffix, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	res, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return nil, &APIError{Status: res.StatusCode}
	}
	const limit = 4 << 20
	body, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if len(body) > limit {
		return nil, fmt.Errorf("repository response exceeds 4 MiB")
	}
	return body, err
}
