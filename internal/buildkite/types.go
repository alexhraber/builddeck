package buildkite

type Organization struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	WebURL string `json:"web_url"`
}

type Pipeline struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	Repository string `json:"repository"`
	WebURL     string `json:"web_url"`
}

type Build struct {
	ID         string   `json:"id"`
	Number     int      `json:"number"`
	State      string   `json:"state"`
	Branch     string   `json:"branch"`
	Commit     string   `json:"commit"`
	Message    string   `json:"message"`
	Creator    *Creator `json:"creator"`
	CreatedAt  string   `json:"created_at"`
	StartedAt  string   `json:"started_at"`
	FinishedAt string   `json:"finished_at"`
	WebURL     string   `json:"web_url"`
	PipelineID string   `json:"pipeline_id"`
	Jobs       []Job    `json:"jobs"`
}

type Creator struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Job struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`
	State           string   `json:"state"`
	Name            string   `json:"name"`
	Label           string   `json:"label"`
	Command         string   `json:"command"`
	AgentQueryRules []string `json:"agent_query_rules"`
	ExitStatus      *int     `json:"exit_status"`
	StartedAt       string   `json:"started_at"`
	FinishedAt      string   `json:"finished_at"`
	Agent           *Agent   `json:"agent"`
}

type Agent struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Hostname       string          `json:"hostname"`
	Version        string          `json:"version"`
	ConnectedState string          `json:"connected_state"`
	OS             string          `json:"os"`
	IPAddress      string          `json:"ip_address"`
	Metadata       []string `json:"meta_data"`
	WebURL         string   `json:"web_url"`
}

type Annotation struct {
	ID        string `json:"id"`
	BodyHTML  string `json:"body_html"`
	Style     string `json:"style"`
	Context   string `json:"context"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	WebURL    string `json:"web_url"`
}

type Artifact struct {
	ID          string `json:"id"`
	JobID       string `json:"job_id"`
	URL         string `json:"url"`
	DownloadURL string `json:"download_url"`
	Filename    string `json:"filename"`
	FileSize    int    `json:"file_size"`
	Dirname     string `json:"dirname"`
	ContentType string `json:"content_type"`
	State       string `json:"state"`
	CreatedAt   string `json:"created_at"`
	WebURL      string `json:"web_url"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
