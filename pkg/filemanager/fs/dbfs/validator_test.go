package dbfs

import (
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestValidateFileNameNativeChars(t *testing.T) {
	native := &ent.StoragePolicy{Settings: &types.PolicySetting{AllowNativeName: true}}

	// Windows-illegal characters accepted only under allow_native_name.
	for _, name := range []string{"a:b.txt", "a<b>.txt", "a|b?.txt", `say "hi".txt`} {
		require.Error(t, validateFileName(name, nil))
		require.Error(t, validateFileName(name, &ent.StoragePolicy{Settings: &types.PolicySetting{}}))
		require.NoError(t, validateFileName(name, native))
	}

	// Separators and dot-names stay illegal regardless.
	for _, name := range []string{"a/b", `a\b`, ".", "..", "", string(make([]byte, 300))} {
		require.Error(t, validateFileName(name, native))
	}

	require.NoError(t, validateFileName("normal file.txt", nil))
}

func TestValidatePolicyCapacityRaw(t *testing.T) {
	f := &DBFS{l: logging.NewConsoleLogger(logging.LevelError)}
	capped := &ent.StoragePolicy{Name: "s3", Settings: &types.PolicySetting{MaxTotalSize: 1000}}
	open := &ent.StoragePolicy{Name: "local", Settings: &types.PolicySetting{}}

	// Zero cap disables the check.
	require.NoError(t, f.validatePolicyCapacityRaw(1<<62, open, 1<<62))
	// Within budget.
	require.NoError(t, f.validatePolicyCapacityRaw(400, capped, 500))
	// Exactly at the cap is the last allowed byte.
	require.NoError(t, f.validatePolicyCapacityRaw(500, capped, 500))
	// Over the cap.
	require.ErrorIs(t, f.validatePolicyCapacityRaw(501, capped, 500), fs.ErrInsufficientCapacity)
	require.ErrorIs(t, f.validatePolicyCapacityRaw(1, capped, 1000), fs.ErrInsufficientCapacity)
}
