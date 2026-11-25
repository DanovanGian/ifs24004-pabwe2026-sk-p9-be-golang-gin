package database

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"app/internal/config"
	"app/internal/features/auth"
	"app/internal/features/post"
)

func PostgresDSN(cfg *config.Config) string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
		cfg.DBHost, cfg.DBUsername, cfg.DBPassword, cfg.DBDatabase, cfg.DBPort, cfg.DBSSLMode,
	)
}

func PostgresDialector(cfg *config.Config) gorm.Dialector {
	return postgres.Open(PostgresDSN(cfg))
}

func Connect(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&auth.User{}, &post.Post{})
}

func Seed(db *gorm.DB, cfg *config.Config) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var user auth.User
	return db.Where(auth.User{Username: cfg.AdminUsername}).
		Attrs(auth.User{
			Email:    cfg.AdminUsername + "@delcom.org",
			Password: string(hash),
			Role:     "admin",
		}).
		FirstOrCreate(&user).Error
}
