package auth

import "gorm.io/gorm"

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(u *User) error { return r.db.Create(u).Error }

func (r *Repository) Update(u *User) error { return r.db.Save(u).Error }

func (r *Repository) FindByID(id uint) (*User, error) {
	var u User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) FindByUsername(username string) (*User, error) {
	var u User
	if err := r.db.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) Exists(username, email string) (bool, error) {
	var n int64
	err := r.db.Model(&User{}).Where("username = ? OR email = ?", username, email).Count(&n).Error
	return n > 0, err
}
