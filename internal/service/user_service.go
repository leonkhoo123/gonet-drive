package service

import "go-file-server/internal/repository"

type UserService struct {
	UserRepo  repository.UserRepository
	TokenRepo repository.RefreshTokenRepository
}

func NewUserService(repo repository.UserRepository, tokenRepo repository.RefreshTokenRepository) *UserService {
	return &UserService{
		UserRepo:  repo,
		TokenRepo: tokenRepo,
	}
}
