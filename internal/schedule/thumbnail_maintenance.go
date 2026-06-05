package schedule

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-file-server/internal/logger"
	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/state"
	"go-file-server/internal/storage"
)

const idleThreshold = 10 * time.Second

type fileNeedingThumbnail struct {
	fullPath  string
	thumbPath string
	hash      string
	isVideo   bool
}

func StartThumbnailMaintenanceScheduler(rootPath string, repo repository.ThumbnailRepository) {
	go func() {
		for {
			delay := untilNext(time.Now(), 4, 30)
			logger.L.Debug("thumbnail maintenance scheduled", "delay", delay.Round(time.Second))
			time.Sleep(delay)

			deleted, generated, err := maintainThumbnails(rootPath, repo)
			if err != nil {
				logger.L.Error("thumbnail maintenance failed", "err", err)
			} else {
				logger.L.Info("thumbnail maintenance complete", "orphaned_removed", deleted, "pre_generated", generated)
			}
		}
	}()
}

func maintainThumbnails(rootPath string, repo repository.ThumbnailRepository) (deleted, generated int, err error) {
	defer func() {
		logger.L.Debug("storage calibration re-scanning")
		storage.InitStorageManager(rootPath)
	}()

	thumbDir := filepath.Join(rootPath, ".cloud_reserve", ".thumbnails")
	if _, e := os.Stat(thumbDir); os.IsNotExist(e) {
		return 0, 0, nil
	}

	if err := repo.MarkAllInactive(); err != nil {
		return 0, 0, fmt.Errorf("mark inactive: %w", err)
	}

	var pending []fileNeedingThumbnail

	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if path != filepath.Join(rootPath, ".cloud_reserve") && strings.HasPrefix(path, filepath.Join(rootPath, ".cloud_reserve")+string(filepath.Separator)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}

		hash := md5.Sum([]byte(path))
		hashStr := hex.EncodeToString(hash[:])

		thumbPath := filepath.Join(thumbDir, hashStr+".webp")

		switch {
		case isVideoFile(info.Name()):
			thumbStat, e := os.Stat(thumbPath)
			if e == nil && thumbStat.ModTime().After(info.ModTime()) {
				if uerr := repo.Upsert(hashStr, path, true); uerr != nil {
					logger.L.Warn("thumbnail upsert error during maintenance", "path", path, "err", uerr)
				}
			} else {
				pending = append(pending, fileNeedingThumbnail{
					fullPath: path, thumbPath: thumbPath, hash: hashStr, isVideo: true,
				})
			}
		case isPhotoFile(info.Name()):
			thumbStat, e := os.Stat(thumbPath)
			if e == nil && thumbStat.ModTime().After(info.ModTime()) {
				if uerr := repo.Upsert(hashStr, path, false); uerr != nil {
					logger.L.Warn("thumbnail upsert error during maintenance", "path", path, "err", uerr)
				}
			} else {
				pending = append(pending, fileNeedingThumbnail{
					fullPath: path, thumbPath: thumbPath, hash: hashStr, isVideo: false,
				})
			}
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}

	generated = preGenerateThumbnails(pending, repo, idleThreshold, 5*time.Second)

	orphanHashes, err := repo.DeleteInactive()
	if err != nil {
		return 0, generated, fmt.Errorf("delete inactive: %w", err)
	}

	for _, h := range orphanHashes {
		thumbFile := filepath.Join(thumbDir, h+".webp")
		if err := os.Remove(thumbFile); err != nil && !os.IsNotExist(err) {
			logger.L.Warn("failed to remove orphan thumbnail", "path", thumbFile, "err", err)
		} else if err == nil {
			deleted++
		}
	}

	return deleted, generated, nil
}

func preGenerateThumbnails(pending []fileNeedingThumbnail, repo repository.ThumbnailRepository, idleThreshold, idleSleepDuration time.Duration) int {
	if len(pending) == 0 {
		return 0
	}

	generated := 0
	for _, f := range pending {
		if !state.IsIdle(idleThreshold) {
			logger.L.Info("thumbnail maintenance pausing, API activity detected")
			stuckAt := time.Now()
			for !state.IsIdle(idleThreshold) {
				if time.Since(stuckAt) > 10*time.Minute {
					logger.L.Warn("thumbnail maintenance stuck waiting for idle, API has been active for 10+ minutes", "elapsed", time.Since(stuckAt).Round(time.Second))
					stuckAt = time.Now()
				}
				time.Sleep(idleSleepDuration)
			}
			logger.L.Info("thumbnail maintenance resuming")
		}

		if err := generateThumbnail(f); err != nil {
			logger.L.Error("failed to generate thumbnail", "path", f.fullPath, "err", err)
			continue
		}

		if err := repo.Upsert(f.hash, f.fullPath, f.isVideo); err != nil {
			logger.L.Warn("thumbnail upsert error after generation", "path", f.fullPath, "err", err)
			continue
		}
		generated++
	}
	return generated
}

func generateThumbnail(f fileNeedingThumbnail) error {
	sem := service.GetThumbnailSemaphore()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := sem.Acquire(ctx); err != nil {
		return fmt.Errorf("acquire semaphore: %w", err)
	}
	defer sem.Release()

	if f.isVideo {
		return service.GenerateVideoThumbnail(ctx, f.fullPath, f.thumbPath)
	}
	return service.GeneratePhotoThumbnail(ctx, f.fullPath, f.thumbPath)
}

func isVideoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".mp4", ".mkv", ".mov", ".avi", ".webm":
		return true
	}
	return false
}

func isPhotoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".heic":
		return true
	}
	return false
}
