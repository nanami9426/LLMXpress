package router

import (
	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/internal/router/middlewares"
	"github.com/nanami9426/imgo/internal/service"
)

func RegisterUserRoutes(r *gin.Engine) {
	user := r.Group("/user")
	{
		user.POST("/user_list", service.GetUserList)
		user.POST("/create_user", service.CreateUser)
		user.POST("/del_user", service.DeleteUser)
		user.POST("/update_user", service.UpdateUser)
		user.POST("/user_login", service.UserLogin)
		user.POST("/check_token", service.CheckToken)
	}

	userAuth := user.Group("")
	userAuth.Use(middlewares.AuthMiddleware())
	{
		userAuth.POST("/create_api_key", service.CreateAPIKey)
		userAuth.POST("/api_key_list", service.ListAPIKeys)
		userAuth.POST("/revoke_api_key", service.RevokeAPIKey)
	}

}
