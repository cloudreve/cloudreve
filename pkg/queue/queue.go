package queue

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/jpillora/backoff"
)

type (
	Queue interface {
		// Start resume tasks and starts all workers.
		Start()
		// Shutdown stops all workers.
		Shutdown()
		// SubmitTask submits a Task to the queue.
		QueueTask(ctx context.Context, t Task) error
		// CancelTask marks a queued or suspending task as canceled. The
		// scheduler entry is left in place; workers skip canceled tasks at
		// pickup. Returns false if the task is not in this queue's registry.
		CancelTask(ctx context.Context, taskID int) bool
		// BusyWorkers returns the numbers of workers in the running process.
		BusyWorkers() int
		// BusyWorkers returns the numbers of success tasks.
		SuccessTasks() int
		// FailureTasks returns the numbers of failure tasks.
		FailureTasks() int
		// SubmittedTasks returns the numbers of submitted tasks.
		SubmittedTasks() int
		// SuspendingTasks returns the numbers of suspending tasks.
		SuspendingTasks() int
	}
	queue struct {
		sync.Mutex
		routineGroup *routineGroup
		metric       *metric
		quit         chan struct{}
		ready        chan struct{}
		scheduler    Scheduler
		stopOnce     sync.Once
		stopFlag     int32
		rootCtx      context.Context
		cancel       context.CancelFunc

		// Dependencies
		logger     logging.Logger
		taskClient inventory.TaskClient
		dep        Dep
		registry   TaskRegistry

		// Options
		*options
	}

	Dep interface {
		ForkWithLogger(ctx context.Context, l logging.Logger) context.Context
	}
)

var (
	CriticalErr = errors.New("non-retryable error")
)

