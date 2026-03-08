package service

import (
	"net/http"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/internal/models"
	"github.com/nanami9426/imgo/internal/utils"
)

var (
	createAPIKeyFn        = models.CreateAPIKey
	listAPIKeysByUserFn   = models.ListAPIKeysByUser
	revokeAPIKeyByIDFn    = models.RevokeAPIKeyByIDAndUser
	generateAPIKeyTokenFn = utils.GenerateAPIKeyToken
)

type CreateUserReq struct {
	UserName   string `json:"user_name" form:"user_name" binding:"required"`
	Password   string `json:"password" form:"password" binding:"required"`
	RePassword string `json:"re_password" form:"re_password" binding:"required"`
	Email      string `json:"email" form:"email"`
}

type DeleteUserReq struct {
	UserID int64 `json:"user_id" form:"user_id" binding:"required"`
}

type UpdateUserReq struct {
	UserID   int64  `json:"user_id" form:"user_id" binding:"required"`
	UserName string `json:"user_name" form:"user_name"`
	Email    string `json:"email" form:"email"`
}

type UserLoginReq struct {
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}

type CheckTokenReq struct {
	Token string `json:"token" form:"token"`
}

type CreateAPIKeyReq struct {
	Name      string `json:"name" form:"name" binding:"required"`
	ExpiresAt string `json:"expires_at" form:"expires_at"`
}

type RevokeAPIKeyReq struct {
	APIKeyID int64 `json:"api_key_id" form:"api_key_id" binding:"required"`
}

type apiKeyMetaResp struct {
	APIKeyID   int64   `json:"api_key_id"`
	Name       string  `json:"name"`
	Prefix     string  `json:"prefix"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"created_at"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
	LastUsedIP string  `json:"last_used_ip,omitempty"`
}

type createAPIKeyResp struct {
	APIKeyID  int64   `json:"api_key_id"`
	Name      string  `json:"name"`
	Prefix    string  `json:"prefix"`
	Key       string  `json:"key"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// @Summary 用户列表
// @Description 返回包含所有用户信息的列表
// @Tags users
// @Produce json
// @Router /user/user_list [post]
func GetUserList(c *gin.Context) {
	user_list, err := models.GetUserList()
	if err != nil {
		utils.Fail(c, http.StatusOK, utils.StatDatabaseError, "获取用户列表失败", err)
		return
	}
	utils.Success(c, user_list)
}

// @Summary 创建新用户
// @Tags users
// @Produce json
// @Router /user/create_user [post]
// @param user_name formData string true "用户名"
// @param password formData string true "密码"
// @param re_password formData string true "确认密码"
// @param email formData string false "邮箱"
func CreateUser(c *gin.Context) {
	req := &CreateUserReq{}
	if err := c.ShouldBind(req); err != nil {
		utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "参数错误", err)
		return
	}
	user := &models.UserBasic{}
	user.Name = req.UserName
	password := req.Password
	re_password := req.RePassword
	if password != re_password {
		utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "两次输入的密码不一致", nil)
		return
	}
	user.Password, _ = utils.HashPassword(password)
	if !govalidator.IsEmail(req.Email) {
		utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "邮箱格式错误", nil)
		return
	}
	if models.EmailIsExists(req.Email) {
		utils.Fail(c, http.StatusOK, utils.StatConflict, "该邮箱已注册", nil)
		return
	}
	user.Email = req.Email
	user_id := utils.GenerateUserID()
	user.UserID = user_id
	if err := models.CreateUser(user); err != nil {
		utils.Fail(c, http.StatusOK, utils.StatDatabaseError, "注册失败", err)
		return
	}
	utils.SuccessMessage(c, "注册成功")
}

// @Summary 删除用户
// @Tags users
// @Produce json
// @Router /user/del_user [post]
// @param user_id formData int true "用户id"
func DeleteUser(c *gin.Context) {
	req := &DeleteUserReq{}
	if err := c.ShouldBind(req); err != nil {
		utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "参数错误", err)
		return
	}
	user, rows := models.FindUserByUserID(req.UserID)
	if rows == 0 {
		utils.Fail(c, http.StatusOK, utils.StatNotFound, "用户不存在", nil)
		return
	}
	_, err := models.DeleteUser(&user)
	if err != nil {
		utils.Fail(c, http.StatusOK, utils.StatDatabaseError, "删除失败", err)
		return
	}

	utils.SuccessMessage(c, "删除成功")
}

