package service

import (
	"go-file-server/internal/repository"

	gonetjwt "github.com/leonkhoo123/gonet-auth/jwt"
	"golang.org/x/crypto/bcrypt"
)

const BcryptCost = bcrypt.DefaultCost + 2 // 12 - standard cost for all password/PIN hashing

type SharingService struct {
	ShareRepo repository.SharingRepository
	BaseDir   string
	JWT       *gonetjwt.Service
}

func NewSharingService(repo repository.SharingRepository, baseDir string, jwtSvc *gonetjwt.Service) *SharingService {
	return &SharingService{ShareRepo: repo, BaseDir: baseDir, JWT: jwtSvc}
}
