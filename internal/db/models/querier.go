// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0

package models

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Querier interface {
	CreateIntegration(ctx context.Context, arg CreateIntegrationParams) (Integration, error)
	CreateNotionDraft(ctx context.Context, arg CreateNotionDraftParams) (NotionDraft, error)
	CreateNotionIntegration(ctx context.Context, arg CreateNotionIntegrationParams) (NotionIntegration, error)
	CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
	DeactivateUser(ctx context.Context, id uuid.UUID) error
	DeleteNotionIntegrationByIntegrationID(ctx context.Context, integrationID uuid.UUID) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	//Delete user and all user integrations
	DeleteUserIntegrations(ctx context.Context, userID pgtype.UUID) error
	GetDraftsPagesNeedingValidation(ctx context.Context) ([]NotionIntegration, error)
	GetIntegrationByService(ctx context.Context, arg GetIntegrationByServiceParams) (Integration, error)
	GetIntegrationsForUser(ctx context.Context, userID pgtype.UUID) ([]Integration, error)
	GetNotionDraftByID(ctx context.Context, id uuid.UUID) (NotionDraft, error)
	GetNotionDraftByPageID(ctx context.Context, notionPageID string) (NotionDraft, error)
	GetNotionIntegrationByIntegrationID(ctx context.Context, integrationID uuid.UUID) (NotionIntegration, error)
	GetNotionIntegrationByUserAndWorkspace(ctx context.Context, arg GetNotionIntegrationByUserAndWorkspaceParams) (NotionIntegration, error)
	GetNotionIntegrationByWorkspaceID(ctx context.Context, workspaceID string) (NotionIntegration, error)
	GetNotionIntegrationsForUser(ctx context.Context, userID pgtype.UUID) ([]Integration, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (User, error)
	IsValidNotionDraftPage(ctx context.Context, arg IsValidNotionDraftPageParams) (bool, error)
	ListNotionDraftsForUser(ctx context.Context, userID uuid.UUID) ([]NotionDraft, error)
	ListOrphanedNotionDrafts(ctx context.Context) ([]NotionDraft, error)
	ListUsers(ctx context.Context) ([]User, error)
	MarkDraftsAsOrphanedByIntegration(ctx context.Context, notionIntegrationID uuid.UUID) error
	SetPublishedPageForDraft(ctx context.Context, arg SetPublishedPageForDraftParams) error
	UpdateDraftStatus(ctx context.Context, arg UpdateDraftStatusParams) error
	UpdateDraftsPageID(ctx context.Context, arg UpdateDraftsPageIDParams) error
	UpdateDraftsPageValidationStatus(ctx context.Context, arg UpdateDraftsPageValidationStatusParams) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

var _ Querier = (*Queries)(nil)
