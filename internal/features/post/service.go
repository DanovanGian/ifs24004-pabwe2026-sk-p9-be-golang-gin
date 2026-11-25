package post

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

var Categories = []string{"Teknologi", "Pendidikan", "Gaya Hidup", "Bisnis", "Opini", "Umum"}

var ErrNotFound = errors.New("artikel tidak ditemukan")

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(s string) string {
	slug := strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if slug == "" {
		return "artikel"
	}
	return slug
}

// ReadingTime menghitung estimasi menit baca (200 kata per menit, minimal 1).
func ReadingTime(content string) int {
	minutes := (len(strings.Fields(content)) + 199) / 200
	if minutes < 1 {
		return 1
	}
	return minutes
}

func mapErr(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func withReadingTime(posts []Post) {
	for i := range posts {
		posts[i].ReadingTime = ReadingTime(posts[i].Content)
	}
}

func setPublished(p *Post, v bool) {
	if v && !p.IsPublished {
		now := time.Now()
		p.PublishedAt = &now
	}
	if !v {
		p.PublishedAt = nil
	}
	p.IsPublished = v
}

func apply(p *Post, req PostRequest) {
	p.Title = req.Title
	p.Category = req.Category
	if p.Category == "" {
		p.Category = "Umum"
	}
	p.Excerpt = req.Excerpt
	p.Content = req.Content
	p.CoverImagePath = req.CoverImagePath
	setPublished(p, req.IsPublished)
}

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) uniqueSlug(title string) (string, error) {
	base := Slugify(title)
	slug := base
	for i := 2; ; i++ {
		exists, err := s.repo.SlugExists(slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func (s *Service) Create(userID uint, req PostRequest) (*Post, error) {
	slug, err := s.uniqueSlug(req.Title)
	if err != nil {
		return nil, err
	}
	p := &Post{UserID: userID, Slug: slug}
	apply(p, req)
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	p.ReadingTime = ReadingTime(p.Content)
	return p, nil
}

func (s *Service) Update(id, userID uint, req PostRequest) (*Post, error) {
	p, err := s.repo.FindOwned(id, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	apply(p, req)
	if err := s.repo.Save(p); err != nil {
		return nil, err
	}
	p.ReadingTime = ReadingTime(p.Content)
	return p, nil
}

func (s *Service) Toggle(id, userID uint) (*Post, error) {
	p, err := s.repo.FindOwned(id, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	setPublished(p, !p.IsPublished)
	if err := s.repo.Save(p); err != nil {
		return nil, err
	}
	p.ReadingTime = ReadingTime(p.Content)
	return p, nil
}

func (s *Service) Delete(id, userID uint) error {
	rows, err := s.repo.Delete(id, userID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) GetMine(id, userID uint) (*Post, error) {
	p, err := s.repo.FindOwned(id, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	p.ReadingTime = ReadingTime(p.Content)
	return p, nil
}

func (s *Service) ListMine(userID uint) ([]Post, error) {
	posts, err := s.repo.ListByUser(userID)
	withReadingTime(posts)
	return posts, err
}

func (s *Service) ListPublic(q, category string) ([]Post, error) {
	if category == "Semua" {
		category = ""
	}
	posts, err := s.repo.ListPublic(q, category)
	withReadingTime(posts)
	return posts, err
}

func (s *Service) GetPublic(slug string) (*Post, error) {
	p, err := s.repo.FindPublicBySlug(slug)
	if err != nil {
		return nil, mapErr(err)
	}
	p.ReadingTime = ReadingTime(p.Content)
	return p, nil
}

func (s *Service) Stats(userID uint) (Stats, error) { return s.repo.Stats(userID) }

func (s *Service) AdminUsers() ([]UserSummary, error) { return s.repo.UserSummaries() }

func (s *Service) AdminPosts() ([]Post, error) {
	posts, err := s.repo.ListAll()
	withReadingTime(posts)
	return posts, err
}
