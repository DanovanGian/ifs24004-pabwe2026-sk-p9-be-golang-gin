package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"app/internal/response"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.AbortFail(c, http.StatusUnauthorized, "Token tidak ditemukan")
			return
		}
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), claims,
			func(t *jwt.Token) (interface{}, error) { return []byte(secret), nil },
			jwt.WithValidMethods([]string{"HS256"}))
		if err != nil {
			response.AbortFail(c, http.StatusUnauthorized, "Token tidak valid")
			return
		}
		id, _ := claims["user_id"].(float64)
		username, _ := claims["username"].(string)
		role, _ := claims["role"].(string)
		c.Set("user_id", uint(id))
		c.Set("username", username)
		c.Set("role", role)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		for _, r := range roles {
			if r == role {
				c.Next()
				return
			}
		}
		response.AbortFail(c, http.StatusForbidden, "Akses ditolak")
	}
}

type visitor struct {
	count int
	reset time.Time
}

func RateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	visitors := map[string]*visitor{}
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()
		mu.Lock()
		v, ok := visitors[ip]
		if !ok || now.After(v.reset) {
			v = &visitor{reset: now.Add(window)}
			visitors[ip] = v
		}
		v.count++
		blocked := v.count > limit
		mu.Unlock()
		if blocked {
			response.AbortFail(c, http.StatusTooManyRequests, "Terlalu banyak permintaan")
			return
		}
		c.Next()
	}
}
