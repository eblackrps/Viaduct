package api

import (
	"context"
	"fmt"
)

const defaultWorkspaceJobConcurrency = 4

type workspaceJobTask struct {
	tenantID    string
	workspaceID string
	jobID       string
}

type workspaceJobEnqueuer interface {
	Enqueue(ctx context.Context, task workspaceJobTask) error
}

type workspaceJobEnqueueRequest struct {
	task   workspaceJobTask
	result chan error
}

type workspaceJobExecutor struct {
	ctx           context.Context
	done          chan struct{}
	enqueue       chan workspaceJobEnqueueRequest
	workerReady   chan chan workspaceJobTask
	queueCapacity int
	run           func(context.Context, workspaceJobTask)
}

func newWorkspaceJobExecutor(ctx context.Context, concurrency int, run func(context.Context, workspaceJobTask)) *workspaceJobExecutor {
	if concurrency <= 0 {
		concurrency = defaultWorkspaceJobConcurrency
	}
	if ctx == nil {
		ctx = context.Background()
	}

	executor := &workspaceJobExecutor{
		ctx:           ctx,
		done:          make(chan struct{}),
		enqueue:       make(chan workspaceJobEnqueueRequest),
		workerReady:   make(chan chan workspaceJobTask),
		queueCapacity: concurrency * 2,
		run:           run,
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

	request := workspaceJobEnqueueRequest{
		task:   task,
		result: make(chan error, 1),
	}
	select {
	case <-e.done:
		return e.executorErr()
	case <-ctx.Done():
		return ctx.Err()
	case e.enqueue <- request:
	}

	select {
	case err := <-request.result:
		return err
	case <-e.done:
		return e.executorErr()
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *workspaceJobExecutor) dispatch() {
	if e == nil {
		return
	}
	defer close(e.done)

	queue := make([]workspaceJobEnqueueRequest, 0, e.queueCapacity)
	idleWorkers := make([]chan workspaceJobTask, 0)

	for {
		if len(idleWorkers) > 0 && len(queue) > 0 {
			worker := idleWorkers[0]
			idleWorkers = idleWorkers[1:]
			request := queue[0]
			queue = queue[1:]

			if !e.assignTask(worker, request) {
				e.rejectPending(queue)
				return
			}
			continue
		}

		select {
		case <-e.ctx.Done():
			e.rejectPending(queue)
			return
		case worker := <-e.workerReady:
			if worker == nil {
				continue
			}
			idleWorkers = append(idleWorkers, worker)
		case request := <-e.enqueue:
			if err := e.ctx.Err(); err != nil {
				request.respond(err)
				e.rejectPending(queue)
				return
			}

			if len(idleWorkers) > 0 {
				worker := idleWorkers[0]
				idleWorkers = idleWorkers[1:]
				if !e.assignTask(worker, request) {
					e.rejectPending(queue)
					return
				}
				continue
			}

			for len(queue) >= e.queueCapacity {
				select {
				case <-e.ctx.Done():
					request.respond(e.executorErr())
					e.rejectPending(queue)
					return
				case worker := <-e.workerReady:
					if worker == nil {
						continue
					}
					oldest := queue[0]
					queue = queue[1:]
					if !e.assignTask(worker, oldest) {
						request.respond(e.executorErr())
						e.rejectPending(queue)
						return
					}
				}
			}

			select {
			case <-e.ctx.Done():
				request.respond(e.executorErr())
				e.rejectPending(queue)
				return
			default:
			}

			queue = append(queue, request)
			request.respond(nil)
		}
	}
}

func (e *workspaceJobExecutor) assignTask(worker chan workspaceJobTask, request workspaceJobEnqueueRequest) bool {
	if e == nil {
		request.respond(fmt.Errorf("workspace job executor is not configured"))
		return false
	}

	select {
	case <-e.ctx.Done():
		request.respond(e.executorErr())
		return false
	case worker <- request.task:
		request.respond(nil)
		return true
	}
}

func (e *workspaceJobExecutor) rejectPending(queue []workspaceJobEnqueueRequest) {
	if e == nil {
		return
	}
	err := e.executorErr()
	for _, request := range queue {
		request.respond(err)
	}
}

func (e *workspaceJobExecutor) worker() {
	if e == nil {
		return
	}

	tasks := make(chan workspaceJobTask)
	for {
		select {
		case <-e.ctx.Done():
			return
		case e.workerReady <- tasks:
		}

		select {
		case <-e.ctx.Done():
			return
		case task := <-tasks:
			if e.ctx.Err() != nil {
				return
			}
			if e.run != nil {
				e.run(e.ctx, task)
			}
			if e.ctx.Err() != nil {
				return
			}
		}
	}
}

func (e *workspaceJobExecutor) executorErr() error {
	if e == nil {
		return fmt.Errorf("workspace job executor is not configured")
	}
	if err := e.ctx.Err(); err != nil {
		return err
	}
	return fmt.Errorf("workspace job executor is not accepting new work")
}

func (r workspaceJobEnqueueRequest) respond(err error) {
	select {
	case r.result <- err:
	default:
	}
}
