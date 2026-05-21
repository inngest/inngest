package main

type PullRequest struct {
	Number        int      `json:"number"`
	Title         string   `json:"title"`
	URL           string   `json:"url,omitempty"`
	Body          string   `json:"body,omitempty"`
	Files         []string `json:"files,omitempty"`
	Labels        []string `json:"labels,omitempty"`
	MergeCommit   string   `json:"merge_commit,omitempty"`
	Description   string   `json:"description,omitempty"`
	ReleaseNote   string   `json:"release_note,omitempty"`
	MigrationNote string   `json:"migration_note,omitempty"`
	Excluded      bool     `json:"excluded,omitempty"`
}

type NotesFile struct {
	PRs []PullRequest `json:"prs"`
}
