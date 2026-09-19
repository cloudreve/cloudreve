package dbfs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSourceIDMismatch(t *testing.T) {
	ctx := context.Background()

	// No expectations -> never mismatches.
	require.False(t, sourceIDMismatch(ctx, 0, 42))
	require.False(t, sourceIDMismatch(ctx, 5, 42))

	ctx = WithExpectedSourceIDs(ctx, []int{11, 0, 33})

	// Matching positions pass.
	require.False(t, sourceIDMismatch(ctx, 0, 11))
	// Zero entries disable the check for that position.
	require.False(t, sourceIDMismatch(ctx, 1, 999))
	// Positions beyond the expectation list pass.
	require.False(t, sourceIDMismatch(ctx, 3, 999))
	// Mismatches are caught per position.
	require.True(t, sourceIDMismatch(ctx, 0, 12))
	require.True(t, sourceIDMismatch(ctx, 2, 34))
}
