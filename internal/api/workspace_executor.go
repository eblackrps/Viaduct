package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const defaultWorkspaceJobConcurrency = 4
const defaultWorkspaceEnqueueTimeout = 30 * time.Second

var (
	// ErrExecutorShuttingDown reports that the workspace executor stopped accepting work because the server is shutting down.
	ErrExecutorShuttingDown = errors.New("workspace job executor is shutting down")
	// ErrTenantQueueFull reports that a single tenant exhausted its fair-share of the queued workspace capacity.
	ErrTenantQueueFull = errors.New("workspace job executor tenant queue share is full")
	// ErrEnqueueTimeout reports that a caller waited too long for the executor to acknowledge queue admission.
	ErrEnqueueTimeout = errors.New("workspace job executor enqueue timed out")
	// ErrResultChannelFull reports that executor bookkeeping could not deliver an enqueue result to the waiting caller.
	ErrResultChannelFull = errors.New("workspace job executor result channel is not accepting values")
)

type workspaceJobTask struct {
	tenantID    string
	workspaceID string
	jobID       string
	ctx         context.Context
	release     context.CancelFunc
}

type workspaceJobEnqueuer interface {
	Enqueue(ctx context.Context, task workspaceJobTask) error
}

type workspaceQueueDepthReporter interface {
	QueueDepthByTenant() map[string]int
}

type workspaceJobEnqueueRequest struct {
	task       workspaceJobTask
	waitCtx    context.Context
	result     chan error
	enqueuedAt int64
}

type workspaceWorkerHandle struct {
	tasks chan workspaceJobTask
	exit  <-chan struct{}
}

type workspaceJobExecutor struct {
	ctx            context.Context
	done           chan struct{}
	enqueue        chan workspaceJobEnqueueRequest
	workerReady    chan workspaceWorkerHandle
	queueCapacity  int
	enqueueTimeout time.Duration
	run            func(context.Context, workspaceJobTask)

	queueMu        sync.RWMutex
	queuedByTenant map[string]int
}

func newWorkspaceJobExecutor(ctx context.Context, concurrency int, run func(context.Context, workspaceJobTask)) *workspaceJobExecutor {
	return newWorkspaceJobExecutorWithTimeout(ctx, concurrency, defaultWorkspaceEnqueueTimeout, run)
}

