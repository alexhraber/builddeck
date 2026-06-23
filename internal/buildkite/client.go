package buildkite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		BaseURL:    "https://api.buildkite.com/v2",
		Token:      token,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) get(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Message)
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func decode[T any](data []byte) ([]T, error) {
	var result []T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return result, nil
}

func (c *Client) ListOrganizations(ctx context.Context) ([]Organization, error) {
	data, err := c.get(ctx, "/organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("listing organizations: %w", err)
	}
	return decode[Organization](data)
}

func (c *Client) ListPipelines(ctx context.Context, orgSlug string) ([]Pipeline, error) {
	data, err := c.get(ctx, fmt.Sprintf("/organizations/%s/pipelines", orgSlug), map[string]string{
		"per_page": "100",
	})
	if err != nil {
		return nil, fmt.Errorf("listing pipelines for %s: %w", orgSlug, err)
	}
	return decode[Pipeline](data)
}

func (c *Client) ListBuilds(ctx context.Context, orgSlug, pipelineSlug string) ([]Build, error) {
	data, err := c.get(ctx, fmt.Sprintf("/organizations/%s/pipelines/%s/builds", orgSlug, pipelineSlug), map[string]string{
		"per_page": "25",
	})
	if err != nil {
		return nil, fmt.Errorf("listing builds for %s/%s: %w", orgSlug, pipelineSlug, err)
	}
	builds, err := decode[Build](data)
	if err != nil {
		return nil, err
	}
	for i := range builds {
		if len(builds[i].Jobs) == 0 {
			builds[i].Jobs = nil
		}
	}
	return builds, nil
}

func (c *Client) GetBuild(ctx context.Context, orgSlug, pipelineSlug string, buildNumber int) (*Build, error) {
	path := fmt.Sprintf("/organizations/%s/pipelines/%s/builds/%d", orgSlug, pipelineSlug, buildNumber)
	data, err := c.get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting build %s/%s#%d: %w", orgSlug, pipelineSlug, buildNumber, err)
	}
	var build Build
	if err := json.Unmarshal(data, &build); err != nil {
		return nil, fmt.Errorf("decoding build: %w", err)
	}
	return &build, nil
}
