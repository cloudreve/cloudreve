package inventory

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const heatWaveDDL = "CREATE TABLE `groups` (\n" +
	"  `id` bigint NOT NULL AUTO_INCREMENT,\n" +
	"  `created_at` datetime(3) NULL,\n" +
	"  `name` varchar(255) NOT NULL,\n" +
	"  `permissions` blob NOT NULL COMMENT 'json blob' NOT SECONDARY,\n" +
	"  `note` varchar(255) NULL DEFAULT 'not secondary',\n" +
	"  PRIMARY KEY (`id`),\n" +
	"  UNIQUE KEY `groups_name` (`name`),\n" +
	"  KEY `groups_created_at` (`created_at`)\n" +
	") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"

func TestExtractColumnDef(t *testing.T) {
	t.Run("strips NOT SECONDARY keeping the rest", func(t *testing.T) {
		def, err := extractColumnDef(heatWaveDDL, "permissions")
		require.NoError(t, err)
		require.Equal(t, "`permissions` blob NOT NULL COMMENT 'json blob'", def)
	})

	t.Run("leaves unrelated columns intact", func(t *testing.T) {
		def, err := extractColumnDef(heatWaveDDL, "note")
		require.NoError(t, err)
		// The string 'not secondary' inside a DEFAULT literal must survive.
		require.Equal(t, "`note` varchar(255) NULL DEFAULT 'not secondary'", def)
	})

	t.Run("index lines are not matched", func(t *testing.T) {
		_, err := extractColumnDef(heatWaveDDL, "groups_name")
		require.Error(t, err)
	})

	t.Run("missing column errors", func(t *testing.T) {
		_, err := extractColumnDef(heatWaveDDL, "nope")
		require.Error(t, err)
	})
}
