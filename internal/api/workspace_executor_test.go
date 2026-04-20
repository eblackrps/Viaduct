package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
	"go.uber.org/goleak"
)

func TestWorkspaceJobExecutor_BoundsConcurrency_Expected(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const (
		concurrency = 2
		jobCount    = 6
	)

	var (
		current atomic.Int32
		peak    atomic.Int32
		wg      sync.WaitGroup
	)
	started := make(chan struct{}, jobCount)
	release := make(chan struct{})
	wg.Add(jobCount)

	executor := newWorkspaceJobExecutor(ctx, concurrency, func(_ context.Context, _ workspaceJobTask) {
		running := current.Add(1)
		for {
			seen := peak.Load()
			if running <= seen || peak.CompareAndSwap(seen, running) {
				break
			}
		}
		started <- struct{}{}
		<-release
		current.Add(-1)
		wg.Done()
	})

	for index := 0; index < jobCount; index++ {
		if err := executor.Enqueue(context.Background(), workspaceJobTask{jobID: fmt.Sprintf("job-%d", index)}); err != nil {
			t.Fatalf("Enqueue(%d) error = %v", index, err)
		}
	}

	for startedCount := 0; startedCount < concurrency; startedCount++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatalf("executor started %d jobs, want %d before timeout", startedCount, concurrency)
		}
	}

	time.Sleep(100 * time.Millisecond)
	if got := peak.Load(); got != concurrency {
		t.Fatalf("peak concurrency = %d, want %d", got, concurrency)
	}
	if got := current.Load(); got != concurrency {
		t.Fatalf("running workers = %d, want %d while jobs are blocked", got, concurrency)
	}

	close(release)
	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("executor did not drain queued jobs before timeout")
	}
}

func TestServer_RecoverWorkspaceJobs_RequeuesThroughExecutor_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := mustNewServer(t, stateStore)
	recordingExecutor := &workspaceJobRecordingExecutor{}
	server.workspaceJobExecutor = recordingExecutor
	ctx := store.ContextWithTenantID(context.Background(), store.DefaultTenantID)

	if err := stateStore.CreateWorkspace(ctx, store.DefaultTenantID, models.PilotWorkspace{
		ID:     "workspace-recover-queue",
		Name:   "Recover Queue Workspace",
		Status: models.PilotWorkspaceStatusDraft,
	}); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}
	if err := stateStore.SaveWorkspaceJob(ctx, store.DefaultTenantID, models.WorkspaceJob{
		ID:            "job-requeue",
		TenantID:      store.DefaultTenantID,
		WorkspaceID:   "workspace-recover-queue",
		Type:          models.WorkspaceJobTypeGraph,
		Status:        models.WorkspaceJobStatusRunning,
		RequestedAt:   time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		StartedAt:     time.Now().UTC(),
		CorrelationID: "req-requeue",
		InputJSON:     json.RawMessage(`{"type":"graph"}`),
	}); err != nil {
		t.Fatalf("SaveWorkspaceJob() error = %v", err)
	}

	if err := server.recoverWorkspaceJobs(ctx); err != nil {
		t.Fatalf("recoverWorkspaceJobs() error = %v", err)
	}

	if len(recordingExecutor.tasks) != 1 {
		t.Fatalf("len(recordingExecutor.tasks) = %d, want 1", len(recordingExecutor.tasks))
	}
	task := recordingExecutor.tasks[0]
	if task.tenantID != store.DefaultTenantID || task.workspaceID != "workspace-recover-queue" || task.jobID != "job-requeue" {
		t.Fatalf("unexpected requeued task: %#v", task)
	}

	job, err := stateStore.GetWorkspaceJob(ctx, store.DefaultTenantID, "workspace-recover-queue", "job-requeue")
	if err != nil {
		t.Fatalf("GetWorkspaceJob() error = %v", err)
	}
	if job.Status != models.WorkspaceJobStatusQueued {
		t.Fatalf("job.Status = %s, want queued", job.Status)
	}
}

