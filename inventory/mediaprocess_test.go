package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent/mediaprocesstask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMediaProcessEnqueueIdempotent covers APP-101: Enqueue is idempotent per
// active entity, ListPending filters by status+media_type, and SetStatus(done)
// clears the active guard so a fresh Enqueue is allowed again.
func TestMediaProcessEnqueueIdempotent(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	c := NewMediaProcessClient(client, "sqlite")

	// First enqueue creates a pending row.
	row1, err := c.Enqueue(ctx, &MediaProcessEnqueueArgs{
		EntityID: 101, FileID: 11, OwnerID: 1, MediaType: mediaprocesstask.MediaTypeImage,
	})
	require.NoError(t, err)
	assert.Equal(t, mediaprocesstask.StatusPending, row1.Status)
	assert.Equal(t, 101, row1.EntityID)

	// Second enqueue for the same entity returns the same row (no duplicate).
	row2, err := c.Enqueue(ctx, &MediaProcessEnqueueArgs{
		EntityID: 101, FileID: 11, OwnerID: 1, MediaType: mediaprocesstask.MediaTypeImage,
	})
	require.NoError(t, err)
	assert.Equal(t, row1.ID, row2.ID, "duplicate enqueue must reuse the active row")

	active, err := c.HasActive(ctx, 101)
	require.NoError(t, err)
	assert.True(t, active)

	// A different entity is a separate row.
	_, err = c.Enqueue(ctx, &MediaProcessEnqueueArgs{
		EntityID: 202, FileID: 22, OwnerID: 1, MediaType: mediaprocesstask.MediaTypeImage,
	})
	require.NoError(t, err)

	// ListPending returns both pending image rows.
	pending, err := c.ListPending(ctx, mediaprocesstask.MediaTypeImage, 50)
	require.NoError(t, err)
	assert.Len(t, pending, 2)

	// Video pending is empty (discriminator filter).
	vids, err := c.ListPending(ctx, mediaprocesstask.MediaTypeVideo, 50)
	require.NoError(t, err)
	assert.Len(t, vids, 0)

	// Marking entity 101 done clears the active guard and drops it from pending.
	_, err = c.SetStatus(ctx, row1.ID, &MediaProcessStatusArgs{Status: mediaprocesstask.StatusDone, ResultSize: 1234})
	require.NoError(t, err)

	active, err = c.HasActive(ctx, 101)
	require.NoError(t, err)
	assert.False(t, active, "done row must not count as active")

	pending, err = c.ListPending(ctx, mediaprocesstask.MediaTypeImage, 50)
	require.NoError(t, err)
	assert.Len(t, pending, 1, "only the still-pending entity remains")

	// After done, a new enqueue for the same entity is allowed (re-upload case).
	row3, err := c.Enqueue(ctx, &MediaProcessEnqueueArgs{
		EntityID: 101, FileID: 11, OwnerID: 1, MediaType: mediaprocesstask.MediaTypeImage,
	})
	require.NoError(t, err)
	assert.NotEqual(t, row1.ID, row3.ID, "a new pending row is created after the previous one is done")
}

// TestMediaProcessEnqueueVideo covers APP-103: video rows are enqueued and listed
// independently of image rows via the media_type discriminator, sharing the same
// idempotency (active-row) and terminal-state semantics.
func TestMediaProcessEnqueueVideo(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	c := NewMediaProcessClient(client, "sqlite")

	// An image and a video row coexist on the same table.
	img, err := c.Enqueue(ctx, &MediaProcessEnqueueArgs{
		EntityID: 10, FileID: 1, OwnerID: 1, MediaType: mediaprocesstask.MediaTypeImage,
	})
	require.NoError(t, err)
	vid, err := c.Enqueue(ctx, &MediaProcessEnqueueArgs{
		EntityID: 20, FileID: 2, OwnerID: 1, MediaType: mediaprocesstask.MediaTypeVideo,
	})
	require.NoError(t, err)
	assert.Equal(t, mediaprocesstask.MediaTypeVideo, vid.MediaType)

	// ListPending is scoped by the discriminator: each lane sees only its own rows.
	imgs, err := c.ListPending(ctx, mediaprocesstask.MediaTypeImage, 50)
	require.NoError(t, err)
	assert.Len(t, imgs, 1)
	assert.Equal(t, img.ID, imgs[0].ID)

	vids, err := c.ListPending(ctx, mediaprocesstask.MediaTypeVideo, 50)
	require.NoError(t, err)
	require.Len(t, vids, 1)
	assert.Equal(t, vid.ID, vids[0].ID)

	// Idempotency guard is per entity regardless of media type.
	dup, err := c.Enqueue(ctx, &MediaProcessEnqueueArgs{
		EntityID: 20, FileID: 2, OwnerID: 1, MediaType: mediaprocesstask.MediaTypeVideo,
	})
	require.NoError(t, err)
	assert.Equal(t, vid.ID, dup.ID, "duplicate video enqueue must reuse the active row")

	// Completing the video drops it from its lane without touching the image lane.
	_, err = c.SetStatus(ctx, vid.ID, &MediaProcessStatusArgs{Status: mediaprocesstask.StatusDone, ResultSize: 42})
	require.NoError(t, err)
	vids, err = c.ListPending(ctx, mediaprocesstask.MediaTypeVideo, 50)
	require.NoError(t, err)
	assert.Len(t, vids, 0)
	imgs, err = c.ListPending(ctx, mediaprocesstask.MediaTypeImage, 50)
	require.NoError(t, err)
	assert.Len(t, imgs, 1, "the image lane is unaffected by video completion")
}

// TestMediaProcessHasHandledForFile covers APP-102: the backfill sweep skips
// files that already have a terminal (done/skipped) row, so a re-run does not
// re-compress already-processed files.
func TestMediaProcessHasHandledForFile(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	c := NewMediaProcessClient(client, "sqlite")

	handled, err := c.HasHandledForFile(ctx, 500)
	require.NoError(t, err)
	assert.False(t, handled, "no rows yet")

	// A pending row is not terminal → not handled.
	row, err := c.Enqueue(ctx, &MediaProcessEnqueueArgs{EntityID: 900, FileID: 500, OwnerID: 1, MediaType: mediaprocesstask.MediaTypeImage})
	require.NoError(t, err)
	handled, err = c.HasHandledForFile(ctx, 500)
	require.NoError(t, err)
	assert.False(t, handled)

	// Done counts as handled.
	_, err = c.SetStatus(ctx, row.ID, &MediaProcessStatusArgs{Status: mediaprocesstask.StatusDone, ResultSize: 10})
	require.NoError(t, err)
	handled, err = c.HasHandledForFile(ctx, 500)
	require.NoError(t, err)
	assert.True(t, handled)

	// Skipped also counts as handled.
	row2, err := c.Enqueue(ctx, &MediaProcessEnqueueArgs{EntityID: 901, FileID: 501, OwnerID: 1, MediaType: mediaprocesstask.MediaTypeImage})
	require.NoError(t, err)
	_, err = c.SetStatus(ctx, row2.ID, &MediaProcessStatusArgs{Status: mediaprocesstask.StatusSkipped})
	require.NoError(t, err)
	handled, err = c.HasHandledForFile(ctx, 501)
	require.NoError(t, err)
	assert.True(t, handled)

	// fileID 0 is never handled.
	handled, err = c.HasHandledForFile(ctx, 0)
	require.NoError(t, err)
	assert.False(t, handled)
}
