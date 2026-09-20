package workflows

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/stretchr/testify/require"
)

func TestImportTaskSummarizeIncludesPolicyName(t *testing.T) {
	hasher, err := hashid.New("test-salt")
	require.NoError(t, err)

	owner := &ent.User{ID: 1}
	task, err := NewImportTask(context.Background(), owner, "/src/path", true, "cloudreve://my/dst", 7, "Local Storage")
	require.NoError(t, err)

	summary := task.Summarize(hasher)
	require.NotNil(t, summary)
	require.Equal(t, "Local Storage", summary.Props[SummaryKeyDstPolicyName])
	require.Equal(t, hashid.EncodePolicyID(hasher, 7), summary.Props[SummaryKeySrcDstPolicyID])
	require.Equal(t, "/src/path", summary.Props[SummaryKeySrcStr])
	require.Equal(t, "cloudreve://my/dst", summary.Props[SummaryKeyDst])
}

func TestImportTaskSummarizeLegacyState(t *testing.T) {
	hasher, err := hashid.New("test-salt")
	require.NoError(t, err)

	// Tasks created before PolicyName existed have an empty name; the
	// summary must still carry the encoded policy id for client-side lookup.
	owner := &ent.User{ID: 1}
	task, err := NewImportTask(context.Background(), owner, "/src/path", false, "cloudreve://my/dst", 7, "")
	require.NoError(t, err)

	summary := task.Summarize(hasher)
	require.NotNil(t, summary)
	require.Equal(t, "", summary.Props[SummaryKeyDstPolicyName])
	require.Equal(t, hashid.EncodePolicyID(hasher, 7), summary.Props[SummaryKeySrcDstPolicyID])
}

func TestImportTaskFromModelSummarize(t *testing.T) {
	hasher, err := hashid.New("test-salt")
	require.NoError(t, err)

	owner := &ent.User{ID: 1}
	created, err := NewImportTask(context.Background(), owner, "/src/path", true, "cloudreve://my/dst", 7, "S3 Backup")
	require.NoError(t, err)

	// Round-trip through the persisted model state.
	restored := NewImportTaskFromModel(created.(*ImportTask).Task)
	require.Equal(t, queue.ImportTaskType, restored.Type())

	summary := restored.Summarize(hasher)
	require.NotNil(t, summary)
	require.Equal(t, "S3 Backup", summary.Props[SummaryKeyDstPolicyName])
}
