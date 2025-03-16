package jobs

import (
	"fmt"
	"time"
)

// ExecutionTimer returns a function that measures the execution time of a job
// Use it like this:
// timer := jobs.ExecutionTimer()
// defer timer() // call when job is done
func ExecutionTimer() func() {
	start := time.Now()
	return func() {
		fmt.Printf("---- job execution took %v\n", time.Since(start))
	}
}
