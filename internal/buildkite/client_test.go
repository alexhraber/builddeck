package buildkite

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListOrganizations(t *testing.T) {
	orgs := []Organization{
		{ID: "1", Slug: "my-org", Name: "My Org"},
		{ID: "2", Slug: "other-org", Name: "Other Org"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(orgs)
	}))
	defer srv.Close()

	client := NewClient("test-token")
	client.BaseURL = srv.URL

	got, err := client.ListOrganizations(context.Background())
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer srv.Close()

	client := NewClient("bad-token")
	client.BaseURL = srv.URL

	_, err := client.ListOrganizations(context.Background())
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
}

func TestListPipelines(t *testing.T) {
	pipes := []Pipeline{
		{ID: "p1", Slug: "my-pipe", Name: "My Pipeline"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/test-org/pipelines" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(pipes)
	}))
	defer srv.Close()

	client := NewClient("token")
	client.BaseURL = srv.URL

	got, err := client.ListPipelines(context.Background(), "test-org")
	if err != nil {
		t.Fatalf("ListPipelines: %v", err)
	}
	if len(got) != 1 || got[0].Slug != "my-pipe" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestListBuilds(t *testing.T) {
	builds := []Build{
		{ID: "b1", Number: 42, State: "passed", Branch: "main", Commit: "abc1234", Jobs: []Job{{ID: "j1", State: "passed"}}},
		{ID: "b2", Number: 41, State: "failed", Branch: "main", Commit: "def5678", Jobs: nil},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(builds)
	}))
	defer srv.Close()

	client := NewClient("token")
	client.BaseURL = srv.URL

	got, err := client.ListBuilds(context.Background(), "org", "pipe")
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(build)
	}))
	defer srv.Close()

	client := NewClient("token")
	client.BaseURL = srv.URL

	got, err := client.GetBuild(context.Background(), "org", "pipe", 42)
	if err != nil {
		t.Fatalf("GetBuild: %v", err)
	}
	if got.Number != 42 {
		t.Errorf("expected build number 42, got %d", got.Number)
	}
	if len(got.Jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(got.Jobs))
	}
}

func TestMissingToken(t *testing.T) {
	client := NewClient("")
	client.BaseURL = "http://nonexistent:99999"
	_, err := client.ListOrganizations(context.Background())
	if err == nil {
		t.Fatal("expected error with empty token")
	}
}
