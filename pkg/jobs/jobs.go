package jobs

import (
	"context"
	"strings"
	"sync"
)

// Job is a job that can be registered with the scheduler.
type Job struct {
	ID     int
	Runner Runner
}

// Runner is a job runner.
type Runner interface {
	Spec(context.Context) string
	Func(context.Context) func()
}

var (
	mtx  sync.Mutex
	jobs = make(map[string]*Job, 0)
)

// Register registers a job.
func Register(name string, runner Runner) {
	mtx.Lock()
	defer mtx.Unlock()
	jobs[name] = &Job{Runner: runner}
}

// List returns a map of registered jobs.
func List() map[string]*Job {
	mtx.Lock()
	defer mtx.Unlock()
	return jobs
}

// Disabled reports whether a job spec turns the job off. The spec is matched
// case-insensitively against "off", "disabled" and "-". An empty spec is not
// a disable value; jobs fall back to their default schedule for it.
func Disabled(spec string) bool {
	switch strings.ToLower(strings.TrimSpace(spec)) {
	case "off", "disabled", "-":
		return true
	}
	return false
}
