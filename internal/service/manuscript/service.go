package manuscript

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/service/notion"
	"go.uber.org/zap"
	"strings"
)

// Package manuscript provides the core coordination layer for draft staging in Petrel.
//

type Service interface {
	StageDraft(ctx context.Context, userID uuid.UUID, req CreateDraftRequest) (CreateDraftResponse, error)
}

type WorkspaceValidator interface {
	UserHasWorkspace(ctx context.Context, userID uuid.UUID, workspaceID string) (bool, error)
}

// The ManuscriptService is responsible for:
// - Validating incoming draft requests, including destination integrity and append behavior
// - Delegating draft staging to platform-specific services (e.g. Notion, Confluence)
// - Aggregating draft creation results across multiple destinations
// It does not directly handle Markdown parsing or external API calls.
// Those responsibilities are delegated to downstream platform services like NotionDraftService.
// This service represents the entry point for handling /manuscript/draft requests and is
// designed to remain platform-agnostic while enforcing high-level rules and workflows.
type ManuscriptService struct {
	// TODO: add dependencies like NotionClient, ConfluenceClient, DB, MarkdownParser etc
	NotionSvc             notion.DatabaseService
	WorkspaceValidatorMap map[string]WorkspaceValidator
}

func NewManuscriptService(notionSvc notion.DatabaseService) *ManuscriptService {

	validatorMap := map[string]WorkspaceValidator{
		"notion": notionSvc,
	}

	return &ManuscriptService{
		// dependencies injected here
		NotionSvc:             notionSvc,
		WorkspaceValidatorMap: validatorMap,
	}
}

func (s *ManuscriptService) destinationsAreValid(ctx context.Context, userID uuid.UUID, destinations []DraftDestination) []string {
	var validationErrs []string

	for _, destination := range destinations {
		// 1. Check that platform exists
		validator, ok := s.WorkspaceValidatorMap[destination.Platform]
		if !ok {
			errStr := fmt.Sprintf("%s platform does not exist", destination.Platform)
			logger.With(ctx).Error(errStr)
			validationErrs = append(validationErrs, errStr)
			continue
		}

		// 2. Check user has access to workspace
		valid, err := validator.UserHasWorkspace(ctx, userID, destination.WorkspaceID)
		if err != nil {
			logger.With(ctx).Error("error querying database", zap.Error(err))
			validationErrs = append(validationErrs, fmt.Sprintf("workspace check failed for %s: %v", destination.Platform, err))
			continue
		}
		if !valid {
			errStr := fmt.Sprintf("unauthorized access to %s workspace %s", destination.Platform, destination.WorkspaceID)
			logger.With(ctx).Error(errStr)
			validationErrs = append(validationErrs, errStr)
			continue
		}

		// 3. If append is true, check that page_id is valid and belongs to user
		if destination.Append {
			if destination.PageID == "" {
				errStr := fmt.Sprintf("append requested but page_id is missing for platform %s", destination.Platform)
				logger.With(ctx).Error(errStr)
				validationErrs = append(validationErrs, errStr)
				continue
			}

			exists, err := s.NotionSvc.IsValidDraftPage(ctx, userID, destination.PageID)
			if err != nil {
				errStr := fmt.Sprintf("could not validate page_id %s for %s: %v", destination.PageID, destination.Platform, err)
				logger.With(ctx).Error(errStr)
				validationErrs = append(validationErrs, errStr)
				continue
			}
			if !exists {
				errStr := fmt.Sprintf("page_id %s is not a valid draft for platform %s", destination.PageID, destination.Platform)
				logger.With(ctx).Error(errStr)
				validationErrs = append(validationErrs, errStr)
			}
		}
	}

	return validationErrs
}

func (s *ManuscriptService) StageDraft(ctx context.Context, userID uuid.UUID, req CreateDraftRequest) (CreateDraftResponse, error) {
	// 1. Validate destinations
	validationErrors := s.destinationsAreValid(ctx, userID, req.Destinations)
	if len(validationErrors) > 0 {
		// Combine all validation messages into one error
		errMsg := fmt.Sprintf("destination validation failed:\n- %s", strings.Join(validationErrors, "\n- "))
		logger.With(ctx).Error("Validation failed", zap.Strings("errors", validationErrors))
		return CreateDraftResponse{
			Status: "fail",
			Drafts: []DraftResultEntry{}, // No drafts created
		}, fmt.Errorf(errMsg)
	}

	// 2. Proceed with markdown processing and destination publishing (placeholder)
	logger.With(ctx).Info("Manuscript Service passed validation")

	// TODO: 2. Parse markdown into AST
	// TODO: 3. Route draft to each platform's DraftService (e.g. NotionDraftService.StageDraft)
	// TODO: 4. Collect DraftResultEntry per platform
	// TODO: 5. Return combined CreateDraftResponse

	// Example placeholder success response
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
