package service

import "log"

// Job represents a single file operation task
type Job func()

// JobQueue is a buffered channel that holds pending tasks.
// The buffer size limits how many operations can be queued before the HTTP API blocks.
// We set it to 100 which is more than enough for regular NAS operations.
var JobQueue = make(chan Job, 100)

// StartFileOperationWorker starts a background worker that processes file operations sequentially.
func StartFileOperationWorker() {
	go func() {
		log.Println("Starting sequential file operation worker...")
		for job := range JobQueue {
			// Process one job at a time to ensure sequential execution
			job()
		}
	}()
}
