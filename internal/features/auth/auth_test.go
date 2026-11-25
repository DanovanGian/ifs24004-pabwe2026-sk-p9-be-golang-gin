package auth

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&User{}))
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
	}
}

func newSvc(t *testing.T) (*Service, *gorm.DB) {
	db := newDB(t)
	return NewService(NewRepository(db), "secret"), db
}

func register(t *testing.T, s *Service, name string) *User {
	t.Helper()
	u, err := s.Register(RegisterRequest{Username: name, Email: name + "@x.com", Password: "secret"})
	require.NoError(t, err)
	return u
}

func TestRegister(t *testing.T) {
	s, _ := newSvc(t)
	u := register(t, s, "budi")
	assert.Equal(t, "member", u.Role)
	assert.NotEqual(t, "secret", u.Password)
	_, err := s.Register(RegisterRequest{Username: "budi", Email: "lain@x.com", Password: "secret"})
	assert.ErrorIs(t, err, ErrUserExists)
}

func TestRegister_Errors(t *testing.T) {
	req := RegisterRequest{Username: "a", Email: "a@x.com", Password: "secret"}

	s, _ := newSvc(t)
	long := req
	long.Password = strings.Repeat("a", 73)
	_, err := s.Register(long)
	assert.Error(t, err)

	s, db := newSvc(t)
	failOn(db, "create")
	_, err = s.Register(req)
	assert.Error(t, err)

	s, db = newSvc(t)
	failOn(db, "query")
	_, err = s.Register(req)
	assert.Error(t, err)
}

func TestLogin(t *testing.T) {
	s, db := newSvc(t)
	register(t, s, "budi")

	res, err := s.Login(LoginRequest{Username: "budi", Password: "secret"})
	require.NoError(t, err)
	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(res.Token, claims, func(*jwt.Token) (interface{}, error) { return []byte("secret"), nil })
	require.NoError(t, err)
	assert.Equal(t, "budi", claims["username"])

	_, err = s.Login(LoginRequest{Username: "nope", Password: "secret"})
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	_, err = s.Login(LoginRequest{Username: "budi", Password: "salah"})
	assert.ErrorIs(t, err, ErrInvalidCredentials)

	failOn(db, "query")
	_, err = s.Login(LoginRequest{Username: "budi", Password: "secret"})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrInvalidCredentials)
}

func TestProfile(t *testing.T) {
	s, db := newSvc(t)
	u := register(t, s, "budi")

	got, err := s.GetProfile(u.ID)
	require.NoError(t, err)
	assert.Equal(t, "budi", got.Username)
	_, err = s.GetProfile(999)
	assert.ErrorIs(t, err, ErrNotFound)

	upd, err := s.UpdateProfile(u.ID, UpdateProfileRequest{Username: "baru", Email: "baru@x.com", Password: "newpass"})
	require.NoError(t, err)
	assert.Equal(t, "baru", upd.Username)
	_, err = s.UpdateProfile(u.ID, UpdateProfileRequest{})
	assert.NoError(t, err)
	_, err = s.UpdateProfile(999, UpdateProfileRequest{})
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = s.UpdateProfile(u.ID, UpdateProfileRequest{Password: strings.Repeat("a", 73)})
	assert.Error(t, err)

	failOn(db, "update")
	_, err = s.UpdateProfile(u.ID, UpdateProfileRequest{Username: "x"})
	assert.Error(t, err)
	failOn(db, "query")
	_, err = s.GetProfile(u.ID)
	assert.Error(t, err)
}

func newRouter(h *Handler, uid uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/logout", h.Logout)
	g := r.Group("/", func(c *gin.Context) { c.Set("user_id", uid) })
	g.GET("/profile", h.GetProfile)
	g.PUT("/profile", h.UpdateProfile)
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
	s, db := newSvc(t)
	h := NewHandler(s)
	r := newRouter(h, 1)
	reg := `{"username":"budi","email":"budi@x.com","password":"secret"}`

	assert.Equal(t, 400, call(r, "POST", "/register", `{}`).Code)
	assert.Equal(t, 201, call(r, "POST", "/register", reg).Code)
	assert.Equal(t, 409, call(r, "POST", "/register", reg).Code)

	assert.Equal(t, 400, call(r, "POST", "/login", `{}`).Code)
	assert.Equal(t, 401, call(r, "POST", "/login", `{"username":"budi","password":"salah"}`).Code)
	assert.Equal(t, 200, call(r, "POST", "/login", `{"username":"budi","password":"secret"}`).Code)
	assert.Equal(t, 200, call(r, "POST", "/logout", "").Code)

	assert.Equal(t, 200, call(r, "GET", "/profile", "").Code)
	assert.Equal(t, 400, call(r, "PUT", "/profile", "{").Code)
	assert.Equal(t, 200, call(r, "PUT", "/profile", `{"username":"baru"}`).Code)

	r404 := newRouter(h, 99)
	assert.Equal(t, 404, call(r404, "GET", "/profile", "").Code)
	assert.Equal(t, 404, call(r404, "PUT", "/profile", `{}`).Code)

	failOn(db, "query")
	assert.Equal(t, 500, call(r, "GET", "/profile", "").Code)
}
