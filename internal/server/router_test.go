package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"app/internal/config"
	"app/internal/database"
)

func setup(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/search") {
			fmt.Fprint(w, `{"results":[{"title":"a","url":"http://a","snippet":"s"}]}`)
			return
		}
		content, _ := json.Marshal(map[string]string{"title": "T", "category": "Teknologi", "excerpt": "e", "content": "c"})
		resp, _ := json.Marshal(map[string]interface{}{
			"choices": []map[string]interface{}{{"message": map[string]string{"content": string(content)}}},
		})
		_, _ = w.Write(resp)
	}))
	t.Cleanup(srv.Close)

	db, err := database.Connect(sqlite.Open(":memory:"))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.Migrate(db))

	cfg := &config.Config{
		AppKey: "k", AdminUsername: "admin", AdminPassword: "admin",
		SearchEngineEndpoint: srv.URL + "/search", OpenAIEndpoint: srv.URL, OpenAIModel: "m",
	}
	require.NoError(t, database.Seed(db, cfg))
	return NewRouter(db, cfg)
}

func call(r *gin.Engine, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func login(t *testing.T, r *gin.Engine, user, pass string) string {
	t.Helper()
	w := call(r, "POST", "/api/auth/login", fmt.Sprintf(`{"username":%q,"password":%q}`, user, pass), "")
	require.Equal(t, 200, w.Code)
	var b struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
	return b.Data.Token
}

func TestFullFlow(t *testing.T) {
	r := setup(t)

	assert.Equal(t, 201, call(r, "POST", "/api/auth/register", `{"username":"budi","email":"budi@x.com","password":"secret"}`, "").Code)
	tok := login(t, r, "budi", "secret")
	adm := login(t, r, "admin", "admin")

	assert.Equal(t, 401, call(r, "GET", "/api/posts", "", "").Code)
	assert.Equal(t, 200, call(r, "GET", "/api/auth/profile", "", tok).Code)

	assert.Equal(t, 201, call(r, "POST", "/api/posts", `{"title":"Belajar Go","content":"isi","is_published":true}`, tok).Code)
	assert.Equal(t, 200, call(r, "GET", "/api/public/posts?q=belajar", "", "").Code)
	assert.Equal(t, 200, call(r, "GET", "/api/public/posts/belajar-go", "", "").Code)
	assert.Equal(t, 200, call(r, "GET", "/api/posts/stats", "", tok).Code)
	assert.Equal(t, 200, call(r, "PATCH", "/api/posts/1/toggle-publish", "", tok).Code)

	assert.Equal(t, 200, call(r, "POST", "/api/posts/ai-generate", `{"topic":"go","include_web_search":true}`, tok).Code)

	assert.Equal(t, 403, call(r, "GET", "/api/admin/users", "", tok).Code)
	assert.Equal(t, 200, call(r, "GET", "/api/admin/users", "", adm).Code)
	assert.Equal(t, 200, call(r, "GET", "/api/admin/posts", "", adm).Code)
	assert.Equal(t, 200, call(r, "GET", "/api/admin/users/2/posts", "", adm).Code)

	assert.Equal(t, 200, call(r, "DELETE", "/api/posts/1", "", tok).Code)
}

func TestDocsAndNoRoute(t *testing.T) {
	r := setup(t)
	assert.Equal(t, 404, call(r, "GET", "/tidak-ada", "", "").Code)
	assert.Equal(t, 200, call(r, "GET", "/api.json", "", "").Code)
	assert.Equal(t, 200, call(r, "GET", "/swagger/index.html", "", "").Code)

	req := httptest.NewRequest("GET", "/api.json", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Body.String(), "https://")
}
