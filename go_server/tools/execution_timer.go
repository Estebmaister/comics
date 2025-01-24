package jobs

import (
	"fmt"
	"time"
)

func ExecutionTimer() func() {
	start := time.Now()
	return func() {
		fmt.Printf("---- job execution took %v\n", time.Since(start))
	}
}
