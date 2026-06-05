package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/h2non/bimg"

	"go-file-server/internal/logger"
	"go-file-server/internal/repository"
)

const ffmpegMemoryLimit = "524288000"

var thumbRepo repository.ThumbnailRepository

func SetThumbnailRepo(repo repository.ThumbnailRepository) {
	thumbRepo = repo
}

func GenerateVideoThumbnail(ctx context.Context, fullPath, thumbPath string) error {
	logger.L.Debug("generating video thumbnail", "input", fullPath, "output", thumbPath)
	cmd := exec.CommandContext(ctx,
		"prlimit",
		"--as="+ffmpegMemoryLimit,
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

func GeneratePhotoThumbnail(ctx context.Context, fullPath, thumbPath string) error {
	logger.L.Debug("generating photo thumbnail", "input", fullPath, "output", thumbPath)

	buffer, err := bimg.Read(fullPath)
	if err != nil {
		return fmt.Errorf("bimg read: %w", err)
	}

	options := bimg.Options{
		Width:         300,
		Height:        0,
		Quality:       85,
		StripMetadata: true,
		Type:          bimg.WEBP,
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	newImage, err := bimg.NewImage(buffer).Process(options)
	if err != nil {
		return fmt.Errorf("bimg process: %w", err)
	}

	if err := bimg.Write(thumbPath, newImage); err != nil {
		return fmt.Errorf("bimg write: %w", err)
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
