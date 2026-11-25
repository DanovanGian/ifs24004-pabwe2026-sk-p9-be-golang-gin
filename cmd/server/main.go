package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"
)

func main() {
	cfg := config.LoadConfig()
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Connect(database.PostgresDialector(cfg))
	if err != nil {
		log.Fatalf("koneksi database gagal: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrasi gagal: %v", err)
	}
	if err := database.Seed(db, cfg); err != nil {
		log.Fatalf("seeder gagal: %v", err)
	}

	r := server.NewRouter(db, cfg)
	log.Printf("Server berjalan di :%s", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}
