package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func run(t *testing.T, fn func(c *gin.Context)) (int, map[string]interface{}) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	fn(c)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w.Code, body
}

func TestSuccess(t *testing.T) {
	code, body := run(t, func(c *gin.Context) { Success(c, 200, "ok", gin.H{"a": 1}) })
	assert.Equal(t, 200, code)
	assert.Equal(t, "success", body["status"])
	assert.NotNil(t, body["data"])
}

func TestSuccess_NoData(t *testing.T) {
	_, body := run(t, func(c *gin.Context) { Success(c, 200, "ok", nil) })
	assert.NotContains(t, body, "data")
}

func TestFail(t *testing.T) {
	code, body := run(t, func(c *gin.Context) { Fail(c, 400, "x") })
	assert.Equal(t, 400, code)
	assert.Equal(t, "fail", body["status"])
}

func TestError(t *testing.T) {
	code, body := run(t, func(c *gin.Context) { Error(c, 500, "x") })
	assert.Equal(t, 500, code)
	assert.Equal(t, "error", body["status"])
}

func TestAbortFail(t *testing.T) {
	code, body := run(t, func(c *gin.Context) { AbortFail(c, 401, "x") })
	assert.Equal(t, 401, code)
	assert.Equal(t, "fail", body["status"])
}

func TestAbortError(t *testing.T) {
	code, body := run(t, func(c *gin.Context) { AbortError(c, 500, "x") })
	assert.Equal(t, 500, code)
	assert.Equal(t, "error", body["status"])
}
