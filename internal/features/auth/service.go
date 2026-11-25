package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserExists         = errors.New("username atau email sudah terdaftar")
	ErrInvalidCredentials = errors.New("username atau password salah")
	ErrNotFound           = errors.New("pengguna tidak ditemukan")
)

type Service struct {
	repo   *Repository
	secret string
}

func NewService(repo *Repository, secret string) *Service {
	return &Service{repo: repo, secret: secret}
}

func (s *Service) Register(req RegisterRequest) (*User, error) {
	exists, err := s.repo.Exists(req.Username, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &User{Username: req.Username, Email: req.Email, Password: string(hash), Role: "member"}
	if err := s.repo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) Login(req LoginRequest) (*LoginResponse, error) {
	user, err := s.repo.FindByUsername(req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return &LoginResponse{Token: s.GenerateToken(user), User: *user}, nil
}

func (s *Service) GenerateToken(u *User) string {
	claims := jwt.MapClaims{
		"user_id":  u.ID,
		"username": u.Username,
		"role":     u.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.secret))
	return token
}

func (s *Service) find(id uint) (*User, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *Service) GetProfile(id uint) (*User, error) { return s.find(id) }

func (s *Service) UpdateProfile(id uint, req UpdateProfileRequest) (*User, error) {
	user, err := s.find(id)
	if err != nil {
		return nil, err
	}
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	taken, err := s.repo.ExistsOther(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrUserExists
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.Password = string(hash)
	}
	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}