func newWorkspaceJobExecutorWithTimeout(ctx context.Context, concurrency int, enqueueTimeout time.Duration, run func(context.Context, workspaceJobTask)) *workspaceJobExecutor {
	if concurrency <= 0 {
		concurrency = defaultWorkspaceJobConcurrency
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if enqueueTimeout <= 0 {
		enqueueTimeout = defaultWorkspaceEnqueueTimeout
	}

	executor := &workspaceJobExecutor{
		ctx:            ctx,
		done:           make(chan struct{}),
		enqueue:        make(chan workspaceJobEnqueueRequest),
		workerReady:    make(chan workspaceWorkerHandle),
		queueCapacity:  concurrency * 2,
		enqueueTimeout: enqueueTimeout,
		run:            run,
		queuedByTenant: make(map[string]int),
	}
	if executor.queueCapacity <= 0 {
		executor.queueCapacity = concurrency
	}
	go executor.dispatch()
	for worker := 0; worker < concurrency; worker++ {
		go executor.worker()
	}
	return executor
}

func (e *workspaceJobExecutor) Enqueue(ctx context.Context, task workspaceJobTask) error {
	if e == nil {
		return fmt.Errorf("workspace job executor is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	waitCtx, cancel := context.WithTimeout(ctx, e.enqueueTimeout)
	defer cancel()
	request := workspaceJobEnqueueRequest{
		task:    task,
		waitCtx: waitCtx,
		result:  make(chan error),
	}

	select {
	case <-e.done:
		return e.executorErr()
	case <-ctx.Done():
		return ctx.Err()
	case <-waitCtx.Done():
		if ctx.Err() == nil && errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return ErrEnqueueTimeout
		}
		return waitCtx.Err()
	case e.enqueue <- request:
	}

	select {
	case err := <-request.result:
		return err
	case <-e.done:
		return e.executorErr()
	case <-ctx.Done():
		return ctx.Err()
	case <-waitCtx.Done():
		if ctx.Err() == nil && errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return ErrEnqueueTimeout
		}
		return waitCtx.Err()
	}
}

func (e *workspaceJobExecutor) QueueDepthByTenant() map[string]int {
	if e == nil {
		return nil
	}

	e.queueMu.RLock()
	defer e.queueMu.RUnlock()

	if len(e.queuedByTenant) == 0 {
		return nil
	}

	items := make(map[string]int, len(e.queuedByTenant))
	for tenantID, depth := range e.queuedByTenant {
		items[tenantID] = depth
	}
	return items
}

func (e *workspaceJobExecutor) dispatch() {
	if e == nil {
		return
	}
	defer close(e.done)
	defer e.clearQueueDepths()

	queue := make([]workspaceJobEnqueueRequest, 0, e.queueCapacity)
	idleWorkers := make([]workspaceWorkerHandle, 0)

	for {
		if len(idleWorkers) > 0 && len(queue) > 0 {
			worker := idleWorkers[0]
			idleWorkers = idleWorkers[1:]
			request := queue[0]
			queue = queue[1:]
			e.removeQueuedRequest(request)

			if !e.assignTask(worker, request) {
				e.rejectPending(queue)
				e.drainEnqueueRequests()
				return
			}
			continue
		}

		select {
		case <-e.ctx.Done():
			e.rejectPending(queue)
			e.drainEnqueueRequests()
			return
		case worker := <-e.workerReady:
			idleWorkers = append(idleWorkers, worker)
		case request := <-e.enqueue:
			if e.ctx.Err() != nil {
				_ = request.respond(ErrExecutorShuttingDown) // Best effort: the caller may already have timed out or closed its receive path.
				e.rejectPending(queue)
				e.drainEnqueueRequests()
				return
			}

			if e.tenantQueueLimitExceeded(request.task.tenantID) {
				request.task.cleanup()
				_ = request.respond(ErrTenantQueueFull) // Best effort: the caller may already have timed out or closed its receive path.
				continue
			}

			for len(queue) >= e.queueCapacity {
				select {
				case <-e.ctx.Done():
					request.task.cleanup()
					_ = request.respond(ErrExecutorShuttingDown) // Best effort: the caller may already have timed out or closed its receive path.
					e.rejectPending(queue)
					e.drainEnqueueRequests()
					return
				case <-request.waitCtx.Done():
					request.task.cleanup()
					continue
				case worker := <-e.workerReady:
					oldest := queue[0]
					queue = queue[1:]
					e.removeQueuedRequest(oldest)
					if !e.assignTask(worker, oldest) {
						request.task.cleanup()
						_ = request.respond(ErrExecutorShuttingDown) // Best effort: the caller may already have timed out or closed its receive path.
						e.rejectPending(queue)
						e.drainEnqueueRequests()
						return
					}
				}
			}

			if e.ctx.Err() != nil {
				request.task.cleanup()
				_ = request.respond(ErrExecutorShuttingDown) // Best effort: the caller may already have timed out or closed its receive path.
				e.rejectPending(queue)
				e.drainEnqueueRequests()
				return
			}

			request.enqueuedAt = time.Now().UTC().UnixNano()
			queue = append(queue, request)
			e.adjustTenantQueueDepth(request.task.tenantID, 1)
			if err := request.respond(nil); err != nil {
				removed := queue[len(queue)-1]
				queue = queue[:len(queue)-1]
				e.removeQueuedRequest(removed)
				request.task.cleanup()
				continue
			}
			// Queue admission is the only result delivered back to the caller. Clear the
			// queued copy's result channel so later shutdown cleanup cannot block trying
			// to send a second terminal status to a caller that already returned.
			queue[len(queue)-1].result = nil
		}
	}
}

func (e *workspaceJobExecutor) assignTask(worker workspaceWorkerHandle, request workspaceJobEnqueueRequest) bool {
	if e == nil {
		request.task.cleanup()
		_ = request.respond(fmt.Errorf("workspace job executor is not configured")) // Best effort: the caller may already have timed out or closed its receive path.
		return false
	}

	select {
	case <-e.ctx.Done():
		request.task.cleanup()
		_ = request.respond(ErrExecutorShuttingDown) // Best effort: the caller may already have timed out or closed its receive path.
		return false
	case <-worker.exit:
		request.task.cleanup()
		_ = request.respond(ErrExecutorShuttingDown) // Best effort: the caller may already have timed out or closed its receive path.
		return false
	case worker.tasks <- request.task:
		return true
	}
}

func (e *workspaceJobExecutor) rejectPending(queue []workspaceJobEnqueueRequest) {
	if e == nil {
		return
	}
	for _, request := range queue {
		e.removeQueuedRequest(request)
		request.task.cleanup()
		_ = request.respond(ErrExecutorShuttingDown) // Best effort: the caller may already have timed out or closed its receive path.
	}
	e.clearQueueDepths()
}

func (e *workspaceJobExecutor) drainEnqueueRequests() {
	if e == nil {
		return
	}
	for {
		select {
		case request := <-e.enqueue:
			request.task.cleanup()
			_ = request.respond(ErrExecutorShuttingDown) // Best effort: the caller may already have timed out or closed its receive path.
		default:
			return
		}
	}
}

func (e *workspaceJobExecutor) worker() {
	if e == nil {
		return
	}

	tasks := make(chan workspaceJobTask)
	workerExit := make(chan struct{})
	defer close(workerExit)
	handle := workspaceWorkerHandle{tasks: tasks, exit: workerExit}
	for {
		select {
		case <-e.ctx.Done():
			return
		case e.workerReady <- handle:
		}

		select {
		case <-e.ctx.Done():
			return
		case task := <-tasks:
			if e.ctx.Err() != nil {
				task.cleanup()
				return
			}
			if e.run != nil {
				runCtx, cancel := e.taskContext(task)
				e.run(runCtx, task)
				cancel()
			}
			task.cleanup()
			if e.ctx.Err() != nil {
				return
			}
		}
	}
}

func (e *workspaceJobExecutor) taskContext(task workspaceJobTask) (context.Context, context.CancelFunc) {
	baseCtx := task.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	runCtx, cancel := context.WithCancel(baseCtx)
	go func() {
		select {
		case <-e.ctx.Done():
			cancel()
		case <-runCtx.Done():
		}
	}()
	return runCtx, cancel
}

func (e *workspaceJobExecutor) tenantQueueLimitExceeded(tenantID string) bool {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" || e.queueCapacity <= 0 {
		return false
	}

	maxTenantDepth := e.queueCapacity / 2
	if maxTenantDepth <= 0 {
		maxTenantDepth = 1
	}

	e.queueMu.RLock()
	defer e.queueMu.RUnlock()
	return e.queuedByTenant[tenantID]+1 > maxTenantDepth
}