func TestWorkspaceJobExecutor_CancelledExecutorRejectsQueuedWork_Expected(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan string, 2)
	release := make(chan struct{})
	executor := newWorkspaceJobExecutor(ctx, 1, func(_ context.Context, task workspaceJobTask) {
		started <- task.jobID
		<-release
	})

	if err := executor.Enqueue(context.Background(), workspaceJobTask{jobID: "job-1"}); err != nil {
		t.Fatalf("Enqueue(job-1) error = %v", err)
	}
	select {
	case jobID := <-started:
		if jobID != "job-1" {
			t.Fatalf("started job = %s, want job-1", jobID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("executor did not start first job before timeout")
	}

	if err := executor.Enqueue(context.Background(), workspaceJobTask{jobID: "job-2"}); err != nil {
		t.Fatalf("Enqueue(job-2) error = %v", err)
	}
	cancel()

	if err := executor.Enqueue(context.Background(), workspaceJobTask{jobID: "job-3"}); !errors.Is(err, ErrExecutorShuttingDown) {
		t.Fatalf("Enqueue(job-3) error = %v, want ErrExecutorShuttingDown", err)
	}

	select {
	case jobID := <-started:
		t.Fatalf("executor started queued work after cancellation: %s", jobID)
	default:
	}

	close(release)
	select {
	case <-executor.done:
	case <-time.After(2 * time.Second):
		t.Fatal("executor did not stop promptly after canceling queued work")
	}
}

func TestWorkspaceJobExecutor_TenantQueueShareExceeded_ReturnsTypedError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	release := make(chan struct{})
	started := make(chan string, 2)
	executor := newWorkspaceJobExecutor(ctx, 2, func(_ context.Context, _ workspaceJobTask) {
		started <- "running"
		<-release
	})

	for _, jobID := range []string{"job-1", "job-2"} {
		if err := executor.Enqueue(context.Background(), workspaceJobTask{tenantID: "tenant-a", jobID: jobID}); err != nil {
			t.Fatalf("Enqueue(%s) error = %v", jobID, err)
		}
	}
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("executor did not start running jobs before queue-share assertion")
		}
	}

	for _, jobID := range []string{"job-3", "job-4"} {
		if err := executor.Enqueue(context.Background(), workspaceJobTask{tenantID: "tenant-a", jobID: jobID}); err != nil {
			t.Fatalf("Enqueue(%s) error = %v", jobID, err)
		}
	}

	if queueReporter, ok := interface{}(executor).(workspaceQueueDepthReporter); ok {
		queueDepths := queueReporter.QueueDepthByTenant()
		if queueDepths["tenant-a"] != 2 {
			t.Fatalf("queue depth = %#v, want tenant-a depth 2", queueDepths)
		}
	} else {
		t.Fatal("executor does not expose queue depth reporter")
	}

	if err := executor.Enqueue(context.Background(), workspaceJobTask{tenantID: "tenant-a", jobID: "job-5"}); !errors.Is(err, ErrTenantQueueFull) {
		t.Fatalf("Enqueue(job-5) error = %v, want ErrTenantQueueFull", err)
	}

	close(release)
}

