package service

import "go-file-server/internal/repository"

type SharingService struct {
	ShareRepo repository.SharingRepository
	BaseDir   string
}

func NewSharingService(repo repository.SharingRepository, baseDir string) *SharingService {
	return &SharingService{ShareRepo: repo, BaseDir: baseDir}
}
