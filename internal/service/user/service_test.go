package userservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/obi2na/petrel/internal/db/models"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetOrCreateUser(t *testing.T) {

	//initialize logger
	logger.Init()

	// create mockQueries object
	mockQueries := new(utils.MockQueries)

	ctx := context.Background()
	userId := uuid.New()
	userEmail := "test@example.com"
	userName := "test-user"
	userAvatarUrl := "user-avatar-url"

	tests := []struct {
		name                     string
		userEmail                string
		userId                   uuid.UUID
		userName                 string
		userAvatarUrl            string
		errExpected              bool
		expectedErr              string
		GetUserByEmailReturnUser models.User
		GetUserByEmailReturnErr  error
		CreateUserReturnUser     models.User
		CreateUserReturnErr      error
		CreateUserMockParams     models.CreateUserParams
		UpdateLastLoginReturnErr error
	}{
		{
			name:                     "test user exists fetch user successfully",
			userEmail:                userEmail,
			userId:                   userId,
			userName:                 userName,
			userAvatarUrl:            userAvatarUrl,
			errExpected:              false,
			GetUserByEmailReturnUser: buildMockUser(userId, userEmail, userName),
			GetUserByEmailReturnErr:  nil,
			CreateUserReturnUser:     models.User{},
			CreateUserReturnErr:      nil,
			UpdateLastLoginReturnErr: nil,
		},
		{
			name:                     "test user exists fails",
			userEmail:                userEmail,
			userId:                   userId,
			userName:                 userName,
			userAvatarUrl:            userAvatarUrl,
			errExpected:              true,
			GetUserByEmailReturnUser: buildMockUser(userId, userEmail, userName),
			GetUserByEmailReturnErr:  errors.New("some db error"),
			CreateUserReturnUser:     models.User{},
			CreateUserReturnErr:      nil,
			UpdateLastLoginReturnErr: nil,
		},
		{
			name:                     "user not found, create succeeds",
			userEmail:                userEmail,
			userId:                   userId,
			userName:                 userName,
			userAvatarUrl:            userAvatarUrl,
			errExpected:              false,
			GetUserByEmailReturnUser: buildMockUser(uuid.UUID{}, userEmail, userName),
			GetUserByEmailReturnErr:  sql.ErrNoRows,
			CreateUserMockParams:     buildMockCreateUserParams(uuid.UUID{}, userEmail, userName),
			CreateUserReturnUser:     buildMockUser(uuid.UUID{}, userEmail, userName),
			CreateUserReturnErr:      nil,
			UpdateLastLoginReturnErr: nil,
		},
		{
			name:                     "user not found, create fails",
			userEmail:                userEmail,
			userId:                   userId,
			userName:                 userName,
			userAvatarUrl:            userAvatarUrl,
			errExpected:              true,
			GetUserByEmailReturnUser: buildMockUser(uuid.UUID{}, userEmail, userName),
			GetUserByEmailReturnErr:  sql.ErrNoRows,
			CreateUserMockParams:     buildMockCreateUserParams(uuid.UUID{}, userEmail, userName),
			CreateUserReturnUser:     buildMockUser(uuid.UUID{}, userEmail, userName),
			CreateUserReturnErr:      errors.New("create failed"),
			UpdateLastLoginReturnErr: nil,
		},
		{
			name:                     "update last login fails (non-fatal)",
			userEmail:                userEmail,
			userId:                   userId,
			userName:                 userName,
			userAvatarUrl:            userAvatarUrl,
			errExpected:              true,
			GetUserByEmailReturnUser: buildMockUser(userId, userEmail, userName),
			GetUserByEmailReturnErr:  nil,
			CreateUserReturnUser:     models.User{},
			CreateUserReturnErr:      nil,
			UpdateLastLoginReturnErr: errors.New("logins fail"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			mockQueries.ExpectedCalls = nil
			mockQueries.Calls = nil

			mockQueries.On("GetUserByEmail", mock.Anything, tc.userEmail).Return(tc.GetUserByEmailReturnUser, tc.GetUserByEmailReturnErr)
			mockQueries.On("CreateUser", mock.Anything, tc.CreateUserMockParams).Return(tc.CreateUserReturnUser, tc.CreateUserReturnErr)
			mockQueries.On("UpdateLastLogin", mock.Anything, tc.userId).Return(tc.UpdateLastLoginReturnErr)

			//plug mock into userService
			userService := UserService{
				queries: mockQueries,
			}

			result, err := userService.GetOrCreateUser(ctx, tc.userEmail, tc.userName, tc.userAvatarUrl)
			if tc.errExpected {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
				return
			}
			assert.NoError(t, err)
			fmt.Println(result)
		})
	}

}

func buildMockUser(userId uuid.UUID, email, name string) models.User {
	return models.User{
		ID:        userId,
		Email:     email,
		Name:      name,
		AvatarUrl: pgtype.Text{String: "user-avatar-url", Valid: true},
		Provider:  pgtype.Text{String: "petrel", Valid: true},
	}
}

func buildMockCreateUserParams(userId uuid.UUID, email, name string) models.CreateUserParams {
	return models.CreateUserParams{
		ID:        userId,
		Email:     email,
		Name:      name,
		AvatarUrl: pgtype.Text{String: "user-avatar-url", Valid: true},
		Provider:  pgtype.Text{String: "petrel", Valid: true},
	}
}
