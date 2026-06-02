package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"go-file-server/internal/logger"
	"go-file-server/internal/repository"
	"go-file-server/internal/ws"
)

var videoIntegrityRepo repository.VideoIntegrityRepository

func SetVideoIntegrityRepo(repo repository.VideoIntegrityRepository) {
	videoIntegrityRepo = repo
}

func GetVideoIntegrityRepo() repository.VideoIntegrityRepository {
	return videoIntegrityRepo
}

var scanRunning atomic.Bool
var scanStop atomic.Bool

// ResetIntegrityScanForTest resets the scan gate for test isolation.
// Not intended for production use.
func ResetIntegrityScanForTest() {
	scanRunning.Store(false)
	scanStop.Store(false)
}

// SetScanRunningForTest sets the scan gate for test purposes.
// Not intended for production use.
func SetScanRunningForTest(running bool) {
	scanRunning.Store(running)
}

// SetScanStopForTest sets the stop flag for test purposes.
// Not intended for production use.
func SetScanStopForTest(stop bool) {
	scanStop.Store(stop)
}

// RequestScanStop signals a running scan to stop after the current file completes.
// Returns false if no scan is running.
func RequestScanStop() bool {
	if !scanRunning.Load() {
		return false
	}
	scanStop.Store(true)
	return true
}

type ScanResult struct {
	TotalScanned int   `json:"total_scanned"`
	CorruptCount int   `json:"corrupt_count"`
	StartTime    int64 `json:"start_time"`
	EndTime      int64 `json:"end_time"`
}

func IsVideoIntegrityScanRunning() bool {
	return scanRunning.Load()
}

func GetScanStatus() (corruptCount int, lastScan *time.Time, running bool, err error) {
	if videoIntegrityRepo == nil {
		return 0, nil, false, fmt.Errorf("video integrity repository not initialized")
	}
	count, err := videoIntegrityRepo.Count()
	if err != nil {
		return 0, nil, false, err
	}
	lastScan, err = videoIntegrityRepo.LastScanTime()
	if err != nil {
		return count, nil, false, err
	}
	return count, lastScan, scanRunning.Load(), nil
}