// @Summary 更新用户信息
// @Tags users
// @Produce json
// @Router /user/update_user [post]
// @param user_id formData int true "用户id"
// @param user_name formData string false "用户名"
// @param email formData string false "邮箱"
func UpdateUser(c *gin.Context) {
	req := &UpdateUserReq{}
	if err := c.ShouldBind(req); err != nil {
		utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "参数错误", err)
		return
	}
	if !govalidator.IsEmail(req.Email) && "" != req.Email {
		utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "邮箱格式错误", nil)
		return
	}
	data_update := map[string]interface{}{
		"UserID": req.UserID,
		"Name":   req.UserName,
	}
	if "" != req.Email {
		if models.EmailIsExists(req.Email) {
			utils.Fail(c, http.StatusOK, utils.StatConflict, "该邮箱已注册", nil)
			return
		}
		data_update["Email"] = req.Email
	}
	rows, err := models.UpdateUser(data_update)
	if err != nil {
		utils.Fail(c, http.StatusOK, utils.StatDatabaseError, "修改失败", err)
		return
	}
	if rows == 0 {
		utils.Fail(c, http.StatusOK, utils.StatNotFound, "用户不存在", nil)
		return
	}
	utils.SuccessMessage(c, "修改成功")
}

// @Summary 用户登录
// @Tags users
// @Produce json
// @Router /user/user_login [post]
// @param email formData string true "邮箱"
// @param password formData string true "密码"
func UserLogin(c *gin.Context) {
	req := &UserLoginReq{}
	if err := c.ShouldBind(req); err != nil {
		utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "参数错误", err)
		return
	}
	if !govalidator.IsEmail(req.Email) || !models.EmailIsExists(req.Email) {
		utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "邮箱格式有误或邮箱不存在", nil)
		return
	}
	user, _ := models.FindUserByEmail(req.Email)
	hashed_password := user.Password
	if !utils.CheckPassword(hashed_password, req.Password) {
		utils.Fail(c, http.StatusOK, utils.StatUnauthorized, "密码错误", nil)
		return
	}
	role := user.Identity
	if role == "" {
		role = "user"
	}

	version, err := utils.GetTokenVersion(c, uint(user.UserID))
	if err != nil {
		utils.Fail(c, http.StatusOK, utils.StatInternalError, "内部错误", err)
		return
	}
	version = (version + 1) % utils.TokenVersionMax

	token, err := utils.GenerateToken(utils.JWTSecret(), uint(user.UserID), role, utils.JWTTTL(), version)
	if err != nil {
		utils.Fail(c, http.StatusOK, utils.StatInternalError, "生成token失败", err)
		return
	}

	_, err = utils.IncrTokenVersion(c, uint(user.UserID))
	if err != nil {
		utils.Fail(c, http.StatusOK, utils.StatInternalError, "内部错误", err)
		return
	}

	utils.Success(c, gin.H{
		"token":   token,
		"version": version,
		"user_id": user.UserID,
	})
}

// @Summary 校验 token 是否有效
// @Tags users
// @Produce json
// @Router /user/check_token [post]
// @param token header string false "Bearer token"
// @param token formData string false "token"
func CheckToken(c *gin.Context) {
	token := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	if token == "" {
		token = strings.TrimSpace(c.Query("token"))
	}
	if token == "" {
		req := &CheckTokenReq{}
		_ = c.ShouldBind(req)
		token = strings.TrimSpace(req.Token)
	}
	if token == "" {
		utils.Fail(c, http.StatusUnauthorized, utils.StatInvalidParam, "token不能为空", nil)
		return
	}
	uintDiff := func(a, b uint) uint {
		if a >= b {
			return a - b
		}
		return b - a
	}
	claims, err := utils.CheckToken(token, utils.JWTSecret())

	if err != nil {
		utils.Fail(c, http.StatusUnauthorized, utils.StatUnauthorized, "token无效或已过期", err)
		return
	}

	latest_version, _ := utils.GetTokenVersion(c, claims.UserID)
	diff := uintDiff(latest_version, claims.Version)
	if diff >= utils.LoginDeviceMax {
		utils.Fail(c, http.StatusUnauthorized, utils.StatUnauthorized, "登录设备达到上限", nil)
		return
	}

	exp := int64(0)
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Unix()
	}
	utils.Success(c, gin.H{
		"user_id": claims.UserID,
		"role":    claims.Role,
		"exp":     exp,
	})
}

