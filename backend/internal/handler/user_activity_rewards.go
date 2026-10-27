package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func (h *UserHandler) GetCheckinCalendar(c *gin.Context) { h.checkinCalendar(c, false) }
func (h *UserHandler) ClaimCheckin(c *gin.Context)       { h.checkinCalendar(c, true) }
func (h *UserHandler) checkinCalendar(c *gin.Context, claim bool) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.affiliateService == nil {
		response.Error(c, 503, "Activity rewards unavailable")
		return
	}
	result, err := h.affiliateService.ActivityCalendar(c.Request.Context(), subject.UserID, claim)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
func (h *UserHandler) GetRewardCards(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.affiliateService == nil {
		response.Error(c, 503, "Activity rewards unavailable")
		return
	}
	result, err := h.affiliateService.RewardCards(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
