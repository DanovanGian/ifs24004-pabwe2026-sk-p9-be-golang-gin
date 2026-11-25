package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func engine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func okHandler(c *gin.Context) { c.String(200, "ok") }

func do(r *gin.Engine, method, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func token(secret string, claims jwt.MapClaims) string {
	s, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return s
}

func TestCORS(t *testing.T) {
	r := engine()
	r.Use(CORS())
	r.GET("/x", okHandler)
	r.OPTIONS("/x", okHandler)

	w := do(r, "GET", "/x", "")
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, 204, do(r, "OPTIONS", "/x", "").Code)
}

func TestAuth(t *testing.T) {
	r := engine()
	r.GET("/me", Auth("s"), func(c *gin.Context) {
		c.JSON(200, gin.H{"id": c.GetUint("user_id"), "role": c.GetString("role"), "username": c.GetString("username")})
	})

	assert.Equal(t, 401, do(r, "GET", "/me", "").Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer abc").Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer "+token("lain", jwt.MapClaims{"user_id": 1})).Code)
	expired := token("s", jwt.MapClaims{"user_id": 1, "exp": time.Now().Add(-time.Hour).Unix()})
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer "+expired).Code)

	good := token("s", jwt.MapClaims{"user_id": 7, "role": "admin", "username": "u", "exp": time.Now().Add(time.Hour).Unix()})
	w := do(r, "GET", "/me", "Bearer "+good)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"id":7`)
	assert.Contains(t, w.Body.String(), `"role":"admin"`)
}

func TestRequireRole(t *testing.T) {
	r := engine()
	r.GET("/a", func(c *gin.Context) { c.Set("role", c.Query("role")) }, RequireRole("admin", "editor"), okHandler)
	assert.Equal(t, 200, do(r, "GET", "/a?role=admin", "").Code)
	assert.Equal(t, 403, do(r, "GET", "/a?role=member", "").Code)
}

func TestRateLimiter(t *testing.T) {
	r := engine()
	r.Use(RateLimiter(2, 50*time.Millisecond))
	r.GET("/x", okHandler)

	assert.Equal(t, 200, do(r, "GET", "/x", "").Code)
	assert.Equal(t, 200, do(r, "GET", "/x", "").Code)
	assert.Equal(t, 429, do(r, "GET", "/x", "").Code)
	time.Sleep(80 * time.Millisecond)
	assert.Equal(t, 200, do(r, "GET", "/x", "").Code)
}
