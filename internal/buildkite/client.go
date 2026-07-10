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

func (c *Client) put(ctx context.Context, path string) (*apiResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")

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

	return &apiResponse{Body: body, Headers: resp.Header}, nil
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
		if len(builds[i].Steps) == 0 {
			builds[i].Steps = nil
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

// GetTagArtifact fetches the tag.txt artifact content for a build
func (c *Client) GetTagArtifact(ctx context.Context, orgSlug, pipelineSlug string, buildNumber int) (string, error) {
	// List artifacts for this build
	artifacts, err := c.ListArtifacts(ctx, orgSlug, pipelineSlug, buildNumber)
	if err != nil {
		return "", err
	}

	// Find tag.txt artifact
	var tagArtifact *Artifact
	for _, a := range artifacts {
		if strings.HasSuffix(a.Filename, "tag.txt") {
			tagArtifact = &a
			break
		}
	}
	if tagArtifact == nil {
		return "", nil // Not found
	}

	// Download the artifact
	url, err := c.DownloadArtifactURL(ctx, orgSlug, pipelineSlug, buildNumber, tagArtifact.StepID, tagArtifact.ID)
	if err != nil {
		return "", err
	}
	if url == "" {
		return "", nil
	}

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("downloading tag artifact: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading tag artifact: %w", err)
	}

	return strings.TrimSpace(string(body)), nil
}

func (c *Client) RebuildBuild(ctx context.Context, orgSlug, pipelineSlug string, buildNumber int) (*Build, error) {
	path := fmt.Sprintf("/organizations/%s/pipelines/%s/builds/%d/rebuild", orgSlug, pipelineSlug, buildNumber)
	resp, err := c.put(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("rebuilding build %s/%s#%d: %w", orgSlug, pipelineSlug, buildNumber, err)
	}
	var build Build
	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &build); err != nil {
			return nil, fmt.Errorf("decoding rebuilt build %s/%s#%d: %w", orgSlug, pipelineSlug, buildNumber, err)
		}
	}
	return &build, nil
}

func (c *Client) CancelBuild(ctx context.Context, orgSlug, pipelineSlug string, buildNumber int) (*Build, error) {
	path := fmt.Sprintf("/organizations/%s/pipelines/%s/builds/%d/cancel", orgSlug, pipelineSlug, buildNumber)
	resp, err := c.put(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("canceling build %s/%s#%d: %w", orgSlug, pipelineSlug, buildNumber, err)
	}
	var build Build
	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &build); err != nil {
			return nil, fmt.Errorf("decoding canceled build %s/%s#%d: %w", orgSlug, pipelineSlug, buildNumber, err)
		}
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
	var allArtifacts []Artifact
	page := 1
	for {
		path := fmt.Sprintf("/organizations/%s/pipelines/%s/builds/%d/artifacts", orgSlug, pipelineSlug, buildNumber)
		resp, err := c.get(ctx, path, map[string]string{"per_page": "100", "page": strconv.Itoa(page)})
		if err != nil {
			return nil, fmt.Errorf("listing artifacts for %s/%s#%d (page %d): %w", orgSlug, pipelineSlug, buildNumber, page, err)
		}

		artifacts, err := decode[Artifact](resp.Body)
		if err != nil {
			return nil, fmt.Errorf("decoding artifacts for %s/%s#%d (page %d): %w", orgSlug, pipelineSlug, buildNumber, page, err)
		}

		if len(artifacts) == 0 {
			break
		}

		allArtifacts = append(allArtifacts, artifacts...)
		if len(artifacts) < 100 {
			break
		}
		page++
	}
	return allArtifacts, nil
}

func (c *Client) getRawText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "text/plain")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) GetStepLog(ctx context.Context, orgSlug, pipelineSlug string, buildNumber int, stepID string) (*StepLog, error) {
	path := fmt.Sprintf("/organizations/%s/pipelines/%s/builds/%d/jobs/%s/log", orgSlug, pipelineSlug, buildNumber, stepID)
	resp, err := c.get(ctx, path, map[string]string{"content": "true"})
	if err != nil {
		return nil, fmt.Errorf("getting step log for %s/%s#%d step %s: %w", orgSlug, pipelineSlug, buildNumber, stepID, err)
	}
	var stepLog StepLog
	if err := json.Unmarshal(resp.Body, &stepLog); err != nil {
		return nil, fmt.Errorf("decoding step log: %w", err)
	}
	if stepLog.Content == "" && stepLog.URL != "" {
		raw, err := c.getRawText(ctx, stepLog.URL)
		if err == nil && raw != "" {
			stepLog.Content = strings.TrimRight(raw, "\n\r\t ")
		}
	}
	return &stepLog, nil
}

