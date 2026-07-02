package service

import (
	"go-file-server/internal/repository"

	"github.com/leonkhoo123/gonet-auth/auth"
)

type UserService struct {
	UserRepo     repository.UserRepository
	TokenRepo    repository.RefreshTokenRepository
	AuthInstance *auth.Auth
}

func NewUserService(repo repository.UserRepository, tokenRepo repository.RefreshTokenRepository, authInstance *auth.Auth) *UserService {
	return &UserService{
		UserRepo:     repo,
		TokenRepo:    tokenRepo,
		AuthInstance: authInstance,
	}
}