func (e *workspaceJobExecutor) adjustTenantQueueDepth(tenantID string, delta int) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" || delta == 0 {
		return
	}

	e.queueMu.Lock()
	defer e.queueMu.Unlock()

	next := e.queuedByTenant[tenantID] + delta
	if next <= 0 {
		delete(e.queuedByTenant, tenantID)
		return
	}
	e.queuedByTenant[tenantID] = next
}

func (e *workspaceJobExecutor) clearQueueDepths() {
	e.queueMu.Lock()
	defer e.queueMu.Unlock()
	clear(e.queuedByTenant)
}

func (e *workspaceJobExecutor) removeQueuedRequest(request workspaceJobEnqueueRequest) {
	if e == nil || request.enqueuedAt == 0 {
		return
	}
	e.adjustTenantQueueDepth(request.task.tenantID, -1)
}

func (e *workspaceJobExecutor) executorErr() error {
	if e == nil {
		return fmt.Errorf("workspace job executor is not configured")
	}
	if e.ctx.Err() != nil {
		return ErrExecutorShuttingDown
	}
	return fmt.Errorf("workspace job executor is not accepting new work")
}

func (t workspaceJobTask) cleanup() {
	if t.release != nil {
		t.release()
	}
}

func (r workspaceJobEnqueueRequest) respond(err error) (sendErr error) {
	if r.result == nil {
		return ErrResultChannelFull
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			sendErr = ErrResultChannelFull
		}
	}()
	select {
	case <-r.waitCtx.Done():
		return r.waitCtx.Err()
	case r.result <- err:
		return nil
	}
}