func (c *Client) RetryStep(ctx context.Context, orgSlug, pipelineSlug string, buildNumber int, stepID string) error {
	path := fmt.Sprintf("/organizations/%s/pipelines/%s/builds/%d/jobs/%s/retry", orgSlug, pipelineSlug, buildNumber, stepID)
	if _, err := c.put(ctx, path); err != nil {
		return fmt.Errorf("retrying step %s/%s#%d %s: %w", orgSlug, pipelineSlug, buildNumber, stepID, err)
	}
	return nil
}

func (c *Client) UnblockStep(ctx context.Context, orgSlug, pipelineSlug string, buildNumber int, stepID string) error {
	path := fmt.Sprintf("/organizations/%s/pipelines/%s/builds/%d/jobs/%s/unblock", orgSlug, pipelineSlug, buildNumber, stepID)
	if _, err := c.put(ctx, path); err != nil {
		return fmt.Errorf("unblocking step %s/%s#%d %s: %w", orgSlug, pipelineSlug, buildNumber, stepID, err)
	}
	return nil
}

// DownloadArtifactURL returns a redirect URL for artifact download.
func (c *Client) DownloadArtifactURL(ctx context.Context, orgSlug, pipelineSlug string, buildNumber int, stepID, artifactID string) (string, error) {
	path := fmt.Sprintf("/organizations/%s/pipelines/%s/builds/%d/jobs/%s/artifacts/%s/download", orgSlug, pipelineSlug, buildNumber, stepID, artifactID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return "", fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	// Don't follow redirects — we want the redirect URL
	noRedirectClient := &http.Client{
		Timeout: c.HTTPClient.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("artifact download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusSeeOther {
		return resp.Header.Get("Location"), nil
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("artifact download failed (status %d): %s", resp.StatusCode, string(body))
	}

	return resp.Header.Get("Location"), nil
}

func (c *Client) ListEmojis(ctx context.Context, orgSlug string) ([]EmojiEntry, error) {
	path := fmt.Sprintf("/organizations/%s/emojis", orgSlug)
	resp, err := c.get(ctx, path, map[string]string{"per_page": "100"})
	if err != nil {
		return nil, fmt.Errorf("listing emojis for %s: %w", orgSlug, err)
	}
	emojis, err := decode[EmojiEntry](resp.Body)
	if err != nil {
		return nil, err
	}
	page := resp.NextPage
	for page > 0 {
		resp, err = c.get(ctx, path, map[string]string{"per_page": "100", "page": strconv.Itoa(page)})
		if err != nil {
			break
		}
		more, err := decode[EmojiEntry](resp.Body)
		if err != nil {
			break
		}
		emojis = append(emojis, more...)
		page = resp.NextPage
	}
	return emojis, nil
}

// ParseAgentQueue extracts the "queue" from agent metadata.
func ParseAgentQueue(agent Agent) string {
	for _, meta := range agent.Metadata {
		if strings.HasPrefix(meta, "queue=") {
			return strings.TrimPrefix(meta, "queue=")
		}
	}
	return "default"
}

// GetTagsForCommit fetches tags pointing to a specific commit SHA
func (c *Client) GetTagsForCommit(ctx context.Context, orgSlug, pipelineSlug, commitSHA string) ([]Tag, error) {
	path := fmt.Sprintf("/organizations/%s/pipelines/%s/builds/%s/tags", orgSlug, pipelineSlug, commitSHA)
	resp, err := c.get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting tags for commit %s: %w", commitSHA, err)
	}
	var tags []Tag
	if err := json.Unmarshal(resp.Body, &tags); err != nil {
		return nil, fmt.Errorf("decoding tags for commit %s: %w", commitSHA, err)
	}
	return tags, nil
}

// ListTagsForCommit fetches tags for a commit (alias for GetTagsForCommit)
func (c *Client) ListTagsForCommit(ctx context.Context, orgSlug, pipelineSlug, commitSHA string) ([]Tag, error) {
	return c.GetTagsForCommit(ctx, orgSlug, pipelineSlug, commitSHA)
}
