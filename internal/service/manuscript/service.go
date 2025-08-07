package manuscript

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/models"
	utils "github.com/obi2na/petrel/internal/pkg"
	"github.com/obi2na/petrel/internal/service/notion"
	"go.uber.org/zap"
	"strings"
)

// Package manuscript provides the core coordination layer for draft staging in Petrel.
//

type Service interface {
	StageDraft(ctx context.Context, userID uuid.UUID, req petrelmodels.CreateDraftRequest) (petrelmodels.CreateDraftResponse, error)
}

type WorkspaceValidator interface {
	UserHasWorkspace(ctx context.Context, userID uuid.UUID, workspaceID string) (petrelmodels.UserIntegration, bool)
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
	NotionDbSvc           notion.DatabaseService
	WorkspaceValidatorMap map[string]WorkspaceValidator
	Parser                utils.Parser
	Linter                utils.MarkdownLinter
	NotionDraftService    notion.DraftService
}

func NewManuscriptService(notionSvc *notion.NotionDatabaseService, notionDraftService *notion.NotionDraftService) *ManuscriptService {

	validatorMap := map[string]WorkspaceValidator{
		"notion": notionSvc,
	}

	notionMapper := notion.NewPetrelMarkdownToNotionMapper()
	notionMapper.RegisterMappers()

	return &ManuscriptService{
		// dependencies injected here
		NotionDbSvc:           notionSvc,
		WorkspaceValidatorMap: validatorMap,
		Parser:                utils.NewDefaultMarkdownParser(),
		Linter:                utils.NewPetrelMarkdownLinter(),
		NotionDraftService:    notionDraftService,
	}
}

func (s *ManuscriptService) validateDestinations(ctx context.Context, userID uuid.UUID, destinations []petrelmodels.DraftDestination) (map[string][]petrelmodels.ValidatedDestination, []string) {
	var validationErrs []string
	validatedDestinations := make(map[string][]petrelmodels.ValidatedDestination, len(destinations))

	// for each destination in destinations slice
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
		integration, ok := validator.UserHasWorkspace(ctx, userID, destination.WorkspaceID)
		if !ok {
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

			exists, err := s.NotionDbSvc.IsValidDraftPage(ctx, userID, destination.PageID)
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

		// 4. Add validated destination to the map
		validated := petrelmodels.ValidatedDestination{
			Workspace:       destination.WorkspaceID,
			UserIntegration: integration,
		}
		result := validatedDestinations[destination.Platform]
		result = append(result, validated)
		validatedDestinations[destination.Platform] = result
	}

	return validatedDestinations, validationErrs
}

func (s *ManuscriptService) StageDraft(ctx context.Context, userID uuid.UUID, req petrelmodels.CreateDraftRequest) (petrelmodels.CreateDraftResponse, error) {
	// 1. Validate destinations
	validated, validationErrors := s.validateDestinations(ctx, userID, req.Destinations)
	if len(validationErrors) > 0 {
		// Combine all validation messages into one error
		errMsg := fmt.Sprintf("destination validation failed:\n- %s", strings.Join(validationErrors, "\n- "))
		logger.With(ctx).Error("Validation failed", zap.Strings("errors", validationErrors))
		return petrelmodels.CreateDraftResponse{
			Status: "fail",
			Drafts: []petrelmodels.DraftResultEntry{}, // No drafts created
		}, fmt.Errorf(errMsg)
	}

	// TODO: 2. Parse markdown into AST
	doc, source, err := s.Parser.Parse(req.Markdown)
	if err != nil {
		err = fmt.Errorf("markdown invalid: %w", err)
		logger.With(ctx).Error("markdown validation failed", zap.Error(err))
		return petrelmodels.CreateDraftResponse{
			Status: "fail",
			Drafts: []petrelmodels.DraftResultEntry{}, // No drafts created
		}, err
	}

	// Walk AST and collect warnings
	_, err = s.Linter.Lint(doc, source)

	// TODO: 3. Route draft to each platform's DraftService (e.g. NotionDraftService.StageDraft)
	notionDestinations := validated["notion"]
	draftResponse, err := s.NotionDraftService.StageDraft(ctx, userID, notionDestinations, doc, source)

	// TODO: 4. Collect DraftResultEntry per platform
	// TODO: 5. Return combined CreateDraftResponse

	logger.With(ctx).Info("Manuscript Service passed validation")

	// Example placeholder success response
	response := petrelmodels.CreateDraftResponse{
		Status: "success",
		Drafts: draftResponse,
	}

	return response, nil
}
