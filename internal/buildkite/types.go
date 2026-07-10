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
	Emoji      string `json:"emoji"`
}

type Build struct {
	ID         string   `json:"id"`
	Number     int      `json:"number"`
	State      string   `json:"state"`
	Branch     string   `json:"branch"`
	Tag        string   `json:"tag"`
	Commit     string   `json:"commit"`
	Message    string   `json:"message"`
	Creator    *Creator `json:"creator"`
	CreatedAt  string   `json:"created_at"`
	StartedAt  string   `json:"started_at"`
	FinishedAt string   `json:"finished_at"`
	WebURL     string   `json:"web_url"`
	PipelineID string   `json:"pipeline_id"`
	Steps      []Step   `json:"jobs"`
}

type Creator struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Step struct {
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
	WebURL          string   `json:"web_url"`
	UnblockableID   string   `json:"unblockable_id,omitempty"`
}

type Agent struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Hostname       string   `json:"hostname"`
	Version        string   `json:"version"`
	ConnectedState string   `json:"connected_state"`
	OS             string   `json:"os"`
	IPAddress      string   `json:"ip_address"`
	Metadata       []string `json:"meta_data"`
	WebURL         string   `json:"web_url"`
	Queue          string   `json:"-"` // derived from metadata "queue=xxx"
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
	StepID      string `json:"job_id"`
	URL         string `json:"url"`
	DownloadURL string `json:"download_url"`
	Filename    string `json:"filename"`
	FileSize    int    `json:"file_size"`
	Dirname     string `json:"dirname"`
	ContentType string `json:"content_type"`
	State       string `json:"state"`
	CreatedAt   string `json:"created_at"`
	WebURL      string `json:"web_url"`
	Checksum    string `json:"-"` // populated from companion .sha256 artifact
	Tag         string `json:"-"` // populated from companion .tag artifact
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type StepLog struct {
	URL         string `json:"url"`
	Content     string `json:"content"`
	Size        int    `json:"size"`
	HeaderTimes []int  `json:"header_times"`
}

type EmojiEntry struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Tag struct {
	Name   string `json:"name"`
	Commit string `json:"commit"`
}
