package petrelmodels

import "github.com/obi2na/petrel/internal/pkg"

type CreateDraftRequest struct {
	Markdown     string             `json:"markdown" binding:"required"`
	Title        string             `json:"title" binding:"required"`
	Metadata     *DraftMetadata     `json:"metadata,omitempty"`
	Destinations []DraftDestination `json:"destinations" binding:"required"`
}

type DraftMetadata struct {
	Source string   `json:"source,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

type DraftDestination struct {
	Platform    string `json:"platform" binding:"required"` // e.g "notion", "confluence"
	WorkspaceID string `json:"workspace_id,omitempty"`      // for notion, reuse for confluence
	Append      bool   `json:"append,omitempty"`            // append to existing page or create new page
	PageID      string `json:"page_id,omitempty"`           // required if append==true
}

type CreateDraftResponse struct {
	Status string             `json:"status"` // e.g "success", "fail" or "partial_success"
	Drafts []DraftResultEntry `json:"drafts"`
}

type DraftResultEntry struct {
	DraftID      string              `json:"draft_id"`
	Platform     string              `json:"platform"`               // e.g. "notion", "confluence"
	WorkspaceID  string              `json:"workspace_id,omitempty"` // notion workspace or team id
	PageID       string              `json:"page_id"`                // internal page ID
	URL          string              `json:"url"`                    // public-facing or redirect-safe URL
	Status       string              `json:"status"`                 // e.g. "draft"
	Action       string              `json:"action"`                 // e.g. "created", "appended"
	ErrorMessage string              `json:"error,omitempty"`        // optional field for partial failures
	LintWarnings []utils.LintWarning `json:"lint_warnings,omitempty"`
}

type ValidatedDestination struct {
	NotionUserIntegration
	Workspace string
	Append    bool
	PageID    string
}

type NotionUserIntegration struct {
	Token        string
	DraftsRepoID string
}
