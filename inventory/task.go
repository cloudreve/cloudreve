package inventory

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/gofrs/uuid"
	"github.com/samber/lo"
)

type (
	// Ctx keys for eager loading options.
	LoadTaskUser struct{}

	TaskArgs struct {
		Status        task.Status
		Type          string
		PublicState   *types.TaskPublicState
		PrivateState  string
		OwnerID       int
		CorrelationID uuid.UUID
	}
)

type TaskClient interface {
	TxOperator
	// New creates a new task with the given args.
	New(ctx context.Context, task *TaskArgs) (*ent.Task, error)
	// Update updates the task with the given args.
	Update(ctx context.Context, task *ent.Task, args *TaskArgs) (*ent.Task, error)
	// GetPendingTasks returns all pending tasks of given type.
	GetPendingTasks(ctx context.Context, taskType ...string) ([]*ent.Task, error)
	// GetTaskByID returns the task with the given ID.
	GetTaskByID(ctx context.Context, taskID int) (*ent.Task, error)
	// SetCompleteByID sets the task with the given ID to complete.
	SetCompleteByID(ctx context.Context, taskID int) error
	// List returns a list of tasks with the given args.
	List(ctx context.Context, args *ListTaskArgs) (*ListTaskResult, error)
	// DeleteByIDs deletes the tasks with the given IDs.
	DeleteByIDs(ctx context.Context, ids ...int) error
	// DeleteBy deletes the tasks with the given args.
	DeleteBy(ctx context.Context, args *DeleteTaskArgs) error
}

type (
	ListTaskArgs struct {
		*PaginationArgs
		Types         []string
		Status        []task.Status
		UserID        int
		CorrelationID *uuid.UUID
	}

	ListTaskResult struct {
		*PaginationResults
		Tasks []*ent.Task
	}

	DeleteTaskArgs struct {
		NotAfter time.Time
		Types    []string
		Status   []task.Status
	}
)

func NewTaskClient(client *ent.Client, dbType conf.DBType, hasher hashid.Encoder) TaskClient {
	return &taskClient{client: client, maxSQlParam: sqlParamLimit(dbType), hasher: hasher}
}

type taskClient struct {
	maxSQlParam int
	hasher      hashid.Encoder
	client      *ent.Client
}

func (c *taskClient) SetClient(newClient *ent.Client) TxOperator {
	return &taskClient{client: newClient, maxSQlParam: c.maxSQlParam, hasher: c.hasher}
}

func (c *taskClient) GetClient() *ent.Client {
	return c.client
}

func (c *taskClient) New(ctx context.Context, task *TaskArgs) (*ent.Task, error) {
	stm := c.client.Task.
		Create().
		SetType(task.Type).
		SetPublicState(task.PublicState)
	if task.PrivateState != "" {
		stm.SetPrivateState(task.PrivateState)
	}

	if task.OwnerID != 0 {
		stm.SetUserID(task.OwnerID)
	}

	if task.Status != "" {
		stm.SetStatus(task.Status)
	}

	if task.CorrelationID.String() != uuid.Nil.String() {
		stm.SetCorrelationID(task.CorrelationID)
	}

	newTask, err := stm.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return newTask, nil
}

func (c *taskClient) DeleteByIDs(ctx context.Context, ids ...int) error {
	_, err := c.client.Task.Delete().Where(task.IDIn(ids...)).Exec(ctx)
	return err
}

func (c *taskClient) DeleteBy(ctx context.Context, args *DeleteTaskArgs) error {
	query := c.client.Task.
		Delete().
		Where(task.CreatedAtLTE(args.NotAfter))

	if len(args.Status) > 0 {
		query.Where(task.StatusIn(args.Status...))
	}

	if len(args.Types) > 0 {
		query.Where(task.TypeIn(args.Types...))
	}

	_, err := query.Exec(ctx)
	return err
}

