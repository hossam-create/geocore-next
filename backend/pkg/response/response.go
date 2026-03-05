package response

import "github.com/gin-gonic/gin"

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

func OK(c *gin.Context, data interface{})                     { c.JSON(200, R{true, data, "", nil}) }
func OKMeta(c *gin.Context, data, meta interface{})           { c.JSON(200, R{true, data, "", meta}) }
func Created(c *gin.Context, data interface{})                { c.JSON(201, R{true, data, "", nil}) }
func BadRequest(c *gin.Context, err string)                   { c.JSON(400, R{false, nil, err, nil}) }
func Unauthorized(c *gin.Context)                             { c.JSON(401, R{false, nil, "Unauthorized", nil}) }
func Forbidden(c *gin.Context)                                { c.JSON(403, R{false, nil, "Forbidden", nil}) }
func NotFound(c *gin.Context, r string)                       { c.JSON(404, R{false, nil, r + " not found", nil}) }
func InternalError(c *gin.Context, _ error)                   { c.JSON(500, R{false, nil, "Internal server error", nil}) }
