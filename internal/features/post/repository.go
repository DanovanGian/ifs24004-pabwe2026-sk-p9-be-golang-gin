package post

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ListPublic(q, category string) ([]Post, error) {
	tx := r.db.Preload("User").Where("is_published = ?", true)
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		tx = tx.Where("LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ? OR LOWER(content) LIKE ?", like, like, like)
	}
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	var posts []Post
	err := tx.Order("published_at DESC").Find(&posts).Error
	return posts, err
}

func (r *Repository) FindPublicBySlug(slug string) (*Post, error) {
	var p Post
	err := r.db.Preload("User").Where("slug = ? AND is_published = ?", slug, true).First(&p).Error
	return &p, err
}

func (r *Repository) ListByUser(userID uint) ([]Post, error) {
	var posts []Post
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&posts).Error
	return posts, err
}

func (r *Repository) ListAll() ([]Post, error) {
	var posts []Post
	err := r.db.Preload("User").Order("created_at DESC").Find(&posts).Error
	return posts, err
}

func (r *Repository) FindOwned(id, userID uint) (*Post, error) {
	var p Post
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&p).Error
	return &p, err
}

func (r *Repository) Create(p *Post) error {
	return r.db.Omit(clause.Associations).Create(p).Error
}

func (r *Repository) Save(p *Post) error {
	return r.db.Omit(clause.Associations).Save(p).Error
}

func (r *Repository) Delete(id, userID uint) (int64, error) {
	res := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&Post{})
	return res.RowsAffected, res.Error
}

func (r *Repository) SlugExists(slug string) (bool, error) {
	var n int64
	err := r.db.Model(&Post{}).Where("slug = ?", slug).Count(&n).Error
	return n > 0, err
}

func (r *Repository) Stats(userID uint) (Stats, error) {
	var s Stats
	err := r.db.Model(&Post{}).
		Select("COUNT(*) AS total, COALESCE(SUM(CASE WHEN is_published THEN 1 ELSE 0 END), 0) AS published").
		Where("user_id = ?", userID).Scan(&s).Error
	s.Draft = s.Total - s.Published
	return s, err
}

func (r *Repository) UserSummaries() ([]UserSummary, error) {
	var list []UserSummary
	err := r.db.Raw(`
		SELECT u.id, u.username, u.email, u.role,
		       COUNT(p.id) AS total_posts,
		       COALESCE(SUM(CASE WHEN p.is_published THEN 1 ELSE 0 END), 0) AS total_published
		FROM users u
		LEFT JOIN posts p ON p.user_id = u.id
		GROUP BY u.id, u.username, u.email, u.role
		ORDER BY u.id`).Scan(&list).Error
	return list, err
}
