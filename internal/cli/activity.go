package cli

import (
	"fmt"
	"io"
	"time"
)

// generationActivity owns one transient terminal line. stop joins the worker
// before the reporter writes, so progress frames cannot interleave with logs.
type generationActivity struct {
	out     io.Writer
	enabled bool
	cancel  chan struct{}
	done    chan struct{}
}

func (a *generationActivity) start() {
	if !a.enabled {
		return
	}
	a.stop()
	a.cancel, a.done = make(chan struct{}), make(chan struct{})
	go func() {
		defer close(a.done)
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		start := time.Now()
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		for i := 0; ; i++ {
			_, _ = fmt.Fprintf(a.out, "\r\x1b[2K  %s Working · %s", frames[i%len(frames)], time.Since(start).Truncate(time.Second))
			select {
			case <-a.cancel:
				_, _ = fmt.Fprint(a.out, "\r\x1b[2K")
				return
			case <-ticker.C:
			}
		}
	}()
}

func (a *generationActivity) stop() {
	if a.cancel == nil {
		return
	}
	close(a.cancel)
	<-a.done
	a.cancel, a.done = nil, nil
}
