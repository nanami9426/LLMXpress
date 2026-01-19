package service

import "github.com/gin-gonic/gin"

type GetIndexResp struct {
	Message string `json:"message" example:"hello"`
}

// GetIndex
// @Summary 测试路由
// @Tags example
// @Produce json
// @Success 200 {object} GetIndexResp
// @Router /index [get]
func GetIndex(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "hello",
	})
}
