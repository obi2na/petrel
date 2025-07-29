package manuscript

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/obi2na/petrel/internal/service/manuscript"
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

	// get user_id
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
		return
	}
	_, ok := userIDRaw.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID format"})
		return
	}

	// TODO: implement
	c.JSON(http.StatusOK, gin.H{"message": "Not yet implemented"})
}
