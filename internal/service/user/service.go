package userservice

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/obi2na/petrel/internal/db/models"
	"github.com/obi2na/petrel/internal/logger"
	"go.uber.org/zap"
)

type Service interface {
	GetOrCreateUser(ctx context.Context, email, name, avatarURL string) (*models.User, error)
}

type UserService struct {
	queries models.Querier
}

func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{
		queries: models.New(db),
	}
}

func (s *UserService) GetOrCreateUser(ctx context.Context, email, name, avatarURL string) (*models.User, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)

	if err != nil {
		// user does not exist in table. no rows found
		if errors.Is(err, sql.ErrNoRows) {
			logger.With(ctx).Info("user does not exist. creating new user",
				zap.String("email", email),
			)
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

		logger.With(ctx).Error("GetUserByEmail query failed", zap.Error(err))
		return nil, err
	}

	logger.With(ctx).Info("user fetched from db",
		zap.String("email", email),
	)

	// user retrieval successful. update last_login_at
	if err := s.queries.UpdateLastLogin(ctx, user.ID); err != nil {
		logger.With(ctx).Error("UpdateLastLogin query failed", zap.Error(err))
		return nil, err
	}
	return &user, nil

}
