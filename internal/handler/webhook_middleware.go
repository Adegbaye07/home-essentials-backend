package handler

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

const rawBodyKey = "rawBody"

func PaystackWebhookRawBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		c.Set(rawBodyKey, raw)
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		c.Next()
	}
}

func rawBodyFromContext(c *gin.Context) ([]byte, bool) {
	v, ok := c.Get(rawBodyKey)
	if !ok {
		return nil, false
	}
	b, ok := v.([]byte)
	return b, ok
}
