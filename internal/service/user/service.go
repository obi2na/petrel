package userservice

import (
	"context"
	"database/sql"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/obi2na/petrel/internal/db/models"
	"github.com/obi2na/petrel/internal/logger"
	"go.uber.org/zap"
)

type UserService struct {
	queries *models.Queries
}

func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{
		queries: models.New(db),
	}
}

func (s *UserService) GetOrCreateUser(ctx context.Context, email, name, avatarURL string) (*models.User, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	// user retrieval successful
	if err == nil {
		// update last_login_at
		if err := s.queries.UpdateLastLogin(ctx, user.ID); err != nil {
			logger.With(ctx).Error("UpdateLastLogin query failed", zap.Error(err))
		}
		return &user, nil
	}

	// database query failed
	if err != sql.ErrNoRows {
		logger.With(ctx).Error("GetUserByEmail query failed", zap.Error(err))
		return nil, err
	}

	// create new user
	newUserParams := models.CreateUserParams{
		Email:     email,
		Name:      name,
		AvatarUrl: pgtype.Text{String: avatarURL, Valid: avatarURL != ""},
		Provider:  pgtype.Text{String: "petrel", Valid: avatarURL != ""},
	}

	newUser, err := s.queries.CreateUser(ctx, newUserParams)
	if err != nil {
		logger.With(ctx).Error("CreateUser query failed", zap.Error(err))
		return nil, err
	}
	return &newUser, nil
}
