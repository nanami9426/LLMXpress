package router

import (
	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/docs"
	"github.com/nanami9426/imgo/service"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Router() *gin.Engine {
	r := gin.Default()
	docs.SwaggerInfo.BasePath = "/"
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	r.GET("/index", service.GetIndex)
	r.GET("/user/user_list", service.GetUserList)
	return r
}
