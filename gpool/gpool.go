package gpool

// Pool represents a goroutine pool
type Pool struct {
    work chan func()
    sem  chan struct{}
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
    select {
    case p.work <- task:
    case p.sem <- struct{}{}:
        go p.worker(task)
    }
}

func (p *Pool) worker(task func()) {
    defer func() { <-p.sem }()
    for {
        task()
        task = <-p.work
    }
}
