package validator

  import (
  	"net/http"

  	"github.com/gin-gonic/gin"
  )

  // Validatable is implemented by all request structs that have a Validate method.
  type Validatable interface {
  	Validate() ValidationErrors
  }

  // Bind is a Gin middleware factory that:
  //  1. Binds the incoming JSON body into the provided struct pointer.
  //  2. Calls .Validate() on it.
  //  3. If there are errors, it aborts with 400 and a structured error response.
  //  4. If everything is valid, it stores the struct in the Gin context under the
  //     key "validatedBody" and calls c.Next().
  //
  // Usage in route registration:
  //
  //	import v "github.com/geocore-next/backend/pkg/validator"
  //
  //	router.POST("/auth/register", v.Bind(&v.RegisterRequest{}), handler.Register)
  //
  // In the handler, retrieve the validated body:
  //
  //	body := c.MustGet("validatedBody").(*v.RegisterRequest)
  func Bind(target Validatable) gin.HandlerFunc {
  	return func(c *gin.Context) {
  		if err := c.ShouldBindJSON(target); err != nil {
  			c.AbortWithStatusJSON(http.StatusBadRequest, errorResponse(
  				[]FieldError{{Field: "_body", Message: "invalid JSON: " + err.Error()}},
  			))
  			return
  		}

  		if errs := target.Validate(); errs.Any() {
  			c.AbortWithStatusJSON(http.StatusBadRequest, errorResponse(errs))
  			return
  		}

  		c.Set("validatedBody", target)
  		c.Next()
  	}
  }

  // errorResponse builds the standard validation error JSON body.
  //
  //	{
  //	  "error":   "validation_failed",
  //	  "message": "Validation errors occurred",
  //	  "details": [
  //	    {"field": "email", "message": "must be a valid email address"}
  //	  ]
  //	}
  func errorResponse(errs ValidationErrors) gin.H {
  	details := make([]gin.H, len(errs))
  	for i, e := range errs {
  		details[i] = gin.H{"field": e.Field, "message": e.Message}
  	}
  	return gin.H{
  		"error":   "validation_failed",
  		"message": "Validation errors occurred",
  		"details": details,
  	}
  }
  