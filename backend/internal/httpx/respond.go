package httpx

import (
	"github.com/gin-gonic/gin"
)

// OK writes {"status":"success","data":<payload>}. Pass nil for no data.
func OK(c *gin.Context, code int, data any) {
	body := gin.H{"status": "success"}
	if data != nil {
		body["data"] = data
	}
	c.JSON(code, body)
}

// OKMeta writes {"status":"success","data":<payload>,"meta":<meta>}.
// meta is omitted when nil (forward-compat slot for pagination etc.).
func OKMeta(c *gin.Context, code int, data any, meta any) {
	body := gin.H{"status": "success"}
	if data != nil {
		body["data"] = data
	}
	if meta != nil {
		body["meta"] = meta
	}
	c.JSON(code, body)
}

// Msg is sugar for OK(c, code, gin.H{"message": msg}).
func Msg(c *gin.Context, code int, msg string) {
	OK(c, code, gin.H{"message": msg})
}

// Err writes {"status":"error","error":<msg>}.
func Err(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"status": "error", "error": msg})
}

// Abort is Err + c.Abort() for middleware / early returns.
func Abort(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(code, gin.H{"status": "error", "error": msg})
}
