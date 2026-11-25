package post

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"app/internal/features/auth"
)

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&auth.User{}, &Post{}))
	require.NoError(t, db.Create(&auth.User{Username: "u1", Email: "u1@x.com", Password: "x", Role: "member"}).Error)
	return db
}

func failOn(db *gorm.DB, op string) {
	cb := func(d *gorm.DB) { _ = d.AddError(errors.New("boom")) }
	switch op {
	case "query":
		_ = db.Callback().Query().Before("gorm:query").Register("test:fail", cb)
	case "create":
		_ = db.Callback().Create().Before("gorm:create").Register("test:fail", cb)
	case "update":
		_ = db.Callback().Update().Before("gorm:update").Register("test:fail", cb)
	case "delete":
		_ = db.Callback().Delete().Before("gorm:delete").Register("test:fail", cb)
	}
}

func TestSlugify(t *testing.T) {
	assert.Equal(t, "halo-dunia", Slugify("Halo Dunia!"))
	assert.Equal(t, "go-gin-2026", Slugify("Go & Gin 2026"))
	assert.Equal(t, "artikel", Slugify("  --  "))
}

func TestReadingTime(t *testing.T) {
	assert.Equal(t, 1, ReadingTime(""))
	assert.Equal(t, 1, ReadingTime(strings.Repeat("kata ", 200)))
	assert.Equal(t, 2, ReadingTime(strings.Repeat("kata ", 201)))
}

func TestService_Errors(t *testing.T) {
	db := newDB(t)
	s := NewService(NewRepository(db))
	req := PostRequest{Title: "Judul"}

	p, err := s.Create(1, req)
	require.NoError(t, err)
	p2, err := s.Create(1, req)
	require.NoError(t, err)
	assert.Equal(t, "judul-2", p2.Slug)

	failOn(db, "update")
	_, err = s.Update(p.ID, 1, req)
	assert.Error(t, err)
	_, err = s.Toggle(p.ID, 1)
	assert.Error(t, err)

	failOn(db, "delete")
	assert.Error(t, s.Delete(p.ID, 1))

	failOn(db, "create")
	_, err = s.Create(1, PostRequest{Title: "Baru"})
	assert.Error(t, err)

	failOn(db, "query")
	_, err = s.Create(1, req)
	assert.Error(t, err)
	_, err = s.GetMine(p.ID, 1)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)
}

func newRouter(h *Handler, uid uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/public/categories", h.Categories)
	r.GET("/public/posts", h.ListPublic)
	r.GET("/public/posts/:slug", h.GetPublic)
	g := r.Group("/", func(c *gin.Context) { c.Set("user_id", uid) })
	g.GET("/posts", h.ListMine)
	g.POST("/posts", h.Create)
	g.GET("/posts/stats", h.Stats)
	g.GET("/posts/:id", h.GetMine)
	g.PUT("/posts/:id", h.Update)
	g.PATCH("/posts/:id/toggle-publish", h.Toggle)
	g.DELETE("/posts/:id", h.Delete)
	g.GET("/admin/users", h.AdminUsers)
	g.GET("/admin/posts", h.AdminPosts)
	g.GET("/admin/users/:id/posts", h.AdminUserPosts)
	return r
}

func call(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandlers(t *testing.T) {
	db := newDB(t)
	h := NewHandler(NewService(NewRepository(db)))
	r := newRouter(h, 1)

	assert.Equal(t, 400, call(r, "POST", "/posts", "{").Code)
	assert.Equal(t, 400, call(r, "POST", "/posts", `{}`).Code)
	assert.Equal(t, 201, call(r, "POST", "/posts", `{"title":"Halo Dunia","content":"isi artikel","is_published":true}`).Code)

	assert.Equal(t, 200, call(r, "GET", "/public/categories", "").Code)
	assert.Equal(t, 200, call(r, "GET", "/public/posts?q=halo&category=Umum", "").Code)
	assert.Equal(t, 200, call(r, "GET", "/public/posts?category=Semua", "").Code)
	assert.Equal(t, 200, call(r, "GET", "/public/posts/halo-dunia", "").Code)
	assert.Equal(t, 404, call(r, "GET", "/public/posts/tidak-ada", "").Code)

	assert.Equal(t, 200, call(r, "GET", "/posts", "").Code)
	assert.Equal(t, 200, call(r, "GET", "/posts/stats", "").Code)
	assert.Equal(t, 200, call(r, "GET", "/posts/1", "").Code)
	assert.Equal(t, 400, call(r, "GET", "/posts/abc", "").Code)
	assert.Equal(t, 404, call(r, "GET", "/posts/99", "").Code)

	assert.Equal(t, 200, call(r, "PUT", "/posts/1", `{"title":"Halo Baru","is_published":true}`).Code)
	assert.Equal(t, 400, call(r, "PUT", "/posts/abc", `{"title":"x"}`).Code)
	assert.Equal(t, 400, call(r, "PUT", "/posts/1", "{").Code)
	assert.Equal(t, 404, call(r, "PUT", "/posts/99", `{"title":"x"}`).Code)

	assert.Equal(t, 200, call(r, "PATCH", "/posts/1/toggle-publish", "").Code)
	assert.Equal(t, 200, call(r, "PATCH", "/posts/1/toggle-publish", "").Code)
	assert.Equal(t, 400, call(r, "PATCH", "/posts/abc/toggle-publish", "").Code)
	assert.Equal(t, 404, call(r, "PATCH", "/posts/99/toggle-publish", "").Code)

	assert.Equal(t, 200, call(r, "GET", "/admin/users", "").Code)
	assert.Equal(t, 200, call(r, "GET", "/admin/posts", "").Code)
	assert.Equal(t, 200, call(r, "GET", "/admin/users/1/posts", "").Code)
	assert.Equal(t, 400, call(r, "GET", "/admin/users/abc/posts", "").Code)

	assert.Equal(t, 200, call(r, "DELETE", "/posts/1", "").Code)
	assert.Equal(t, 400, call(r, "DELETE", "/posts/abc", "").Code)
	assert.Equal(t, 404, call(r, "DELETE", "/posts/1", "").Code)

	failOn(db, "query")
	assert.Equal(t, 500, call(r, "GET", "/posts", "").Code)
}
