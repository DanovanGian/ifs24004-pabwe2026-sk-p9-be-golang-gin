package database

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"app/internal/config"
	"app/internal/features/auth"
)

func memDB(t *testing.T) *gorm.DB {
	db, err := Connect(sqlite.Open(":memory:"))
	assert.NoError(t, err)
	return db
}

func TestPostgresDSN(t *testing.T) {
	cfg := &config.Config{DBHost: "h", DBUsername: "u", DBPassword: "p", DBDatabase: "d", DBPort: "5432", DBSSLMode: "disable"}
	dsn := PostgresDSN(cfg)
	assert.Contains(t, dsn, "host=h")
	assert.Contains(t, dsn, "dbname=d")
	assert.NotNil(t, PostgresDialector(cfg))
}

func TestConnect(t *testing.T) {
	assert.NotNil(t, memDB(t))
}

func TestConnect_Error(t *testing.T) {
	db, err := Connect(sqlite.Open("/nonexistent-dir/x.db"))
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestMigrate(t *testing.T) {
	db := memDB(t)
	assert.NoError(t, Migrate(db))
	assert.True(t, db.Migrator().HasTable("users"))
	assert.True(t, db.Migrator().HasTable("posts"))
}

func TestSeed(t *testing.T) {
	db := memDB(t)
	assert.NoError(t, Migrate(db))
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: "admin"}

	assert.NoError(t, Seed(db, cfg))
	assert.NoError(t, Seed(db, cfg)) // idempotent

	var count int64
	db.Model(&auth.User{}).Where("username = ?", "admin").Count(&count)
	assert.Equal(t, int64(1), count)

	var u auth.User
	db.Where("username = ?", "admin").First(&u)
	assert.Equal(t, "admin", u.Role)
}

func TestSeed_HashError(t *testing.T) {
	db := memDB(t)
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: string(make([]byte, 100))}
	assert.Error(t, Seed(db, cfg))
}

func TestSeed_DBError(t *testing.T) {
	db := memDB(t) // tanpa Migrate -> tabel users tidak ada
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: "admin"}
	assert.Error(t, Seed(db, cfg))
}