// @Summary 创建 API Key
// @Description 为当前登录用户创建一个新的 API Key，完整 key 仅返回一次
// @Tags users
// @Produce json
// @Param Authorization header string true "Bearer JWT"
// @Param name formData string true "API Key 名称"
// @Param expires_at formData string false "过期时间 (RFC3339)"
// @Router /user/create_api_key [post]
func CreateAPIKey(c *gin.Context) {
	userID, ok := parseUserID(c)
	if !ok || userID <= 0 {
		utils.Fail(c, http.StatusUnauthorized, utils.StatUnauthorized, "token无效或已过期", nil)
		return
	}

	req := &CreateAPIKeyReq{}
	if err := c.ShouldBind(req); err != nil {
		utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "参数错误", err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "name 不能为空", nil)
		return
	}

	var expiresAt *time.Time
	if raw := strings.TrimSpace(req.ExpiresAt); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "expires_at 格式错误，应为 RFC3339", err)
			return
		}
		parsed = parsed.UTC()
		expiresAt = &parsed
	}

	prefix, fullKey, secretHash, err := generateAPIKeyTokenFn()
	if err != nil {
		utils.Fail(c, http.StatusOK, utils.StatInternalError, "生成 API Key 失败", err)
		return
	}

	apiKey := &models.APIKey{
		APIKeyID:   utils.GenerateID(),
		UserID:     userID,
		Name:       req.Name,
		Prefix:     prefix,
		SecretHash: secretHash,
		Status:     models.APIKeyStatusActive,
		ExpiresAt:  expiresAt,
	}
	if err := createAPIKeyFn(apiKey); err != nil {
		utils.Fail(c, http.StatusOK, utils.StatDatabaseError, "创建 API Key 失败", err)
		return
	}

	utils.Success(c, createAPIKeyResp{
		APIKeyID:  apiKey.APIKeyID,
		Name:      apiKey.Name,
		Prefix:    apiKey.Prefix,
		Key:       fullKey,
		Status:    apiKey.Status,
		CreatedAt: apiKey.CreatedAt.UTC().Format(time.RFC3339Nano),
		ExpiresAt: formatOptionalTime(apiKey.ExpiresAt),
	})
}

// @Summary 获取 API Key 列表
// @Description 返回当前登录用户拥有的全部 API Key 元数据，不返回完整 key
// @Tags users
// @Produce json
// @Param Authorization header string true "Bearer JWT"
// @Router /user/api_key_list [post]
func ListAPIKeys(c *gin.Context) {
	userID, ok := parseUserID(c)
	if !ok || userID <= 0 {
		utils.Fail(c, http.StatusUnauthorized, utils.StatUnauthorized, "token无效或已过期", nil)
		return
	}

	list, err := listAPIKeysByUserFn(userID)
	if err != nil {
		utils.Fail(c, http.StatusOK, utils.StatDatabaseError, "查询 API Key 列表失败", err)
		return
	}

	resp := make([]apiKeyMetaResp, 0, len(list))
	for _, item := range list {
		resp = append(resp, buildAPIKeyMetaResp(item))
	}
	utils.Success(c, gin.H{"list": resp})
}

// @Summary 吊销 API Key
// @Description 吊销当前登录用户的 API Key；已吊销的 key 会按幂等成功处理
// @Tags users
// @Produce json
// @Param Authorization header string true "Bearer JWT"
// @Param api_key_id formData int64 true "API Key ID"
// @Router /user/revoke_api_key [post]
func RevokeAPIKey(c *gin.Context) {
	userID, ok := parseUserID(c)
	if !ok || userID <= 0 {
		utils.Fail(c, http.StatusUnauthorized, utils.StatUnauthorized, "token无效或已过期", nil)
		return
	}

	req := &RevokeAPIKeyReq{}
	if err := c.ShouldBind(req); err != nil {
		utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "参数错误", err)
		return
	}
	if req.APIKeyID <= 0 {
		utils.Fail(c, http.StatusOK, utils.StatInvalidParam, "api_key_id 必须大于 0", nil)
		return
	}

	found, err := revokeAPIKeyByIDFn(req.APIKeyID, userID)
	if err != nil {
		utils.Fail(c, http.StatusOK, utils.StatDatabaseError, "吊销 API Key 失败", err)
		return
	}
	if !found {
		utils.Fail(c, http.StatusOK, utils.StatNotFound, "API Key不存在", nil)
		return
	}
	utils.SuccessMessage(c, "吊销成功")
}

func buildAPIKeyMetaResp(apiKey *models.APIKey) apiKeyMetaResp {
	if apiKey == nil {
		return apiKeyMetaResp{}
	}
	return apiKeyMetaResp{
		APIKeyID:   apiKey.APIKeyID,
		Name:       apiKey.Name,
		Prefix:     apiKey.Prefix,
		Status:     apiKey.Status,
		CreatedAt:  apiKey.CreatedAt.UTC().Format(time.RFC3339Nano),
		ExpiresAt:  formatOptionalTime(apiKey.ExpiresAt),
		LastUsedAt: formatOptionalTime(apiKey.LastUsedAt),
		LastUsedIP: apiKey.LastUsedIP,
	}
}

func formatOptionalTime(v *time.Time) *string {
	if v == nil {
		return nil
	}
	formatted := v.UTC().Format(time.RFC3339Nano)
	return &formatted
}