func (c *taskClient) Update(ctx context.Context, task *ent.Task, args *TaskArgs) (*ent.Task, error) {
	stm := c.client.Task.UpdateOne(task).
		SetPublicState(args.PublicState)

	task.PublicState = args.PublicState

	if task.PrivateState != "" {
		stm.SetPrivateState(task.PrivateState)
		task.PrivateState = args.PrivateState
	}

	if task.Status != "" {
		stm.SetStatus(args.Status)
		task.Status = args.Status
	}

	if err := stm.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

func (c *taskClient) GetPendingTasks(ctx context.Context, taskType ...string) ([]*ent.Task, error) {
	tasks, err := withTaskEagerLoading(ctx, c.client.Task.Query()).
		Where(task.StatusIn(task.StatusProcessing, task.StatusQueued, task.StatusSuspending)).
		Where(task.TypeIn(taskType...)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// Anonymous user is not loaded by default, so we need to load it manually.
	userClient := NewUserClient(c.client)
	anonymous, err := userClient.AnonymousUser(ctx)
	for _, t := range tasks {
		if t.UserTasks == 0 {
			if err != nil {
				return nil, err
			}
			t.SetUser(anonymous)
		}
	}

	return tasks, nil
}

func (c *taskClient) GetTaskByID(ctx context.Context, taskID int) (*ent.Task, error) {
	return withTaskEagerLoading(ctx, c.client.Task.Query()).
		Where(task.ID(taskID)).
		First(ctx)
}

func (c *taskClient) SetCompleteByID(ctx context.Context, taskID int) error {
	_, err := c.client.Task.UpdateOneID(taskID).
		SetStatus(task.StatusCompleted).
		Save(ctx)
	return err
}

func (c *taskClient) List(ctx context.Context, args *ListTaskArgs) (*ListTaskResult, error) {
	q := c.client.Task.Query()
	if args.UserID != 0 {
		q.Where(task.UserTasks(args.UserID))
	}

	if args.Types != nil {
		q.Where(task.TypeIn(args.Types...))
	}

	if args.Status != nil {
		q.Where(task.StatusIn(args.Status...))
	}

	if args.CorrelationID != nil {
		q.Where(task.CorrelationID(*args.CorrelationID))
	}

	q = withTaskEagerLoading(ctx, q)
	var (
		tasks         []*ent.Task
		err           error
		paginationRes *PaginationResults
	)

	if args.UseCursorPagination {
		tasks, paginationRes, err = c.cursorPagination(ctx, q, args, 1)
	} else {
		tasks, paginationRes, err = c.offsetPagination(ctx, q, args, 1)
	}

	if err != nil {
		return nil, fmt.Errorf("query failed with paginiation: %w", err)
	}

	return &ListTaskResult{
		Tasks:             tasks,
		PaginationResults: paginationRes,
	}, nil
}

func (c *taskClient) cursorPagination(ctx context.Context, query *ent.TaskQuery, args *ListTaskArgs, paramMargin int) ([]*ent.Task, *PaginationResults, error) {
	pageSize := capPageSize(c.maxSQlParam, args.PageSize, paramMargin)
	query.Order(task.ByID(sql.OrderDesc()))

	var (
		pageToken  *PageToken
		err        error
		queryPaged = query
	)
	if args.PageToken != "" {
		pageToken, err = pageTokenFromString(args.PageToken, c.hasher, hashid.TaskID)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid page token %q: %w", args.PageToken, err)
		}

		queryPaged = query.Where(task.IDLT(pageToken.ID))
	}

	// Use page size + 1 to determine if there are more items to come
	queryPaged.Limit(pageSize + 1)

	tasks, err := queryPaged.
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	// More items to come
	nextTokenStr := ""
	if len(tasks) > pageSize {
		lastItem := tasks[len(tasks)-2]
		nextToken, err := getTaskNextPageToken(c.hasher, lastItem)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate next page token: %w", err)
		}

		nextTokenStr = nextToken
	}

	return lo.Subset(tasks, 0, uint(pageSize)), &PaginationResults{
		PageSize:      pageSize,
		NextPageToken: nextTokenStr,
		IsCursor:      true,
	}, nil
}

func (c *taskClient) offsetPagination(ctx context.Context, query *ent.TaskQuery, args *ListTaskArgs, paramMargin int) ([]*ent.Task, *PaginationResults, error) {
	pageSize := capPageSize(c.maxSQlParam, args.PageSize, paramMargin)
	query.Order(getTaskOrderOption(args)...)

	// Count total items
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	logs, err := query.
		Limit(pageSize).
		Offset(args.Page * args.PageSize).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return logs, &PaginationResults{
		PageSize:   pageSize,
		TotalItems: total,
		Page:       args.Page,
	}, nil

}

func getTaskOrderOption(args *ListTaskArgs) []task.OrderOption {
	orderTerm := getOrderTerm(args.Order)
	switch args.OrderBy {
	default:
		return []task.OrderOption{task.ByID(orderTerm)}
	}
}

// getTaskNextPageToken returns the next page token for the given last task.
func getTaskNextPageToken(hasher hashid.Encoder, last *ent.Task) (string, error) {
	token := &PageToken{
		ID: last.ID,
	}

	return token.Encode(hasher, hashid.EncodeTaskID)
}

func withTaskEagerLoading(ctx context.Context, q *ent.TaskQuery) *ent.TaskQuery {
	if v, ok := ctx.Value(LoadTaskUser{}).(bool); ok && v {
		q.WithUser(func(q *ent.UserQuery) {
			withUserEagerLoading(ctx, q)
		})
	}

	return q
}
