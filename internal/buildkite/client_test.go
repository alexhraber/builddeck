package buildkite

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func testClient(srv *httptest.Server) *Client {
	c := NewClient("test-token")
	c.BaseURL = srv.URL
	return c
}

func TestListOrganizations(t *testing.T) {
	orgs := []Organization{
		{ID: "1", Slug: "my-org", Name: "My Org"},
		{ID: "2", Slug: "other-org", Name: "Other Org"},
	}
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/organizations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(orgs)
	})
	defer srv.Close()

	got, err := testClient(srv).ListOrganizations(context.Background())
	if err != nil {
		t.Fatalf("ListOrganizations: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 orgs, got %d", len(got))
	}
	if got[0].Slug != "my-org" {
		t.Errorf("expected slug my-org, got %s", got[0].Slug)
	}
}

func TestListOrganizationsAuthError(t *testing.T) {
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Unauthorized"}`))
	})
	defer srv.Close()

	_, err := testClient(srv).ListOrganizations(context.Background())
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
}

func TestListOrganizationsServerError(t *testing.T) {
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`internal server error`))
	})
	defer srv.Close()

	_, err := testClient(srv).ListOrganizations(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestListPipelines(t *testing.T) {
	pipes := []Pipeline{
		{ID: "p1", Slug: "my-pipe", Name: "My Pipeline"},
	}
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/test-org/pipelines" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(pipes)
	})
	defer srv.Close()

	got, err := testClient(srv).ListPipelines(context.Background(), "test-org")
	if err != nil {
		t.Fatalf("ListPipelines: %v", err)
	}
	if len(got) != 1 || got[0].Slug != "my-pipe" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestListPipelinesPagination(t *testing.T) {
	page1 := []Pipeline{{ID: "p1", Slug: "pipe-1", Name: "Pipe 1"}}
	page2 := []Pipeline{{ID: "p2", Slug: "pipe-2", Name: "Pipe 2"}}
	callCount := 0
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Query().Get("page") == "" {
			w.Header().Set("Link", `<http://example.com?page=2>; rel="next"`)
			json.NewEncoder(w).Encode(page1)
		} else {
			json.NewEncoder(w).Encode(page2)
		}
	})
	defer srv.Close()

	got, err := testClient(srv).ListPipelines(context.Background(), "test-org")
	if err != nil {
		t.Fatalf("ListPipelines: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 pipelines with pagination, got %d", len(got))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

func TestListBuilds(t *testing.T) {
	builds := []Build{
		{ID: "b1", Number: 42, State: "passed", Branch: "main", Commit: "abc1234", Jobs: []Job{{ID: "j1", State: "passed"}}},
		{ID: "b2", Number: 41, State: "failed", Branch: "main", Commit: "def5678", Jobs: nil},
	}
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(builds)
	})
	defer srv.Close()

	got, err := testClient(srv).ListBuilds(context.Background(), "org", "pipe")
	if err != nil {
		t.Fatalf("ListBuilds: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 builds, got %d", len(got))
	}
	if len(got[0].Jobs) != 1 {
		t.Errorf("expected 1 job for first build, got %d", len(got[0].Jobs))
	}
	if got[1].Jobs != nil {
		t.Errorf("expected nil jobs for second build, got %v", got[1].Jobs)
	}
}

func TestGetBuild(t *testing.T) {
	build := Build{ID: "b1", Number: 42, State: "passed", Branch: "main", Commit: "abc1234", Jobs: []Job{{ID: "j1", State: "passed", Label: "Build"}}}
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/org/pipelines/pipe/builds/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(build)
	})
	defer srv.Close()

	got, err := testClient(srv).GetBuild(context.Background(), "org", "pipe", 42)
	if err != nil {
		t.Fatalf("GetBuild: %v", err)
	}
	if got.Number != 42 {
		t.Errorf("expected build number 42, got %d", got.Number)
	}
}

func TestListAgents(t *testing.T) {
	agents := []Agent{
		{ID: "a1", Name: "agent-1", Hostname: "host-1", ConnectedState: "connected"},
		{ID: "a2", Name: "agent-2", Hostname: "host-2", ConnectedState: "disconnected"},
	}
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/test-org/agents" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(agents)
	})
	defer srv.Close()

	got, err := testClient(srv).ListAgents(context.Background(), "test-org")
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(got))
	}
	if got[0].Name != "agent-1" {
		t.Errorf("expected agent-1, got %s", got[0].Name)
	}
}