func ScanVideoIntegrity(rootPath string) (*ScanResult, error) {
	if !scanRunning.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("scan already running")
	}
	defer scanRunning.Store(false)
	scanStop.Store(false)

	logger.L.Info("video integrity scan started", "root", rootPath)
	// Log ffprobe version for debugging version-specific behavior
	logFFprobeVersion()

	startTime := time.Now()
	result := &ScanResult{StartTime: startTime.Unix()}

	if videoIntegrityRepo == nil {
		return nil, fmt.Errorf("video integrity repository not initialized")
	}

	if err := videoIntegrityRepo.DeleteAll(); err != nil {
		logger.L.Error("video integrity scan failed to clear previous results", "err", err)
		return nil, fmt.Errorf("failed to clear previous results: %w", err)
	}
	logger.L.Debug("video integrity scan cleared previous results")

	var videoFiles []string
	walkErr := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.L.Warn("video integrity walk error", "path", path, "err", err)
			return nil
		}
		if info.IsDir() {
			if path != filepath.Join(rootPath, ".cloud_reserve") &&
				strings.HasPrefix(path, filepath.Join(rootPath, ".cloud_reserve")+string(filepath.Separator)) {
				return filepath.SkipDir
			}
			return nil
		}
		name := info.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".mp4" || ext == ".mov" {
			videoFiles = append(videoFiles, path)
		}
		return nil
	})
	if walkErr != nil {
		logger.L.Error("video integrity walk failed", "err", walkErr)
		return nil, fmt.Errorf("file walk error: %w", walkErr)
	}

	totalFiles := len(videoFiles)
	logger.L.Info("video integrity scan found video files", "count", totalFiles)

	if totalFiles == 0 {
		result.EndTime = time.Now().Unix()
		ws.Broadcast(ws.OperationMessage{
			OpId:     "integrity-scan",
			OpType:   "integrity-scan",
			OpStatus: "completed",
		})
		logger.L.Info("video integrity scan completed", "total", 0, "corrupt", 0)
		return result, nil
	}

	for i, path := range videoFiles {
		if scanStop.Load() {
			result.EndTime = time.Now().Unix()
			completed := float64(i) / float64(totalFiles) * 100
			fileCountStr := fmt.Sprintf("%d/%d", i, totalFiles)
			ws.Broadcast(ws.OperationMessage{
				OpId:         "integrity-scan",
				OpType:       "integrity-scan",
				OpStatus:     "aborted",
				OpPercentage: &completed,
				OpFileCount:  &fileCountStr,
			})
			logger.L.Info("video integrity scan stopped by user",
				"scanned", i, "total", totalFiles, "corrupt", result.CorruptCount,
				"duration", time.Since(startTime).Round(time.Second))
			return result, nil
		}

		hash := md5.Sum([]byte(path))
		hashStr := hex.EncodeToString(hash[:])

		mimeCodec, codecName, err := probeVideoStream(path)
		if err != nil {
			logger.L.Warn("video integrity ffprobe failed, skipping", "path", path, "err", err)
			result.TotalScanned++
			continue
		}

		logger.L.Debug("video integrity probe", "file", filepath.Base(path), "codec", codecName, "mime", mimeCodec)

		if codecName == "h264" && strings.HasPrefix(mimeCodec, "avc1.00") {
			if err := videoIntegrityRepo.Upsert(hashStr, path, "corrupt_avcC", mimeCodec); err != nil {
				logger.L.Error("video integrity upsert failed", "path", path, "err", err)
			} else {
				result.CorruptCount++
				logger.L.Warn("video integrity detected corrupt avcC", "file", filepath.Base(path), "mime", mimeCodec)
			}
		}

		result.TotalScanned++
		percentage := float64(i+1) / float64(totalFiles) * 100
		fileCountStr := fmt.Sprintf("%d/%d", i+1, totalFiles)

		ws.Broadcast(ws.OperationMessage{
			OpId:         "integrity-scan",
			OpType:       "integrity-scan",
			OpStatus:     "in-progress",
			OpPercentage: &percentage,
			OpFileCount:  &fileCountStr,
		})
	}

	result.EndTime = time.Now().Unix()

	ws.Broadcast(ws.OperationMessage{
		OpId:     "integrity-scan",
		OpType:   "integrity-scan",
		OpStatus: "completed",
	})

	logger.L.Info("video integrity scan completed",
		"total", result.TotalScanned,
		"corrupt", result.CorruptCount,
		"duration", time.Since(startTime).Round(time.Second))

	return result, nil
}

type ffprobeStream struct {
	CodecName        string `json:"codec_name"`
	MimeCodecString  string `json:"mime_codec_string"`
	CodecType        string `json:"codec_type"`
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
}

func probeVideoStream(path string) (mimeCodec, codecName string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		"ffprobe", "-v", "error",
		"-show_streams", "-select_streams", "v:0",
		"-of", "json",
		path,
	)

	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			logger.L.Debug("ffprobe stderr", "file", filepath.Base(path), "stderr", strings.TrimSpace(stderr.String()))
		}
		return "", "", fmt.Errorf("ffprobe: %w", err)
	}

	rawJSON := stdout.String()
	logger.L.Debug("ffprobe raw output", "file", filepath.Base(path), "json", rawJSON)

	var output ffprobeOutput
	if err := json.Unmarshal([]byte(rawJSON), &output); err != nil {
		return "", "", fmt.Errorf("ffprobe json parse: %w", err)
	}

	if len(output.Streams) == 0 {
		return "", "", fmt.Errorf("no video stream found")
	}

	return output.Streams[0].MimeCodecString, output.Streams[0].CodecName, nil
}

func logFFprobeVersion() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffprobe", "-version")
	out, err := cmd.Output()
	if err != nil {
		logger.L.Warn("ffprobe version check failed", "err", err)
		return
	}
	// First line contains version
	firstLine := strings.SplitN(string(out), "\n", 2)[0]
	logger.L.Info("ffprobe version", "version", firstLine)
}
