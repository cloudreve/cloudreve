package inventory

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSQLiteWALAndBusyTimeout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sqlite_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	drv := sqlite3Driver{Driver: &sqlite.Driver{}}
	
	sqlDB := sql.OpenDB(driverConnector{driver: drv, name: dbPath})
	defer sqlDB.Close()

	// Verify journal_mode is WAL
	var journalMode string
	err = sqlDB.QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected journal_mode to be 'wal', got %q", journalMode)
	}

	// Verify busy_timeout is set (e.g. 5000ms)
	var busyTimeout int
	err = sqlDB.QueryRow("PRAGMA busy_timeout;").Scan(&busyTimeout)
	if err != nil {
		t.Fatalf("failed to query busy_timeout: %v", err)
	}
	if busyTimeout < 5000 {
		t.Errorf("expected busy_timeout to be at least 5000, got %d", busyTimeout)
	}

	// Verify foreign_keys is ON
	var foreignKeys int
	err = sqlDB.QueryRow("PRAGMA foreign_keys;").Scan(&foreignKeys)
	if err != nil {
		t.Fatalf("failed to query foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("expected foreign_keys to be 1, got %d", foreignKeys)
	}

	// Verify concurrent access safety (multiple readers/writers)
	_, err = sqlDB.Exec("CREATE TABLE IF NOT EXISTS test_table (id INTEGER PRIMARY KEY, val TEXT);")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errChan := make(chan error, 10)
	for i := 0; i < 5; i++ {
		go func(n int) {
			_, err := sqlDB.ExecContext(ctx, "INSERT INTO test_table (val) VALUES (?);", n)
			errChan <- err
		}(i)
	}

	for i := 0; i < 5; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("concurrent insert failed: %v", err)
		}
	}
}

type driverConnector struct {
	driver sqlite3Driver
	name   string
}

func (c driverConnector) Connect(context.Context) (driver.Conn, error) {
	return c.driver.Open(c.name)
}

func (c driverConnector) Driver() driver.Driver {
	return c.driver
}