func TestListAnnotations(t *testing.T) {
	anns := []Annotation{
		{ID: "ann1", BodyHTML: "<p>Test</p>", Style: "info", Context: "default"},
	}
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/org/pipelines/pipe/builds/42/annotations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(anns)
	})
	defer srv.Close()

	got, err := testClient(srv).ListAnnotations(context.Background(), "org", "pipe", 42)
	if err != nil {
		t.Fatalf("ListAnnotations: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(got))
	}
	if got[0].Style != "info" {
		t.Errorf("expected style info, got %s", got[0].Style)
	}
}

func TestListAnnotationsEmpty(t *testing.T) {
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`))
	})
	defer srv.Close()

	got, err := testClient(srv).ListAnnotations(context.Background(), "org", "pipe", 1)
	if err != nil {
		t.Fatalf("ListAnnotations: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 annotations, got %d", len(got))
	}
}

func TestListArtifacts(t *testing.T) {
	arts := []Artifact{
		{ID: "art1", Filename: "log.txt", FileSize: 1024, State: "finished"},
	}
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/org/pipelines/pipe/builds/42/artifacts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(arts)
	})
	defer srv.Close()

	got, err := testClient(srv).ListArtifacts(context.Background(), "org", "pipe", 42)
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(got))
	}
	if got[0].Filename != "log.txt" {
		t.Errorf("expected filename log.txt, got %s", got[0].Filename)
	}
}

func TestListArtifactsEmpty(t *testing.T) {
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`))
	})
	defer srv.Close()

	got, err := testClient(srv).ListArtifacts(context.Background(), "org", "pipe", 1)
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 artifacts, got %d", len(got))
	}
}

func TestParseNextPage(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		want   int
	}{
		{"no link header", http.Header{}, 0},
		{"next page 2", http.Header{"Link": {`<http://api?page=2>; rel="next"`}}, 2},
		{"next page 3 with other params", http.Header{"Link": {`<http://api?per_page=100&page=3>; rel="next"`}}, 3},
		{"no next rel", http.Header{"Link": {`<http://api?page=2>; rel="prev"`}}, 0},
		{"multiple links", http.Header{"Link": {`<http://api?page=1>; rel="prev", <http://api?page=3>; rel="next"`}}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNextPage(tt.header)
			if got != tt.want {
				t.Errorf("parseNextPage() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestClientMissingToken(t *testing.T) {
	c := NewClient("")
	c.BaseURL = "http://nonexistent:99999"
	_, err := c.ListOrganizations(context.Background())
	if err == nil {
		t.Fatal("expected error with empty token")
	}
}

func TestGetJobLog(t *testing.T) {
	logResp := JobLog{
		URL:     "http://example.com/log",
		Content: "hello log",
	}
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/org/pipelines/pipe/builds/42/jobs/j1/log" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(logResp)
	})
	defer srv.Close()

	got, err := testClient(srv).GetJobLog(context.Background(), "org", "pipe", 42, "j1")
	if err != nil {
		t.Fatalf("GetJobLog: %v", err)
	}
	if got.Content != "hello log" {
		t.Errorf("expected hello log, got %s", got.Content)
	}
}

func TestRetryJob(t *testing.T) {
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/organizations/org/pipelines/pipe/builds/42/jobs/j1/retry" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
	})
	defer srv.Close()

	if err := testClient(srv).RetryJob(context.Background(), "org", "pipe", 42, "j1"); err != nil {
		t.Fatalf("RetryJob: %v", err)
	}
}

func TestRebuildBuild(t *testing.T) {
	buildResp := Build{ID: "b2", Number: 43, State: "scheduled"}
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/organizations/org/pipelines/pipe/builds/42/rebuild" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(buildResp)
	})
	defer srv.Close()

	got, err := testClient(srv).RebuildBuild(context.Background(), "org", "pipe", 42)
	if err != nil {
		t.Fatalf("RebuildBuild: %v", err)
	}
	if got.Number != 43 {
		t.Errorf("expected rebuilt build 43, got %d", got.Number)
	}
}

func TestCancelBuild(t *testing.T) {
	buildResp := Build{ID: "b1", Number: 42, State: "canceling"}
	srv := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/organizations/org/pipelines/pipe/builds/42/cancel" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(buildResp)
	})
	defer srv.Close()

	got, err := testClient(srv).CancelBuild(context.Background(), "org", "pipe", 42)
	if err != nil {
		t.Fatalf("CancelBuild: %v", err)
	}
	if got.State != "canceling" {
		t.Errorf("expected canceling state, got %s", got.State)
	}
}
