package response

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/internal/utils"
)

type Error struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

func SuccessMessage(c *gin.Context, message string) {
	Success(c, gin.H{"message": message})
}

func Fail(c *gin.Context, httpStatus int, code utils.StatCode, message string, err error) {
	c.JSON(httpStatus, Response{
		Success: false,
		Error:   newError(code, message, err),
	})
}

func Abort(c *gin.Context, httpStatus int, code utils.StatCode, message string, err error) {
	c.AbortWithStatusJSON(httpStatus, Response{
		Success: false,
		Error:   newError(code, message, err),
	})
}

func newError(code utils.StatCode, message string, err error) *Error {
	if strings.TrimSpace(message) == "" {
		message = utils.StatText(code)
	}
	apiErr := &Error{
		Message: message,
		Type:    errorType(code),
		Code:    strconv.Itoa(int(code)),
	}
	if err != nil {
		apiErr.Details = err.Error()
	}
	return apiErr
}

func errorType(code utils.StatCode) string {
	switch code {
	case utils.StatInvalidParam, utils.StatNotFound, utils.StatConflict:
		return "invalid_request_error"
	case utils.StatUnauthorized:
		return "authentication_error"
	case utils.StatForbidden:
		return "permission_error"
	case utils.StatInternalError, utils.StatDatabaseError:
		return "server_error"
	default:
		return "unknown_error"
	}
}
