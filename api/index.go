package handler

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"
)

var (
	once    sync.Once
	engine  *gin.Engine
	initErr error
)

func setup() {
	cfg := config.LoadConfig()
	gin.SetMode(gin.ReleaseMode)

	db, err := database.Connect(database.PostgresDialector(cfg))
	if err != nil {
		initErr = err
		return
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(3)
		sqlDB.SetMaxIdleConns(1)
	}
	if err := database.Migrate(db); err != nil {
		initErr = err
		return
	}
	if err := database.Seed(db, cfg); err != nil {
		initErr = err
		return
	}
	engine = server.NewRouter(db, cfg)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(setup)
	if initErr != nil {
		http.Error(w, "server initialization failed: "+initErr.Error(), http.StatusInternalServerError)
		return
	}
	engine.ServeHTTP(w, r)
}
