package migrator

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/application/migrator/model"
	"github.com/cloudreve/Cloudreve/v4/ent/file"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
)

func (m *Migrator) migrateShare() error {
	m.l.Info("Migrating shares...")
	batchSize := 1000
	offset := m.state.ShareOffset
	ctx := context.Background()

	if offset > 0 {
		m.l.Info("Resuming share migration from offset %d", offset)
	}

	for {
		m.l.Info("Migrating shares with offset %d", offset)
		var shares []model.Share
		if err := model.DB.Limit(batchSize).Offset(offset).Find(&shares).Error; err != nil {
			return fmt.Errorf("failed to list v3 shares: %w", err)
		}

		if len(shares) == 0 {
			if m.dep.ConfigProvider().Database().Type == conf.PostgresDB {
				m.l.Info("Resetting share ID sequence for postgres...")
				m.v4client.Share.ExecContext(ctx, "SELECT SETVAL('shares_id_seq',  (SELECT MAX(id) FROM shares))")
			}
			break
		}

		tx, err := m.v4client.Tx(ctx)
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		for _, s := range shares {
			sourceId := int(s.SourceID)
			if !s.IsDir {
				sourceId += m.state.LastFolderID
			}

			// check if file exists
			_, err = tx.File.Query().Where(file.ID(sourceId)).First(ctx)
			if err != nil {
				m.l.Warning("File %d not found, skipping share %d", sourceId, s.ID)
				continue
			}

			// check if user exist
			if _, ok := m.state.UserIDs[int(s.UserID)]; !ok {
				m.l.Warning("User %d not found, skipping share %d", s.UserID, s.ID)
				continue
			}

			stm := tx.Share.Create().
				SetCreatedAt(formatTime(s.CreatedAt)).
				SetUpdatedAt(formatTime(s.UpdatedAt)).
				SetViews(s.Views).
				SetRawID(int(s.ID)).
				SetDownloads(s.Downloads).
				SetFileID(sourceId).
				SetUserID(int(s.UserID))

			if s.Password != "" {
				stm.SetPassword(s.Password)
			}

			if s.Expires != nil {
				stm.SetNillableExpires(s.Expires)
			}

			if s.RemainDownloads >= 0 {
				stm.SetRemainDownloads(s.RemainDownloads)
			}

			if _, err := stm.Save(ctx); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("failed to create share %d: %w", s.ID, err)
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		offset += batchSize
		m.state.ShareOffset = offset
		if err := m.saveState(); err != nil {
			m.l.Warning("Failed to save state after share batch: %s", err)
		} else {
			m.l.Info("Saved migration state after processing this batch")
		}
	}
	return nil
}
