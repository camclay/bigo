package workers

import (
	"context"
	"sync"

	"github.com/cammy/bigo/pkg/types"
)

// Worker interface for execution backends
type Worker interface {
	Execute(ctx context.Context, task *types.Task) (*types.ExecutionResult, error)
	Available() bool
	Backend() types.Backend
	CheckQuota(ctx context.Context) error
}

// Pool manages a collection of workers for a specific backend type
type Pool struct {
	backend     types.Backend
	workers     []Worker
	maxWorkers  int
	activeCount int
	mu          sync.Mutex
}

// NewPool creates a new worker pool
func NewPool(backend types.Backend, maxWorkers int) *Pool {
	return &Pool{
		backend:    backend,
		workers:    make([]Worker, 0, maxWorkers),
		maxWorkers: maxWorkers,
	}
}

// Add adds a worker to the pool
func (p *Pool) Add(w Worker) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.workers) < p.maxWorkers {
		p.workers = append(p.workers, w)
	}
}

// Acquire gets an available worker from the pool
func (p *Pool) Acquire() Worker {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, w := range p.workers {
		if w.Available() {
			p.activeCount++
			return w
		}
	}
	return nil
}

// Release returns a worker to the pool
func (p *Pool) Release(w Worker) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.activeCount > 0 {
		p.activeCount--
	}
}

// Available returns true if any worker is available
func (p *Pool) Available() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, w := range p.workers {
		if w.Available() {
			return true
		}
	}
	return false
}

// Size returns the number of workers in the pool
func (p *Pool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.workers)
}

// ActiveCount returns the number of active workers
func (p *Pool) ActiveCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeCount
}

// Backend returns the pool's backend type
func (p *Pool) Backend() types.Backend {
	return p.backend
}
