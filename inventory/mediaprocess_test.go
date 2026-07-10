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
