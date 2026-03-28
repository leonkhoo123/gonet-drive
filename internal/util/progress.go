package util

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ProgressTracker tracks the progress of file operations (copy, move, delete)
type ProgressTracker struct {
	TotalBytes   int64
	CopiedBytes  int64 // Used for copy
	MovedFiles   int   // Used for move
	DeletedFiles int   // Used for delete
	StartTime    time.Time
	LastLogTime  time.Time
	mu           sync.Mutex
	TotalFiles   int
	CopiedFiles  int
	OnProgress   func(*ProgressTracker) // Callback for progress updates
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker() *ProgressTracker {
	now := time.Now()
	return &ProgressTracker{
		StartTime:   now,
		LastLogTime: now,
	}
}

// Update updates the progress and logs if needed (for bytes - mainly copy)
func (pt *ProgressTracker) Update(bytesWritten int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.CopiedBytes += bytesWritten

	// Log progress every 500ms to avoid spamming
	if time.Since(pt.LastLogTime) >= 500*time.Millisecond {
		pt.logProgress()
		if pt.OnProgress != nil {
			pt.OnProgress(pt)
		}
		pt.LastLogTime = time.Now()
	}
}

// FileCompleted marks a file as completed and logs progress
// Used for all operations to increment counts
func (pt *ProgressTracker) FileCompleted() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.CopiedFiles++
	pt.MovedFiles++
	pt.DeletedFiles++

	// Log progress every 100ms to avoid spamming
	if time.Since(pt.LastLogTime) >= 100*time.Millisecond {
		pt.logProgress()
		if pt.OnProgress != nil {
			pt.OnProgress(pt)
		}
		pt.LastLogTime = time.Now()
	}
}

// logProgress logs the current progress based on the context
func (pt *ProgressTracker) logProgress() {
	elapsed := time.Since(pt.StartTime).Seconds()

	// Determine operation type based on non-zero fields or context
	// For generic logging, we try to show what's available

	// Default to generic file count
	percentage := 0.0
	if pt.TotalFiles > 0 {
		// Use CopiedFiles as the generic "Done" counter if mostly doing file operations
		// Note: Since FileCompleted increments all counters, checking TotalFiles is safe
		// However, for bytes-based progress (Copy), we prefer byte calculation
		if pt.TotalBytes > 0 {
			percentage = float64(pt.CopiedBytes) / float64(pt.TotalBytes) * 100
		} else {
			// Fallback to file count (Move/Delete)
			// We can use CopiedFiles since FileCompleted increments it too,
			// or we can just use the Max of the counters.
			// Since we act generically, let's use CopiedFiles which is also incremented in FileCompleted
			percentage = float64(pt.CopiedFiles) / float64(pt.TotalFiles) * 100
		}
	}

	// Calculate speed (bytes per second) - mainly for copy
	var speed float64
	if pt.TotalBytes > 0 && elapsed > 0 {
		speed = float64(pt.CopiedBytes) / elapsed
	}

	// Calculate ETA
	var eta string
	if pt.TotalBytes > 0 && speed > 0 && pt.CopiedBytes < pt.TotalBytes {
		remainingBytes := pt.TotalBytes - pt.CopiedBytes
		remainingSeconds := float64(remainingBytes) / speed
		eta = formatDuration(time.Duration(remainingSeconds) * time.Second)
	} else if pt.TotalFiles > 0 && pt.CopiedFiles > 0 && pt.CopiedFiles < pt.TotalFiles {
		// Estimate based on files
		filesPerSec := float64(pt.CopiedFiles) / elapsed
		remainingFiles := pt.TotalFiles - pt.CopiedFiles
		remainingSeconds := float64(remainingFiles) / filesPerSec
		eta = formatDuration(time.Duration(remainingSeconds) * time.Second)
	} else {
		eta = "calculating..."
	}

	if pt.TotalBytes > 0 {
		// Copy mode log
		log.Printf("Progress: %.2f%% | %s / %s | Speed: %s/s | Files: %d/%d | ETA: %s",
			percentage,
			FormatBytes(pt.CopiedBytes),
			FormatBytes(pt.TotalBytes),
			FormatBytes(int64(speed)),
			pt.CopiedFiles,
			pt.TotalFiles,
			eta,
		)
	} else {
		// Move/Delete mode log
		log.Printf("Progress: %.2f%% | Files: %d/%d | ETA: %s",
			percentage,
			pt.CopiedFiles, // Works as generic "Done" count
			pt.TotalFiles,
			eta,
		)
	}
}

// FinalLog logs the final completion message
func (pt *ProgressTracker) FinalLog() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	elapsed := time.Since(pt.StartTime)

	if pt.TotalBytes > 0 {
		avgSpeed := float64(pt.TotalBytes) / elapsed.Seconds()
		log.Printf("✓ Operation completed! Total: %s | Files: %d | Time: %s | Avg Speed: %s/s",
			FormatBytes(pt.TotalBytes),
			pt.TotalFiles,
			formatDuration(elapsed),
			FormatBytes(int64(avgSpeed)),
		)
	} else {
		log.Printf("✓ Operation completed! Total files: %d | Time: %s",
			pt.TotalFiles,
			formatDuration(elapsed),
		)
	}
}

// FormatBytes formats bytes into human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration into human-readable format
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
