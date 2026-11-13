package user

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/handler"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/internal/service"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/response"
)

// Router 注册由管理令牌保护的用户管理接口。
func Router(engine *gin.Engine, db *gorm.DB, adminToken string) {
	if db == nil || strings.TrimSpace(adminToken) == "" {
		return
	}

	expected := sha256.Sum256([]byte(adminToken))
	users := engine.Group("/users", func(ctx *gin.Context) {
		scheme, token, ok := strings.Cut(ctx.GetHeader("Authorization"), " ")
		actual := sha256.Sum256([]byte(token))
		if !ok || !strings.EqualFold(scheme, "Bearer") || subtle.ConstantTimeCompare(expected[:], actual[:]) != 1 {
			ctx.Header("WWW-Authenticate", "Bearer")
			response.AbortSilently(ctx, errcode.Unauthorized)
		}
	})
	userHandler := handler.NewUserHandler(service.NewUserService(repository.NewUserRepository(db)))
	users.POST("", userHandler.Create)
	users.GET("", userHandler.List)
	users.GET("/:id", userHandler.Get)
	users.PUT("/:id", userHandler.Update)
	users.DELETE("/:id", userHandler.Delete)
}
