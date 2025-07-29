package manuscript

import (
	"context"
	"github.com/google/uuid"
	"github.com/obi2na/petrel/internal/logger"
)

type Service interface {
	CreateDraft(ctx context.Context, userID uuid.UUID, req CreateDraftRequest) (CreateDraftResponse, error)
}

type ManuscriptService struct {
	// TODO: add dependencies like NotionClient, ConfluenceClient, DB, MarkdownParser etc
}

func NewManuscriptService() *ManuscriptService {
	return &ManuscriptService{
		// dependencies injected here
	}
}

func (s *ManuscriptService) CreateDraft(ctx context.Context, userID uuid.UUID, req CreateDraftRequest) (CreateDraftResponse, error) {

	logger.With(ctx).Info("Manuscript Service wired up")

	// Placeholder response
	response := CreateDraftResponse{
		Status: "success",
		Drafts: []DraftResultEntry{
			{
				DraftID:     uuid.New().String(),
				Platform:    "notion",
				WorkspaceID: req.Destinations[0].WorkspaceID,
				PageID:      "mock-page-id",
				URL:         "https://notion.so/mock-page-id",
				Status:      "draft",
				Action:      "created",
			},
		},
	}
	return response, nil
}
