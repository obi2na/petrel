package bootstrap

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/obi2na/petrel/config"
	"github.com/obi2na/petrel/internal/pkg"
	"github.com/obi2na/petrel/internal/service/auth"
	"github.com/obi2na/petrel/internal/service/manuscript"
	"github.com/obi2na/petrel/internal/service/notion"
	"github.com/obi2na/petrel/internal/service/user"
	"net/http"
	"time"
)

type ServiceContainer struct {
	UserSvc        userservice.Service
	AuthSvc        authService.AuthService
	NotionOauthSvc utils.OAuthService[notion.NotionTokenResponse]
	NotionSvc      notion.Service
	ManuscriptSvc  manuscript.Service
}

func NewServiceContainer(db *pgxpool.Pool, cache utils.Cache) *ServiceContainer {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	userSvc := userservice.NewUserService(db, cache, utils.NewJWTProvider())
	authSvc := authService.NewAuthService(config.C.Auth0, httpClient, userSvc)
	notionOauthSvc := notion.NewNotionOAuthService(httpClient)
	notionSvc := notion.NewNotionService(db, httpClient, utils.NewJomeiClient())
	manuscriptSvc := manuscript.NewManuscriptService(notionSvc)

	return &ServiceContainer{
		UserSvc:        userSvc,
		AuthSvc:        authSvc,
		NotionOauthSvc: notionOauthSvc,
		NotionSvc:      notionSvc,
		ManuscriptSvc:  manuscriptSvc,
	}
}
