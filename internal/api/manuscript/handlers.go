package manuscript

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/obi2na/petrel/internal/logger"
	"github.com/obi2na/petrel/internal/service/manuscript"
	"go.uber.org/zap"
	"net/http"
)

func RegisterManuscriptRoutes(r *gin.RouterGroup, manuscriptSvc manuscript.Service) {

	//create manuscript handler
	manuscriptHandler := NewManuscriptHandler(manuscriptSvc)

	//register routes
	r.POST("/draft", manuscriptHandler.CreateDraft)

}

type ManuscriptHandler struct {
	Service manuscript.Service
}

func NewManuscriptHandler(service manuscript.Service) *ManuscriptHandler {
	return &ManuscriptHandler{
		Service: service,
	}
}

func (h *ManuscriptHandler) CreateDraft(c *gin.Context) {

	ctx := c.Request.Context()

	// get user_id
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
		return
	}
	userID, ok := userIDRaw.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID format"})
		return
	}

	var req manuscript.CreateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.With(ctx).Error("invalid payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "details": err.Error()})
		return
	}

	// TODO: finish implementing service
	resp, err := h.Service.CreateDraft(ctx, userID, req)
	if err != nil {
		logger.With(ctx).Error("failed to create draft", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create draft", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "implementation not completed",
		"body":    resp,
	})
}
