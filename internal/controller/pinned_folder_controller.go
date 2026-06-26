package controller

import (
	"go-file-server/internal/service"

	"github.com/gin-gonic/gin"
)

func PinnedFolderRoutes(authRouter *gin.RouterGroup, svc *service.PinnedFolderService) {
	pin := authRouter.Group("/pin")
	{
		pin.POST("", svc.Add)
		pin.DELETE("", svc.Remove)
	}
	pins := authRouter.Group("/pins")
	{
		pins.GET("", svc.List)
		pins.PUT("/reorder", svc.Reorder)
	}
}
