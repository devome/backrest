package orchestrator

import (
	"context"
	"sync"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
)

type testTask struct {
	onRun  func() error
	onNext func(curTime time.Time) *time.Time
}

func (t *testTask) Name() string {
	return "test"
}

func (t *testTask) Next(now time.Time) *time.Time {
	return t.onNext(now)
}

func (t *testTask) Run(ctx context.Context) error {
	return t.onRun()
}

func (t *testTask) Cancel(withStatus v1.OperationStatus) error {
	return nil
}

func (t *testTask) OperationId() int64 {
	return 0
}

func TestTaskScheduling(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orch, err := NewOrchestrator("", config.NewDefaultConfig(), nil, nil)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	task := &testTask{
		onRun: func() error {
			wg.Done()
			cancel()
			return nil
		},
		onNext: func(t time.Time) *time.Time {
			t = t.Add(10 * time.Millisecond)
			return &t
		},
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		orch.Run(ctx)
	}()

	// Act
	orch.ScheduleTask(task, TaskPriorityDefault)

	// Assert passes if all tasks run and the orchestrator exists when cancelled.
	wg.Wait()
}

func TestTaskRescheduling(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orch, err := NewOrchestrator("", config.NewDefaultConfig(), nil, nil)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		orch.Run(ctx)
	}()

	// Act
	count := 0
	ranTimes := 0

	orch.ScheduleTask(&testTask{
		onNext: func(t time.Time) *time.Time {
			if count < 10 {
				count += 1
				return &t
			}
			return nil
		},
		onRun: func() error {
			ranTimes += 1
			if ranTimes == 10 {
				cancel()
			}
			return nil
		},
	}, TaskPriorityDefault)

	wg.Wait()

	if count != 10 {
		t.Errorf("expected 10 Next calls, got %d", count)
	}

	if ranTimes != 10 {
		t.Errorf("expected 10 Run calls, got %d", ranTimes)
	}
}

func TestGracefulShutdown(t *testing.T) {
	t.Parallel()

	// Arrange
	orch, err := NewOrchestrator("", config.NewDefaultConfig(), nil, nil)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	// Act
	orch.Run(ctx)
}

func TestSchedulerWait(t *testing.T) {
	t.Parallel()

	// Arrange
	orch, err := NewOrchestrator("", config.NewDefaultConfig(), nil, nil)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}
	orch.taskQueue.Reset()

	ran := make(chan struct{})
	didRun := false
	orch.ScheduleTask(&testTask{
		onNext: func(t time.Time) *time.Time {
			if didRun {
				return nil
			}
			t = t.Add(100 * time.Millisecond)
			didRun = true
			return &t
		},
		onRun: func() error {
			close(ran)
			return nil
		},
	}, TaskPriorityDefault)

	// Act
	go orch.Run(context.Background())

	// Assert
	select {
	case <-time.NewTimer(20 * time.Millisecond).C:
	case <-ran:
		t.Errorf("expected task to not run yet")
	}

	// Schedule another task just to trigger a queue refresh
	orch.ScheduleTask(&testTask{
		onNext: func(t time.Time) *time.Time {
			t = t.Add(1000 * time.Second)
			return &t
		},
		onRun: func() error {
			t.Fatalf("should never run")
			return nil
		},
	}, TaskPriorityDefault)

	select {
	case <-time.NewTimer(200 * time.Millisecond).C:
		t.Errorf("expected task to run")
	case <-ran:
	}
}
