package userservice

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/db/models"
	"github.com/obi2na/petrel/internal/logger"
	utils "github.com/obi2na/petrel/internal/pkg"
	"go.uber.org/zap"
)

type Service interface {
	GetOrCreateUser(ctx context.Context, email, name, avatarURL string) (*models.User, error)
	GetUserByTokenOrID(ctx context.Context, tokenString string) (*models.User, error)
}

type UserService struct {
	queries    models.Querier
	Cache      utils.Cache
	JWTManager utils.JWTManager
}

func NewUserService(db *pgxpool.Pool, cache utils.Cache, jwtManager utils.JWTManager) *UserService {
	return &UserService{
		queries:    models.New(db),
		Cache:      cache,
		JWTManager: jwtManager,
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
				ID:        uuid.New(),
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
	logger.With(ctx).Info(user.ID.String())
	return &user, nil

}

func (s *UserService) GetUserByTokenOrID(ctx context.Context, tokenString string) (*models.User, error) {
	// 1. Try session → user ID cache
	if userID, ok := s.GetCachedUserIDByToken(ctx, tokenString); ok {
		// Try user-id → user cache
		if user, ok := s.GetCachedUserByID(ctx, userID); ok {
			return user, nil
		}

		// User not in cache -> fallback to DB + cache
		return s.getUserByIDWithCacheFallback(ctx, userID, tokenString)
	}
	// session token not in cache

	// 2. Parse JWT to extract sub
	sub, err := s.JWTManager.ParseTokenAndExtractSub(tokenString, config.C.Auth0.PetrelJWTSecret)
	if err != nil {
		logger.With(ctx).Error("bearer token parser failed", zap.Error(err))
		return nil, err
	}

	// 3. Try user cache again with sub
	if user, ok := s.GetCachedUserByID(ctx, sub); ok {
		// Backfill session cache
		s.CacheSessionToken(tokenString, sub, 3600)
		return user, nil
	}

	// 4. Load from DB and cache
	return s.getUserByIDWithCacheFallback(ctx, sub, tokenString)
}

func (s *UserService) getUserByIDWithCacheFallback(ctx context.Context, userID, tokenString string) (*models.User, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		logger.With(ctx).Error("invalid user ID", zap.String("user_id", userID), zap.Error(err))
		return nil, err
	}

	user, err := s.queries.GetUserByID(ctx, uid)
	if err != nil {
		logger.With(ctx).Error("GetUserByID query failed", zap.Error(err))
		return nil, err
	}

	// Cache both user and token → userID
	if ok := s.CacheUserByID(&user, 3600); !ok {
		logger.With(ctx).Error("CacheUserByID failed")
	}
	if ok := s.CacheSessionToken(tokenString, userID, 3600); !ok {
		logger.With(ctx).Error("CacheSessionToken failed")
	}

	return &user, nil
}

func (s *UserService) CacheSessionToken(token, userID string, ttl int64) bool {
	return s.Cache.Set("session:"+token, userID, ttl)
}

func (s *UserService) CacheUserByID(user *models.User, ttl int64) bool {
	if user == nil || user.ID == uuid.Nil {
		return false
	}

	return s.Cache.Set("user:"+user.ID.String(), user, ttl)
}

func (s *UserService) GetCachedUserByID(ctx context.Context, userID string) (*models.User, bool) {
	if val, ok := s.Cache.Get("user:" + userID); ok {
		if user, ok := val.(*models.User); ok {
			logger.With(ctx).Info("user found in cache")
			return user, true
		}
	}

	logger.With(ctx).Info("user-id does not exist in cache")
	return nil, false
}

func (s *UserService) GetCachedUserIDByToken(ctx context.Context, token string) (string, bool) {
	if val, ok := s.Cache.Get("session:" + token); ok {
		if userID, ok := val.(string); ok {
			logger.With(ctx).Info("user id found in cache")
			return userID, true
		}
	}

	logger.With(ctx).Info("token does not exist in cache")
	return "", false
}

func (s *UserService) ClearUserCache(userID string) {
	s.Cache.Del("user:" + userID)
}

func (s *UserService) ClearSessionCache(token string) {
	s.Cache.Del("session:" + token)
}
