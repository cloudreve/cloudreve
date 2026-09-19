package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/stretchr/testify/require"
)

func TestAbuseReportCreateListSetStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	defer client.Close()
	c := NewAbuseReportClient(client)
	ctx := context.Background()

	r1, err := c.Create(ctx, &CreateAbuseReportParams{
		ReporterID:    3,
		ReporterEmail: "reporter@example.com",
		TargetType:    types.AbuseTargetShare,
		TargetID:      11,
		Reason:        types.AbuseReasonCopyright,
		Description:   "pirated content",
	})
	require.NoError(t, err)
	require.Equal(t, types.AbuseStatusOpen, r1.Status)

	_, err = c.Create(ctx, &CreateAbuseReportParams{
		TargetType: types.AbuseTargetUser,
		TargetID:   5,
		Reason:     types.AbuseReasonSpam,
	})
	require.NoError(t, err)

	all, err := c.List(ctx, &ListAbuseReportArgs{
		PaginationArgs: &PaginationArgs{Page: 0, PageSize: 10},
	})
	require.NoError(t, err)
	require.Equal(t, 2, all.Total)
	require.Len(t, all.Reports, 2)
	// Newest first.
	require.Equal(t, types.AbuseTargetUser, all.Reports[0].TargetType)

	openOnly, err := c.List(ctx, &ListAbuseReportArgs{
		PaginationArgs: &PaginationArgs{Page: 0, PageSize: 10},
		Status:         types.AbuseStatusOpen,
	})
	require.NoError(t, err)
	require.Equal(t, 2, openOnly.Total)

	updated, err := c.SetStatus(ctx, r1.ID, types.AbuseStatusResolved, "share expired")
	require.NoError(t, err)
	require.Equal(t, types.AbuseStatusResolved, updated.Status)
	require.Equal(t, "share expired", updated.AdminNote)

	openOnly, err = c.List(ctx, &ListAbuseReportArgs{
		PaginationArgs: &PaginationArgs{Page: 0, PageSize: 10},
		Status:         types.AbuseStatusOpen,
	})
	require.NoError(t, err)
	require.Equal(t, 1, openOnly.Total)
}
