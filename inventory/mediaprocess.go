package inventory

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/mediaprocesstask"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
)

// MediaProcessClient is the data-access layer for the media_process_task table
// (APP-101): the queue of blobs pending deferred post-processing (compression).
type MediaProcessClient interface {
	TxOperator
	// Enqueue registers a blob as pending for the given media type. It is
	// idempotent: if an active (pending/processing) row already exists for the
	// entity, it returns that row instead of creating a duplicate.
	Enqueue(ctx context.Context, args *MediaProcessEnqueueArgs) (*ent.MediaProcessTask, error)
	// HasActive reports whether an active (pending/processing) row exists for the
	// given entity id.
	HasActive(ctx context.Context, entityID int) (bool, error)
	// HasHandledForFile reports whether a terminal (done/skipped) row already
	// exists for the given file id. Used by the backfill sweep to avoid
	// re-enqueuing files it has already processed once.
	HasHandledForFile(ctx context.Context, fileID int) (bool, error)
	// ListPending returns up to limit pending rows of the given media type,
	// oldest first.
	ListPending(ctx context.Context, mediaType mediaprocesstask.MediaType, limit int) ([]*ent.MediaProcessTask, error)
	// GetByID returns a row by its id.
	GetByID(ctx context.Context, id int) (*ent.MediaProcessTask, error)
	// SetStatus transitions a row to the given status, bumping attempts and
	// persisting error/result_size/props when supplied.
	SetStatus(ctx context.Context, id int, args *MediaProcessStatusArgs) (*ent.MediaProcessTask, error)
}

type (
	MediaProcessEnqueueArgs struct {
		EntityID  int
		FileID    int
		OwnerID   int
		MediaType mediaprocesstask.MediaType
		Props     *types.MediaProcessTaskProps
	}

	MediaProcessStatusArgs struct {
		Status       mediaprocesstask.Status
		BumpAttempts bool
		Error        string
		ResultSize   int64
		Props        *types.MediaProcessTaskProps
	}
)

func NewMediaProcessClient(client *ent.Client, dbType conf.DBType) MediaProcessClient {
	return &mediaProcessClient{client: client, maxSQlParam: sqlParamLimit(dbType)}
}

type mediaProcessClient struct {
	maxSQlParam int
	client      *ent.Client
}

func (c *mediaProcessClient) SetClient(newClient *ent.Client) TxOperator {
	return &mediaProcessClient{client: newClient, maxSQlParam: c.maxSQlParam}
}

func (c *mediaProcessClient) GetClient() *ent.Client {
	return c.client
}

func (c *mediaProcessClient) HasActive(ctx context.Context, entityID int) (bool, error) {
	return c.client.MediaProcessTask.Query().
		Where(
			mediaprocesstask.EntityID(entityID),
			mediaprocesstask.StatusIn(mediaprocesstask.StatusPending, mediaprocesstask.StatusProcessing),
		).
		Exist(ctx)
}

func (c *mediaProcessClient) HasHandledForFile(ctx context.Context, fileID int) (bool, error) {
	if fileID == 0 {
		return false, nil
	}
	return c.client.MediaProcessTask.Query().
		Where(
			mediaprocesstask.FileID(fileID),
			mediaprocesstask.StatusIn(mediaprocesstask.StatusDone, mediaprocesstask.StatusSkipped),
		).
		Exist(ctx)
}

func (c *mediaProcessClient) Enqueue(ctx context.Context, args *MediaProcessEnqueueArgs) (*ent.MediaProcessTask, error) {
	mediaType := args.MediaType
	if mediaType == "" {
		mediaType = mediaprocesstask.MediaTypeImage
	}

	// Idempotency: reuse an existing active row for this entity.
	existing, err := c.client.MediaProcessTask.Query().
		Where(
			mediaprocesstask.EntityID(args.EntityID),
			mediaprocesstask.StatusIn(mediaprocesstask.StatusPending, mediaprocesstask.StatusProcessing),
		).
		First(ctx)
	if err == nil {
		return existing, nil
	}
	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query active media process task: %w", err)
	}

	stm := c.client.MediaProcessTask.Create().
		SetEntityID(args.EntityID).
		SetOwnerID(args.OwnerID).
		SetMediaType(mediaType).
		SetStatus(mediaprocesstask.StatusPending)
	if args.FileID != 0 {
		stm.SetFileID(args.FileID)
	}
	if args.Props != nil {
		stm.SetProps(args.Props)
	}

	created, err := stm.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue media process task: %w", err)
	}

	return created, nil
}

func (c *mediaProcessClient) ListPending(ctx context.Context, mediaType mediaprocesstask.MediaType, limit int) ([]*ent.MediaProcessTask, error) {
	if limit <= 0 {
		limit = 50
	}
	if mediaType == "" {
		mediaType = mediaprocesstask.MediaTypeImage
	}

	return c.client.MediaProcessTask.Query().
		Where(
			mediaprocesstask.StatusEQ(mediaprocesstask.StatusPending),
			mediaprocesstask.MediaTypeEQ(mediaType),
		).
		Order(ent.Asc(mediaprocesstask.FieldID)).
		Limit(limit).
		All(ctx)
}

func (c *mediaProcessClient) GetByID(ctx context.Context, id int) (*ent.MediaProcessTask, error) {
	return c.client.MediaProcessTask.Get(ctx, id)
}

func (c *mediaProcessClient) SetStatus(ctx context.Context, id int, args *MediaProcessStatusArgs) (*ent.MediaProcessTask, error) {
	stm := c.client.MediaProcessTask.UpdateOneID(id).
		SetStatus(args.Status)
	if args.BumpAttempts {
		stm.AddAttempts(1)
	}
	if args.Error != "" {
		stm.SetError(args.Error)
	}
	if args.ResultSize > 0 {
		stm.SetResultSize(args.ResultSize)
	}
	if args.Props != nil {
		stm.SetProps(args.Props)
	}

	updated, err := stm.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update media process task status: %w", err)
	}

	return updated, nil
}
