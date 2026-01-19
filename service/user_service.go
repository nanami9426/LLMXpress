package service

import (
	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/models"
)

type GetUserListResp struct {
	Data []*models.UserBasic `json:"data"`
}

// GetUserList
// @Summary 用户列表
// @Description 返回所有用户的列表
// @Tags users
// @Produce json
// @Success 200 {object} GetUserListResp
// @Router /user/user_list [get]
func GetUserList(c *gin.Context) {
	var user_list []*models.UserBasic
	user_list = models.GetUserList()
	c.JSON(200, gin.H{
		"data": user_list,
	})
}
