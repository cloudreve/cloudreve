package dbfs

import (
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
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
