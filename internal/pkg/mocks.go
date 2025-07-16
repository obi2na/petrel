package utils

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/obi2na/petrel/internal/db/models"
	"github.com/stretchr/testify/mock"
)

type MockQueries struct {
	mock.Mock
}

func (m *MockQueries) CreateIntegration(ctx context.Context, arg models.CreateIntegrationParams) (models.Integration, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(models.Integration), args.Error(1)
}

func (m *MockQueries) CreateNotionIntegration(ctx context.Context, arg models.CreateNotionIntegrationParams) (models.NotionIntegration, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(models.NotionIntegration), args.Error(1)
}

func (m *MockQueries) CreateUser(ctx context.Context, arg models.CreateUserParams) (models.User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockQueries) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQueries) DeleteNotionIntegrationByIntegrationID(ctx context.Context, integrationID uuid.UUID) error {
	args := m.Called(ctx, integrationID)
	return args.Error(0)
}

func (m *MockQueries) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQueries) DeleteUserIntegrations(ctx context.Context, userID pgtype.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockQueries) GetIntegrationByService(ctx context.Context, arg models.GetIntegrationByServiceParams) (models.Integration, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(models.Integration), args.Error(1)
}

func (m *MockQueries) GetIntegrationsForUser(ctx context.Context, userID pgtype.UUID) ([]models.Integration, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Integration), args.Error(1)
}

func (m *MockQueries) GetNotionIntegrationByIntegrationID(ctx context.Context, integrationID uuid.UUID) (models.NotionIntegration, error) {
	args := m.Called(ctx, integrationID)
	return args.Get(0).(models.NotionIntegration), args.Error(1)
}

func (m *MockQueries) GetNotionIntegrationByWorkspaceID(ctx context.Context, workspaceID string) (models.NotionIntegration, error) {
	args := m.Called(ctx, workspaceID)
	return args.Get(0).(models.NotionIntegration), args.Error(1)
}

func (m *MockQueries) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockQueries) GetUserByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockQueries) ListUsers(ctx context.Context) ([]models.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockQueries) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
