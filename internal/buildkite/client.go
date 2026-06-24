package buildkite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

type apiResponse struct {
	Body     []byte
	Headers  http.Header
	NextPage int
}

func (c *Client) get(ctx context.Context, path string, params map[string]string) (*apiResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", path, err)
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
		return nil, fmt.Errorf("request %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", path, err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("API %s (status %d): %s", path, resp.StatusCode, errResp.Message)
		}
		bodyStr := string(body)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}
		return nil, fmt.Errorf("API %s (status %d): %s", path, resp.StatusCode, bodyStr)
	}

	nextPage := parseNextPage(resp.Header)
	return &apiResponse{Body: body, Headers: resp.Header, NextPage: nextPage}, nil
}

func extractPageParam(urlStr string) int {
	for _, prefix := range []string{"?page=", "&page="} {
		if idx := strings.Index(urlStr, prefix); idx >= 0 {
			pageStr := urlStr[idx+len(prefix):]
			for i, c := range pageStr {
				if c == '&' || c == '#' || c == '?' {
					pageStr = pageStr[:i]
					break
				}
			}
			if page, err := strconv.Atoi(pageStr); err == nil {
				return page
			}
		}
	}
	return 0
}

func parseNextPage(h http.Header) int {
	for _, link := range h.Values("Link") {
		// Link header format: <url>; rel="next", <url>; rel="prev"
		segments := strings.Split(link, ",")
		for _, seg := range segments {
			seg = strings.TrimSpace(seg)
			if !strings.Contains(seg, `rel="next"`) {
				continue
			}
			urlStart := strings.Index(seg, "<")
			urlEnd := strings.Index(seg, ">")
			if urlStart < 0 || urlEnd < 0 || urlEnd <= urlStart {
				continue
			}
			urlStr := seg[urlStart+1 : urlEnd]
			page := extractPageParam(urlStr)
			if page > 0 {
				return page
			}
		}
	}
	return 0
}

func decode[T any](data []byte) ([]T, error) {
	var result []T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return result, nil
}

func (c *Client) ListOrganizations(ctx context.Context) ([]Organization, error) {
	resp, err := c.get(ctx, "/organizations", map[string]string{"per_page": "100"})
	if err != nil {
		return nil, fmt.Errorf("listing organizations: %w", err)
	}
	return decode[Organization](resp.Body)
}

func (c *Client) ListPipelines(ctx context.Context, orgSlug string) ([]Pipeline, error) {
	path := fmt.Sprintf("/organizations/%s/pipelines", orgSlug)
	resp, err := c.get(ctx, path, map[string]string{"per_page": "100"})
	if err != nil {
		return nil, fmt.Errorf("listing pipelines for %s: %w", orgSlug, err)
	}
	pipelines, err := decode[Pipeline](resp.Body)
	if err != nil {
		return nil, err
	}
	page := resp.NextPage
	for page > 0 && len(pipelines) < 500 {
		resp, err = c.get(ctx, path, map[string]string{"per_page": "100", "page": strconv.Itoa(page)})
		if err != nil {
			break
		}
		more, err := decode[Pipeline](resp.Body)
		if err != nil {
			break
		}
		pipelines = append(pipelines, more...)
		page = resp.NextPage
	}
	return pipelines, nil
}

func (c *Client) ListBuilds(ctx context.Context, orgSlug, pipelineSlug string) ([]Build, error) {
	path := fmt.Sprintf("/organizations/%s/pipelines/%s/builds", orgSlug, pipelineSlug)
	resp, err := c.get(ctx, path, map[string]string{"per_page": "25"})
	if err != nil {
		return nil, fmt.Errorf("listing builds for %s/%s: %w", orgSlug, pipelineSlug, err)
	}
	builds, err := decode[Build](resp.Body)
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
	resp, err := c.get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting build %s/%s#%d: %w", orgSlug, pipelineSlug, buildNumber, err)
	}
	var build Build
	if err := json.Unmarshal(resp.Body, &build); err != nil {
		return nil, fmt.Errorf("decoding build %s/%s#%d: %w", orgSlug, pipelineSlug, buildNumber, err)
	}
	return &build, nil
}

func (c *Client) ListAgents(ctx context.Context, orgSlug string) ([]Agent, error) {
	path := fmt.Sprintf("/organizations/%s/agents", orgSlug)
	resp, err := c.get(ctx, path, map[string]string{"per_page": "100"})
	if err != nil {
		return nil, fmt.Errorf("listing agents for %s: %w", orgSlug, err)
	}
	agents, err := decode[Agent](resp.Body)
	if err != nil {
		return nil, err
	}
	page := resp.NextPage
	for page > 0 && len(agents) < 500 {
		resp, err = c.get(ctx, path, map[string]string{"per_page": "100", "page": strconv.Itoa(page)})
		if err != nil {
			break
		}
		more, err := decode[Agent](resp.Body)
		if err != nil {
			break
		}
		agents = append(agents, more...)
		page = resp.NextPage
	}
	return agents, nil
}

func (c *Client) ListAnnotations(ctx context.Context, orgSlug, pipelineSlug string, buildNumber int) ([]Annotation, error) {
	path := fmt.Sprintf("/organizations/%s/pipelines/%s/builds/%d/annotations", orgSlug, pipelineSlug, buildNumber)
	resp, err := c.get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing annotations for %s/%s#%d: %w", orgSlug, pipelineSlug, buildNumber, err)
	}
	return decode[Annotation](resp.Body)
}

func (c *Client) ListArtifacts(ctx context.Context, orgSlug, pipelineSlug string, buildNumber int) ([]Artifact, error) {
	path := fmt.Sprintf("/organizations/%s/pipelines/%s/builds/%d/artifacts", orgSlug, pipelineSlug, buildNumber)
	resp, err := c.get(ctx, path, map[string]string{"per_page": "100"})
	if err != nil {
		return nil, fmt.Errorf("listing artifacts for %s/%s#%d: %w", orgSlug, pipelineSlug, buildNumber, err)
	}
	return decode[Artifact](resp.Body)
}
