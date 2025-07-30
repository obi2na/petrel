package notion

import (
	"github.com/gin-gonic/gin"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/pkg"
	"github.com/obi2na/petrel/internal/service/notion"
	"go.uber.org/zap"
	"net/http"
)

func RegisterNotionRoutes(r *gin.RouterGroup, notionIntegrationSvc notion.IntegrationService) {
	notionHandler := NewNotionHandler(notionIntegrationSvc)
	r.GET("/auth", notionHandler.AuthRedirect)
	r.GET("/auth/callback", notionHandler.NotionAuthCallback)
}

type NotionHandler[T any] struct {
	NotionIntegrationService notion.IntegrationService
}

func NewNotionHandler[T notion.NotionTokenResponse](notionIntegrationSvc notion.IntegrationService) *NotionHandler[T] {
	return &NotionHandler[T]{
		NotionIntegrationService: notionIntegrationSvc,
	}
}

func (h *NotionHandler[T]) AuthRedirect(c *gin.Context) {
	ctx := c.Request.Context()

	// handoff to notion integration service to start oauth
	redirectUrl, err := h.NotionIntegrationService.StartOauth(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	logger.With(ctx).Debug("Generated Notion OAuth URL", zap.String("url", redirectUrl))
	logger.With(ctx).Info("Redirecting to Notion OAuth")
	c.Redirect(http.StatusFound, redirectUrl)

}

func (h *NotionHandler[T]) NotionAuthCallback(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Query("code")
	state := c.Query("state")

	// get user_id from gin context
	userID, ok := utils.MustGetUserID(c)
	if !ok {
		return
	}

	logger.With(ctx).Info("Notion OAuth callback", zap.String("code", code), zap.String("state", state))

	// handoff to notion integration service to complete oauth
	notionMeta, err := h.NotionIntegrationService.CompleteOAuth(ctx, code, state, userID)
	if err != nil {
		logger.With(ctx).Error("Failed to complete notion Oauth", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "OAuth failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "success",
		"petrel_user_id": userID.String(),
		"notion": gin.H{
			"workspace_id":       notionMeta.WorkspaceID,
			"workspace_name":     notionMeta.WorkspaceName,
			"drafts_page_id":     notionMeta.DraftsPageID,
			"drafts_repo_url":    utils.BuildNotionDraftRepoUrl(notionMeta.DraftsPageID.String),
			"drafts_page_status": notionMeta.DraftsPageStatus,
			"user_id":            notionMeta.NotionUserID,
			"name":               notionMeta.NotionUserName,
			"email":              notionMeta.NotionUserEmail,
			"avatar_url":         notionMeta.NotionUserAvatar,
		},
	})
}