func New(l logging.Logger, taskClient inventory.TaskClient, registry TaskRegistry, dep Dep, opts ...Option) Queue {
	o := newDefaultOptions()
	for _, opt := range opts {
		opt.apply(o)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &queue{
		routineGroup: newRoutineGroup(),
		scheduler:    NewFifoScheduler(0, l),
		quit:         make(chan struct{}),
		ready:        make(chan struct{}, 1),
		metric:       &metric{},
		options:      o,
		logger:       l,
		registry:     registry,
		taskClient:   taskClient,
		dep:          dep,
		rootCtx:      ctx,
		cancel:       cancel,
	}
}

// Start to enable all worker
func (q *queue) Start() {
	q.routineGroup.Run(func() {
		// Resume tasks in DB
		if len(q.options.resumeTaskType) > 0 && q.taskClient != nil {

			ctx := context.TODO()
			ctx = context.WithValue(ctx, inventory.LoadTaskUser{}, true)
			ctx = context.WithValue(ctx, inventory.LoadUserGroup{}, true)
			tasks, err := q.taskClient.GetPendingTasks(ctx, q.resumeTaskType...)
			if err != nil {
				q.logger.Warning("Failed to get pending tasks from DB for given type %v: %s", q.resumeTaskType, err)
			}

			resumed := 0
			for _, t := range tasks {
				resumedTask, err := NewTaskFromModel(t)
				if err != nil {
					q.logger.Warning("Failed to resume task %d: %s", t.ID, err)
					continue
				}

				if resumedTask.Status() == task.StatusSuspending {
					q.metric.IncSuspendingTask()
					q.metric.IncSubmittedTask()
				}

				if err := q.QueueTask(ctx, resumedTask); err != nil {
					q.logger.Warning("Failed to resume task %d: %s", t.ID, err)
				}
				resumed++
			}

			q.logger.Info("Resumed %d tasks from DB.", resumed)
		}

		q.start()
	})
	q.logger.Info("Queue %q started with %d workers.", q.name, q.workerCount)
}

// Shutdown stops all queues.
func (q *queue) Shutdown() {
	q.logger.Info("Shutting down queue %q...", q.name)
	defer func() {
		q.routineGroup.Wait()
	}()

	if !atomic.CompareAndSwapInt32(&q.stopFlag, 0, 1) {
		return
	}

	q.stopOnce.Do(func() {
		q.cancel()
		if q.metric.BusyWorkers() > 0 {
			q.logger.Info("shutdown all tasks in queue %q: %d workers", q.name, q.metric.BusyWorkers())
		}

		if err := q.scheduler.Shutdown(); err != nil {
			q.logger.Error("failed to shutdown scheduler in queue %q: %w", q.name, err)
		}
		close(q.quit)
	})

}

// BusyWorkers returns the numbers of workers in the running process.
func (q *queue) BusyWorkers() int {
	return int(q.metric.BusyWorkers())
}

// BusyWorkers returns the numbers of success tasks.
func (q *queue) SuccessTasks() int {
	return int(q.metric.SuccessTasks())
}

// BusyWorkers returns the numbers of failure tasks.
func (q *queue) FailureTasks() int {
	return int(q.metric.FailureTasks())
}

// BusyWorkers returns the numbers of submitted tasks.
func (q *queue) SubmittedTasks() int {
	return int(q.metric.SubmittedTasks())
}

// SuspendingTasks returns the numbers of suspending tasks.
func (q *queue) SuspendingTasks() int {
	return int(q.metric.SuspendingTasks())
}

// QueueTask to queue single Task
func (q *queue) QueueTask(ctx context.Context, t Task) error {
	if atomic.LoadInt32(&q.stopFlag) == 1 {
		return ErrQueueShutdown
	}

	if t.Status() != task.StatusSuspending {
		q.metric.IncSubmittedTask()
		if err := q.transitStatus(ctx, t, task.StatusQueued); err != nil {
			return err
		}
	}

	if err := q.scheduler.Queue(t); err != nil {
		return err
	}
	owner := ""
	if t.Owner() != nil {
		owner = t.Owner().Email
	}
	q.logger.Info("New Task with type %q submitted to queue %q by %q", t.Type(), q.name, owner)
	if q.registry != nil {
		q.registry.Set(t.ID(), t)
	}

	return nil
}

// newContext creates a new context for a new Task iteration.
func (q *queue) newContext(t Task) context.Context {
	l := q.logger.CopyWithPrefix(fmt.Sprintf("[Cid: %s TaskID: %d Queue: %s]", t.CorrelationID(), t.ID(), q.name))
	ctx := q.dep.ForkWithLogger(q.rootCtx, l)
	ctx = context.WithValue(ctx, logging.CorrelationIDCtx{}, t.CorrelationID())
	ctx = context.WithValue(ctx, logging.LoggerCtx{}, l)
	ctx = context.WithValue(ctx, inventory.UserCtx{}, t.Owner())
	return ctx
}

// CancelTask marks a queued or suspending task as canceled. The scheduler
// heap entry is not removed; workers skip canceled tasks at pickup.
func (q *queue) CancelTask(ctx context.Context, taskID int) bool {
	if q.registry == nil {
		return false
	}
	t, found := q.registry.Get(taskID)
	if !found {
		return false
	}
	if t.Status() != task.StatusQueued && t.Status() != task.StatusSuspending {
		return false
	}
	return q.transitStatus(ctx, t, task.StatusCanceled) == nil
}

func (q *queue) work(t Task) {
	ctx := q.newContext(t)
	l := logging.FromContext(ctx)
	timeIterationStart := time.Now()

	var err error
	// to handle panic cases from inside the worker
	// in such case, we start a new goroutine
	defer func() {
		q.metric.DecBusyWorker()
		e := recover()
		if e != nil {
			l.Error("Panic error in queue %q: %v", q.name, e)
			t.OnError(fmt.Errorf("panic error: %v", e), time.Since(timeIterationStart))

			_ = q.transitStatus(ctx, t, task.StatusError)
		}
		q.schedule()
	}()

	// Canceled while waiting in the scheduler; nothing to run.
	if t.Status() == task.StatusCanceled {
		return
	}

	err = q.transitStatus(ctx, t, task.StatusProcessing)
	if err != nil {
		l.Error("failed to transit task %d to processing: %s", t.ID(), err.Error())
		panic(err)
	}

	for {
		timeIterationStart = time.Now()
		var next task.Status
		next, err = q.run(ctx, t)
		if err != nil {
			t.OnError(err, time.Since(timeIterationStart))
			l.Error("runtime error in queue %q: %s", q.name, err.Error())

			_ = q.transitStatus(ctx, t, task.StatusError)
			break
		}

		// iteration completes
		t.OnIterationComplete(time.Since(timeIterationStart))
		_ = q.transitStatus(ctx, t, next)
		if next != task.StatusProcessing {
			break
		}
	}
}

func (q *queue) run(ctx context.Context, t Task) (task.Status, error) {
	l := logging.FromContext(ctx)

	// create channel with buffer size 1 to avoid goroutine leak
	done := make(chan struct {
		err  error
		next task.Status
	}, 1)
	panicChan := make(chan interface{}, 1)
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(ctx, q.maxTaskExecution-t.Executed())
	defer func() {
		cancel()
	}()

	// run the job
	go func() {
		// handle panic issue
		defer func() {
			if p := recover(); p != nil {
				q.logger.Error("panic in queue %q: %s", q.name, debug.Stack())
				panicChan <- p
			}
		}()

		l.Debug("Iteration started.")
		next, err := t.Do(ctx)
		l.Debug("Iteration ended with err=%s", err)
		if err != nil && q.maxRetry-t.Retried() > 0 && !errors.Is(err, CriticalErr) && atomic.LoadInt32(&q.stopFlag) != 1 {
			// Retry needed
			t.OnRetry(err)
			b := &backoff.Backoff{
				Max:    q.backoffMaxDuration,
				Factor: q.backoffFactor,
			}
			delay := q.retryDelay
			if q.retryDelay == 0 {
				delay = b.ForAttempt(float64(t.Retried()))
			}

			// Resume after to retry
			l.Info("Will be retried in %s", delay)
			t.OnSuspend(time.Now().Add(delay).Unix())
			err = nil
			next = task.StatusSuspending
		}

		done <- struct {
			err  error
			next task.Status
		}{err: err, next: next}
	}()

	select {
	case p := <-panicChan:
		panic(p)
	case <-ctx.Done(): // timeout reached
		return task.StatusError, ctx.Err()
	case <-q.quit: // shutdown service
		// cancel job
		cancel()

		leftTime := q.maxTaskExecution - t.Executed() - time.Since(startTime)
		// wait job
		select {
		case <-time.After(leftTime):
			return task.StatusError, context.DeadlineExceeded
		case r := <-done: // job finish
			return r.next, r.err
		case p := <-panicChan:
			panic(p)
		}
	case r := <-done: // job finish
		return r.next, r.err
	}
}

// beforeTaskStart updates Task status from queued to processing
func (q *queue) transitStatus(ctx context.Context, task Task, to task.Status) (err error) {
	old := task.Status()
	transition, ok := stateTransitions[task.Status()][to]
	if !ok {
		err = fmt.Errorf("invalid state transition from %s to %s", old, to)
	} else {
		if innerErr := transition(ctx, task, to, q); innerErr != nil {
			err = fmt.Errorf("failed to transit Task status from %s to %s: %w", old, to, innerErr)
		}
	}

	l := logging.FromContext(ctx)
	if err != nil {
		l.Error(err.Error())
	}

	l.Info("Task %d status changed from %q to %q.", task.ID(), old, to)
	return
}

// schedule to check worker number
func (q *queue) schedule() {
	q.Lock()
	defer q.Unlock()
	if q.BusyWorkers() >= q.workerCount {
		return
	}

	select {
	case q.ready <- struct{}{}:
	default:
	}
}

// start to start all worker
func (q *queue) start() {
	tasks := make(chan Task, 1)

	for {
		// check worker number
		q.schedule()

		select {
		// wait worker ready
		case <-q.ready:
		case <-q.quit:
			return
		}

		// request Task from queue in background
		q.routineGroup.Run(func() {
			for {
				t, err := q.scheduler.Request()
				if t == nil || err != nil {
					if err != nil {
						select {
						case <-q.quit:
							if !errors.Is(err, ErrNoTaskInQueue) {
								close(tasks)
								return
							}
						case <-time.After(q.taskPullInterval):
							// sleep to fetch new Task
						}
					}
				}
				if t != nil {
					tasks <- t
					return
				}

				select {
				case <-q.quit:
					if !errors.Is(err, ErrNoTaskInQueue) {
						close(tasks)
						return
					}
				default:
				}
			}
		})

		t, ok := <-tasks
		if !ok {
			return
		}

		// start new Task
		q.metric.IncBusyWorker()
		q.routineGroup.Run(func() {
			q.work(t)
		})
	}
}
