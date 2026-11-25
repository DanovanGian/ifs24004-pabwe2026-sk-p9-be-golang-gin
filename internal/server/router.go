package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"app/docs"
	"app/internal/config"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/response"
)

func NewRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS(), middleware.RateLimiter(100000, time.Minute))

	hc := &http.Client{Timeout: 60 * time.Second}
	authH := auth.NewHandler(auth.NewService(auth.NewRepository(db), cfg.AppKey))
	postH := post.NewHandler(post.NewService(post.NewRepository(db)))
	aiH := ai.NewHandler(ai.NewService(
		ai.NewSearchClient(cfg.SearchEngineEndpoint, cfg.OpenAIKey, hc),
		ai.NewChatClient(cfg.OpenAIEndpoint, cfg.OpenAIKey, cfg.OpenAIModel, hc),
	))

	r.GET("/api.json", func(c *gin.Context) {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		c.Data(http.StatusOK, "application/json", docs.GetSwaggerJSON(c.Request.Host, scheme))
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api.json")))

	api := r.Group("/api")

	pub := api.Group("/public")
	pub.GET("/categories", postH.Categories)
	pub.GET("/posts", postH.ListPublic)
	pub.GET("/posts/:slug", postH.GetPublic)

	api.POST("/auth/register", authH.Register)
	api.POST("/auth/login", authH.Login)

	authed := api.Group("", middleware.Auth(cfg.AppKey))
	authed.POST("/auth/logout", authH.Logout)
	authed.GET("/auth/profile", authH.GetProfile)
	authed.PUT("/auth/profile", authH.UpdateProfile)

	authed.GET("/posts", postH.ListMine)
	authed.POST("/posts", postH.Create)
	authed.GET("/posts/stats", postH.Stats)
	authed.POST("/posts/ai-generate", aiH.Generate)
	authed.GET("/posts/:id", postH.GetMine)
	authed.PUT("/posts/:id", postH.Update)
	authed.PATCH("/posts/:id/toggle-publish", postH.Toggle)
	authed.DELETE("/posts/:id", postH.Delete)

	admin := authed.Group("/admin", middleware.RequireRole("admin"))
	admin.GET("/users", postH.AdminUsers)
	admin.GET("/posts", postH.AdminPosts)
	admin.GET("/users/:id/posts", postH.AdminUserPosts)

	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, "Endpoint not found")
	})
	return r
}
