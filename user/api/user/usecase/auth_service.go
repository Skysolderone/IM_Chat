package usecase

import (
	"context"
	"errors"
	"strings"

	"wsim/user/api/user/domain"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}

type TokenGenerator interface {
	Generate(userID uint, username string) (string, error)
}

type AuthService struct {
	repo   domain.UserRepository
	hasher PasswordHasher
	token  TokenGenerator
}

func NewAuthService(repo domain.UserRepository, hasher PasswordHasher, token TokenGenerator) *AuthService {
	return &AuthService{repo: repo, hasher: hasher, token: token}
}

type AuthResult struct {
	UserID uint
	Token  string
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrBadRequest         = errors.New("bad request")
)

func (s *AuthService) Register(ctx context.Context, username, password string) (*AuthResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrBadRequest
	}

	if u, err := s.repo.FindByUsername(ctx, username); err == nil && u != nil {
		return nil, domain.ErrUserAlreadyExists
	} else if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	u := &domain.User{
		Username:     username,
		PasswordHash: hash,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	tk, err := s.token.Generate(u.ID, u.Username)
	if err != nil {
		return nil, err
	}

	return &AuthResult{UserID: u.ID, Token: tk}, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*AuthResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrBadRequest
	}

	u, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := s.hasher.Compare(u.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	tk, err := s.token.Generate(u.ID, u.Username)
	if err != nil {
		return nil, err
	}
	return &AuthResult{UserID: u.ID, Token: tk}, nil
}


