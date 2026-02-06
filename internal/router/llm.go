package router

import "github.com/gin-gonic/gin"

func RigisterLLMRoutes(r *gin.Engine) {
	v1 := r.Group("/v1")
	v1.Use(AuthMiddleware())
}
