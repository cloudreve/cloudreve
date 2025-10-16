package migrator

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/application/migrator/model"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
)

func (m *Migrator) migrateUser() error {
	m.l.Info("Migrating users...")
	batchSize := 1000
	// Start from the saved offset if available
	offset := m.state.UserOffset
	ctx := context.Background()
	if m.state.UserIDs == nil {
		m.state.UserIDs = make(map[int]bool)
	}

	// If we're resuming, load existing user IDs
	if len(m.state.UserIDs) > 0 {
		m.l.Info("Resuming user migration from offset %d, %d users already migrated", offset, len(m.state.UserIDs))
	}

	for {
		m.l.Info("Migrating users with offset %d", offset)
		var users []model.User
		if err := model.DB.Limit(batchSize).Offset(offset).Find(&users).Error; err != nil {
			return fmt.Errorf("failed to list v3 users: %w", err)
		}

		if len(users) == 0 {
			if m.dep.ConfigProvider().Database().Type == conf.PostgresDB {
				m.l.Info("Resetting user ID sequence for postgres...")
				m.v4client.User.ExecContext(ctx, "SELECT SETVAL('users_id_seq',  (SELECT MAX(id) FROM users))")
			}
			break
		}

		tx, err := m.v4client.Tx(context.Background())
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		for _, u := range users {
			userStatus := user.StatusActive
			switch u.Status {
			case model.Active:
				userStatus = user.StatusActive
			case model.NotActivicated:
				userStatus = user.StatusInactive
			case model.Baned:
				userStatus = user.StatusManualBanned
			case model.OveruseBaned:
				userStatus = user.StatusSysBanned
			}

			setting := &types.UserSetting{
				VersionRetention:    true,
				VersionRetentionMax: 10,
			}

			stm := tx.User.Create().
				SetRawID(int(u.ID)).
				SetCreatedAt(formatTime(u.CreatedAt)).
				SetUpdatedAt(formatTime(u.UpdatedAt)).
				SetEmail(u.Email).
				SetNick(u.Nick).
				SetStatus(userStatus).
				SetStorage(int64(u.Storage)).
				AddGroupIDs(int(u.GroupID)).
				SetSettings(setting).
				SetPassword(u.Password)

			if u.TwoFactor != "" {
				stm.SetTwoFactorSecret(u.TwoFactor)
			}

			if u.Avatar != "" {
				stm.SetAvatar(u.Avatar)
			}

			if _, err := stm.Save(ctx); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("failed to create user %d: %w", u.ID, err)
			}

			m.state.UserIDs[int(u.ID)] = true
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		// Update the offset in state and save after each batch
		offset += batchSize
		m.state.UserOffset = offset
		if err := m.saveState(); err != nil {
			m.l.Warning("Failed to save state after user batch: %s", err)
		} else {
			m.l.Info("Saved migration state after processing %d users", offset)
		}
	}

	return nil
}
