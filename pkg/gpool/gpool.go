package gpool

import "sync"

// Pool represents a goroutine pool
type Pool struct {
    work chan func()
    sem  chan struct{}
    wg   sync.WaitGroup
}

// New creates a new goroutine pool
func New(size int) *Pool {
    return &Pool{
        work: make(chan func()),
        sem:  make(chan struct{}, size),
    }
}

// Schedule schedules work to the pool
func (p *Pool) Schedule(task func()) {
    p.Add(1)
    wrappedTask := func() {
        defer p.Done()
        task()
    }
    
    select {
    case p.work <- wrappedTask:
    case p.sem <- struct{}{}:
        go p.worker(wrappedTask)
    }
}

// Add increments the WaitGroup counter
func (p *Pool) Add(delta int) {
    p.wg.Add(delta)
}

// Done decrements the WaitGroup counter
func (p *Pool) Done() {
    p.wg.Done()
}

// Wait blocks until the WaitGroup counter is zero
func (p *Pool) Wait() {
    p.wg.Wait()
}

func (p *Pool) worker(task func()) {
    defer func() { <-p.sem }()
    for {
        task()
        task = <-p.work
    }
}