func TestWorkspaceJobExecutor_CancelMidEnqueue_ReturnsAckOrShutdownError(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	release := make(chan struct{})
	executor := newWorkspaceJobExecutor(ctx, 1, func(_ context.Context, _ workspaceJobTask) {
		close(started)
		<-release
	})

	if err := executor.Enqueue(context.Background(), workspaceJobTask{tenantID: "tenant-a", jobID: "job-running"}); err != nil {
		t.Fatalf("Enqueue(job-running) error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("executor did not start running job before timeout")
	}

	const enqueueCount = 16
	results := make(chan error, enqueueCount)
	var wg sync.WaitGroup
	wg.Add(enqueueCount)
	for index := 0; index < enqueueCount; index++ {
		jobID := fmt.Sprintf("job-%d", index)
		tenantID := fmt.Sprintf("tenant-%d", index)
		go func(tenantID, jobID string) {
			defer wg.Done()
			results <- executor.Enqueue(context.Background(), workspaceJobTask{tenantID: tenantID, jobID: jobID})
		}(tenantID, jobID)
	}

	time.Sleep(20 * time.Millisecond)
	cancel()
	close(release)

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("enqueue operations did not return before timeout")
	}
	close(results)

	for err := range results {
		if err != nil && !errors.Is(err, ErrExecutorShuttingDown) {
			t.Fatalf("Enqueue() error = %v, want nil or ErrExecutorShuttingDown", err)
		}
	}

	select {
	case <-executor.done:
	case <-time.After(5 * time.Second):
		t.Fatal("executor did not finish dispatch loop before timeout")
	}
}

func TestWorkspaceJobExecutor_CancelWhileEnqueueStress_AllRequestsResolve_Expected(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	release := make(chan struct{})
	executor := newWorkspaceJobExecutorWithTimeout(ctx, 1, 5*time.Second, func(_ context.Context, _ workspaceJobTask) {
		close(started)
		<-release
	})

	if err := executor.Enqueue(context.Background(), workspaceJobTask{tenantID: "tenant-running", jobID: "job-running"}); err != nil {
		t.Fatalf("Enqueue(job-running) error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("executor did not start the blocking job before timeout")
	}

	const enqueueCount = 1000
	results := make(chan error, enqueueCount)
	var wg sync.WaitGroup
	wg.Add(enqueueCount)
	for index := 0; index < enqueueCount; index++ {
		tenantID := fmt.Sprintf("tenant-%d", index)
		jobID := fmt.Sprintf("job-%d", index)
		go func(tenantID, jobID string) {
			defer wg.Done()
			results <- executor.Enqueue(context.Background(), workspaceJobTask{tenantID: tenantID, jobID: jobID})
		}(tenantID, jobID)
	}

	time.Sleep(50 * time.Millisecond)
	cancel()
	close(release)

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("stress enqueue operations did not resolve before timeout")
	}
	close(results)

	for err := range results {
		if err != nil && !errors.Is(err, ErrExecutorShuttingDown) {
			t.Fatalf("Enqueue() error = %v, want nil or ErrExecutorShuttingDown", err)
		}
	}

	select {
	case <-executor.done:
	case <-time.After(5 * time.Second):
		t.Fatal("executor did not stop after stress cancellation")
	}
}

func TestWorkspaceJobExecutor_EnqueueDeadlineExceeded_ReturnsTypedError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})
	release := make(chan struct{})
	executor := newWorkspaceJobExecutorWithTimeout(ctx, 1, 25*time.Millisecond, func(_ context.Context, _ workspaceJobTask) {
		close(started)
		<-release
	})

	if err := executor.Enqueue(context.Background(), workspaceJobTask{tenantID: "tenant-a", jobID: "job-running"}); err != nil {
		t.Fatalf("Enqueue(job-running) error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("executor did not start the blocking job before timeout")
	}

	for index, jobID := range []string{"job-queued-1", "job-queued-2"} {
		if err := executor.Enqueue(context.Background(), workspaceJobTask{tenantID: fmt.Sprintf("tenant-%d", index), jobID: jobID}); err != nil {
			t.Fatalf("Enqueue(%s) error = %v", jobID, err)
		}
	}

	if err := executor.Enqueue(context.Background(), workspaceJobTask{tenantID: "tenant-b", jobID: "job-timeout"}); !errors.Is(err, ErrEnqueueTimeout) {
		t.Fatalf("Enqueue(job-timeout) error = %v, want ErrEnqueueTimeout", err)
	}
	if queueDepths := executor.QueueDepthByTenant(); queueDepths["tenant-b"] != 0 || queueDepths["tenant-0"] != 1 || queueDepths["tenant-1"] != 1 {
		t.Fatalf("QueueDepthByTenant() = %#v, want timed-out enqueue excluded from queued work", queueDepths)
	}

	close(release)
}

