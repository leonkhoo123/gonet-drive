package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"go-file-server/internal/logger"
	"go-file-server/internal/repository"
	"go-file-server/internal/util/imageutil"

	"github.com/chai2010/webp"
)

var thumbRepo repository.ThumbnailRepository

func SetThumbnailRepo(repo repository.ThumbnailRepository) {
	thumbRepo = repo
}

func GenerateVideoThumbnail(ctx context.Context, fullPath, thumbPath string) error {
	logger.L.Debug("generating video thumbnail", "input", fullPath, "output", thumbPath)
	cmd := exec.CommandContext(ctx,
		"prlimit",
		"--as=524288000",
		"ffmpeg",
		"-loglevel", "error",
		"-threads", "1",
		"-ss", "00:00:00.000",
		"-i", fullPath,
		"-an",
		"-vframes", "1",
		"-vf", "scale='min(300,iw)':'min(300,ih)':force_original_aspect_ratio=decrease",
		"-c:v", "libwebp",
		"-y",
		thumbPath)

	if err := cmd.Run(); err != nil {
		os.Remove(thumbPath)
		return fmt.Errorf("ffmpeg: %w", err)
	}
	return nil
}

func GeneratePhotoThumbnail(fullPath, thumbPath string) error {
	logger.L.Debug("generating photo thumbnail", "input", fullPath, "output", thumbPath)
	src, err := imageutil.DecodeImage(fullPath)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	dst := imageutil.ResizeImage(src, 300)

	out, err := os.Create(thumbPath)
	if err != nil {
		return fmt.Errorf("create thumbnail file: %w", err)
	}
	defer out.Close()

	if err := webp.Encode(out, dst, &webp.Options{Lossless: false, Quality: 85}); err != nil {
		return fmt.Errorf("encode webp: %w", err)
	}
	return nil
}

func UpsertThumbnailRecord(hash, filePath string, isVideo bool) {
	if thumbRepo != nil {
		if err := thumbRepo.Upsert(hash, filePath, isVideo); err != nil {
			logger.L.Warn("thumbnail upsert failed", "file", filePath, "err", err)
		}
	}
}
