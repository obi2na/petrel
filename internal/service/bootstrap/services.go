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
	UserSvc                  userservice.Service
	AuthSvc                  authService.AuthService
	NotionOauthSvc           utils.OAuthService[notion.NotionTokenResponse]
	NotionDBSvc              notion.DatabaseService
	NotionIntegrationService notion.IntegrationService
	ManuscriptSvc            manuscript.Service
	NotionDraftSvc           notion.DraftService
}

func NewServiceContainer(db *pgxpool.Pool, cache utils.Cache) *ServiceContainer {

	// set up service dependencies
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	notionApiClient := utils.NewJomeiClient()
	notionMapper := notion.NewPetrelMarkdownToNotionMapper()
	notionMapper.RegisterMappers()

	// create service singletons
	userSvc := userservice.NewUserService(db, cache, utils.NewJWTProvider())
	authSvc := authService.NewAuthService(config.C.Auth0, httpClient, userSvc)
	notionOauthSvc := notion.NewNotionOAuthService(httpClient)
	notionDbSvc := notion.NewNotionDatabaseService(db, httpClient, notionApiClient)
	notionDraftSvc := notion.NewNotionDraftService(notionApiClient, notionMapper)
	manuscriptSvc := manuscript.NewManuscriptService(notionDbSvc, notionDraftSvc)
	notionIntegrationService := notion.NewIntegrationService(notionOauthSvc, notionDbSvc, utils.NewJWTProvider())

	return &ServiceContainer{
		UserSvc:                  userSvc,
		AuthSvc:                  authSvc,
		NotionOauthSvc:           notionOauthSvc,
		NotionDBSvc:              notionDbSvc,
		ManuscriptSvc:            manuscriptSvc,
		NotionIntegrationService: notionIntegrationService,
		NotionDraftSvc:           notionDraftSvc,
	}
}