func TestWorkspaceJobExecutor_EnqueueDeadlineStress_DoesNotRetainTimedOutTasks_Expected(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})
	release := make(chan struct{})
	executor := newWorkspaceJobExecutorWithTimeout(ctx, 1, 50*time.Millisecond, func(_ context.Context, _ workspaceJobTask) {
		close(started)
		<-release
	})

	if err := executor.Enqueue(context.Background(), workspaceJobTask{tenantID: "tenant-running", jobID: "job-running"}); err != nil {
		t.Fatalf("Enqueue(job-running) error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("executor did not start the blocking job before timeout")
	}

	const callerCount = 1000
	results := make(chan error, callerCount)
	var wg sync.WaitGroup
	wg.Add(callerCount)
	for index := 0; index < callerCount; index++ {
		tenantID := fmt.Sprintf("tenant-%d", index)
		jobID := fmt.Sprintf("job-%d", index)
		go func(tenantID, jobID string) {
			defer wg.Done()
			results <- executor.Enqueue(context.Background(), workspaceJobTask{tenantID: tenantID, jobID: jobID})
		}(tenantID, jobID)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("enqueue timeout stress callers did not resolve before timeout")
	}
	close(results)

	successes := 0
	timeouts := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrEnqueueTimeout):
			timeouts++
		default:
			t.Fatalf("Enqueue() error = %v, want nil or ErrEnqueueTimeout", err)
		}
	}
	if timeouts == 0 {
		t.Fatal("enqueue timeout stress produced zero timeout errors, want at least one")
	}

	totalQueued := 0
	for _, depth := range executor.QueueDepthByTenant() {
		totalQueued += depth
	}
	if totalQueued != successes {
		t.Fatalf("queued depth total = %d, want %d admitted tasks with timed-out requests removed", totalQueued, successes)
	}

	cancel()
	close(release)
	select {
	case <-executor.done:
	case <-time.After(2 * time.Second):
		t.Fatal("executor did not stop after timeout stress cancellation")
	}
	if queueDepths := executor.QueueDepthByTenant(); len(queueDepths) != 0 {
		t.Fatalf("QueueDepthByTenant() after shutdown = %#v, want empty", queueDepths)
	}
}

func TestWorkspaceJobEnqueueRequest_ClosedResultChannelReturnsTypedError(t *testing.T) {
	t.Parallel()

	result := make(chan error)
	close(result)

	request := workspaceJobEnqueueRequest{
		waitCtx: context.Background(),
		result:  result,
	}
	if err := request.respond(nil); !errors.Is(err, ErrResultChannelFull) {
		t.Fatalf("respond(nil) error = %v, want ErrResultChannelFull", err)
	}
}

func TestWorkspaceJobExecutor_TaskContextCancelledOnShutdown_Expected(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})
	taskDone := make(chan struct{})
	executor := newWorkspaceJobExecutor(ctx, 1, func(runCtx context.Context, _ workspaceJobTask) {
		close(started)
		<-runCtx.Done()
		close(taskDone)
	})

	if err := executor.Enqueue(context.Background(), workspaceJobTask{tenantID: "tenant-a", jobID: "job-1", ctx: context.Background()}); err != nil {
		t.Fatalf("Enqueue(job-1) error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("executor did not start task before cancellation")
	}

	cancel()

	select {
	case <-taskDone:
	case <-time.After(2 * time.Second):
		t.Fatal("task context was not canceled when executor stopped")
	}
}

type workspaceJobRecordingExecutor struct {
	mu    sync.Mutex
	tasks []workspaceJobTask
	err   error
}

func (e *workspaceJobRecordingExecutor) Enqueue(_ context.Context, task workspaceJobTask) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tasks = append(e.tasks, task)
	return e.err
}
