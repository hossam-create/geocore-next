package response

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

type R struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type Meta struct {
	Total   int64 `json:"total"`
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Pages   int64 `json:"pages"`
}

func OK(c *gin.Context, data interface{})           { c.JSON(200, R{true, data, "", nil}) }
func OKMeta(c *gin.Context, data, meta interface{}) { c.JSON(200, R{true, data, "", meta}) }
func Created(c *gin.Context, data interface{})      { c.JSON(201, R{true, data, "", nil}) }
func BadRequest(c *gin.Context, err string)         { c.JSON(400, R{false, nil, err, nil}) }
func Conflict(c *gin.Context, msg string)           { c.JSON(409, R{false, nil, msg, nil}) }
func Unauthorized(c *gin.Context)                   { c.JSON(401, R{false, nil, "Unauthorized", nil}) }
func Forbidden(c *gin.Context)                      { c.JSON(403, R{false, nil, "Forbidden", nil}) }
func NotFound(c *gin.Context, r string)             { c.JSON(404, R{false, nil, r + " not found", nil}) }
func RateLimited(c *gin.Context, msg string)        { c.JSON(429, R{false, nil, msg, nil}) }
func InternalError(c *gin.Context, err error) {
	slog.Error("internal_error",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"error", err.Error(),
	)
	c.JSON(500, R{false, nil, "Internal server error", nil})
}
