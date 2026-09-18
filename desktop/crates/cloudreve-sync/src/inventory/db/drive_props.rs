use super::InventoryDb;
use crate::inventory::{DriveProps, DrivePropsUpdate};
use anyhow::{Context, Result};
use chrono::Utc;
use diesel::prelude::*;

use crate::inventory::schema::drive_props::{self, dsl as drive_props_dsl};

impl InventoryDb {
    /// Get drive props by drive ID
    pub fn get_drive_props(&self, drive_id: &str) -> Result<Option<DriveProps>> {
        let mut conn = self.connection()?;
        let row = drive_props_dsl::drive_props
            .filter(drive_props_dsl::drive_id.eq(drive_id))
            .first::<DrivePropsRow>(&mut conn)
            .optional()
            .context("Failed to query drive props")?;

        row.map(DriveProps::try_from).transpose()
    }

    /// Check if drive props exist for a drive
    pub fn has_drive_props(&self, drive_id: &str) -> Result<bool> {
        let mut conn = self.connection()?;
        let exists: Option<i64> = drive_props_dsl::drive_props
            .filter(drive_props_dsl::drive_id.eq(drive_id))
            .select(drive_props_dsl::id)
            .first(&mut conn)
            .optional()
            .context("Failed to check drive props existence")?;

        Ok(exists.is_some())
    }

    /// Insert or update drive props
    pub fn upsert_drive_props(&self, drive_id: &str, update: DrivePropsUpdate) -> Result<()> {
        if update.is_empty() {
            return Ok(());
        }

        let mut conn = self.connection()?;
        let now = Utc::now().timestamp();

        // Check if record exists
        let exists = drive_props_dsl::drive_props
            .filter(drive_props_dsl::drive_id.eq(drive_id))
            .select(drive_props_dsl::id)
            .first::<i64>(&mut conn)
            .optional()
            .context("Failed to check drive props existence")?
            .is_some();

        if exists {
            // Update existing record
            let changeset = DrivePropsChangeset::from_update(update, now)?;
            diesel::update(
                drive_props_dsl::drive_props.filter(drive_props_dsl::drive_id.eq(drive_id)),
            )
            .set(changeset)
            .execute(&mut conn)
            .context("Failed to update drive props")?;
        } else {
            // Insert new record
            let row = NewDrivePropsRow::from_update(drive_id, update, now)?;
            diesel::insert_into(drive_props::table)
                .values(&row)
                .execute(&mut conn)
                .context("Failed to insert drive props")?;
        }

        Ok(())
    }

    /// Delete drive props by drive ID
    pub fn delete_drive_props(&self, drive_id: &str) -> Result<()> {
        let mut conn = self.connection()?;
        diesel::delete(drive_props_dsl::drive_props.filter(drive_props_dsl::drive_id.eq(drive_id)))
            .execute(&mut conn)
            .context("Failed to delete drive props")?;
        Ok(())
    }
}

// =========================================================================
// Row Types
// =========================================================================

#[derive(Queryable)]
struct DrivePropsRow {
    id: i64,
    drive_id: String,
    capacity: Option<String>,
    capacity_updated_at: Option<i64>,
    storage_policies: Option<String>,
    storage_policies_updated_at: Option<i64>,
    user_settings: Option<String>,
    user_settings_updated_at: Option<i64>,
    created_at: i64,
    updated_at: i64,
}

impl TryFrom<DrivePropsRow> for DriveProps {
    type Error = anyhow::Error;

    fn try_from(row: DrivePropsRow) -> Result<Self> {
        let capacity = row
            .capacity
            .map(|s| serde_json::from_str(&s))
            .transpose()
            .context("Failed to deserialize capacity")?;
        let storage_policies = row
            .storage_policies
            .map(|s| serde_json::from_str(&s))
            .transpose()
            .context("Failed to deserialize storage policies")?;
        let user_settings = row
            .user_settings
            .map(|s| serde_json::from_str(&s))
            .transpose()
            .context("Failed to deserialize user settings")?;

        Ok(DriveProps {
            id: row.id,
            drive_id: row.drive_id,
            capacity,
            capacity_updated_at: row.capacity_updated_at,
            storage_policies,
            storage_policies_updated_at: row.storage_policies_updated_at,
            user_settings,
            user_settings_updated_at: row.user_settings_updated_at,
            created_at: row.created_at,
            updated_at: row.updated_at,
        })
    }
}

#[derive(Insertable)]
#[diesel(table_name = drive_props)]
struct NewDrivePropsRow {
    drive_id: String,
    capacity: Option<String>,
    capacity_updated_at: Option<i64>,
    storage_policies: Option<String>,
    storage_policies_updated_at: Option<i64>,
    user_settings: Option<String>,
    user_settings_updated_at: Option<i64>,
    created_at: i64,
    updated_at: i64,
}

impl NewDrivePropsRow {
    fn from_update(drive_id: &str, update: DrivePropsUpdate, now: i64) -> Result<Self> {
        let has_capacity = update.capacity.is_some();
        let has_storage_policies = update.storage_policies.is_some();
        let has_user_settings = update.user_settings.is_some();

        let capacity = match update.capacity {
            Some(Some(c)) => {
                Some(serde_json::to_string(&c).context("Failed to serialize capacity")?)
            }
            _ => None,
        };
        let capacity_updated_at = if has_capacity { Some(now) } else { None };

        let storage_policies = match update.storage_policies {
            Some(Some(p)) => {
                Some(serde_json::to_string(&p).context("Failed to serialize storage policies")?)
            }
            _ => None,
        };
        let storage_policies_updated_at = if has_storage_policies {
            Some(now)
        } else {
            None
        };

        let user_settings = match update.user_settings {
            Some(Some(s)) => {
                Some(serde_json::to_string(&s).context("Failed to serialize user settings")?)
            }
            _ => None,
        };
        let user_settings_updated_at = if has_user_settings { Some(now) } else { None };

        Ok(Self {
            drive_id: drive_id.to_string(),
            capacity,
            capacity_updated_at,
            storage_policies,
            storage_policies_updated_at,
            user_settings,
            user_settings_updated_at,
            created_at: now,
            updated_at: now,
        })
    }
}

#[derive(AsChangeset)]
#[diesel(table_name = drive_props)]
struct DrivePropsChangeset {
    capacity: Option<Option<String>>,
    capacity_updated_at: Option<Option<i64>>,
    storage_policies: Option<Option<String>>,
    storage_policies_updated_at: Option<Option<i64>>,
    user_settings: Option<Option<String>>,
    user_settings_updated_at: Option<Option<i64>>,
    updated_at: i64,
}

impl DrivePropsChangeset {
    fn from_update(update: DrivePropsUpdate, now: i64) -> Result<Self> {
        let has_capacity = update.capacity.is_some();
        let has_storage_policies = update.storage_policies.is_some();
        let has_user_settings = update.user_settings.is_some();

        let capacity = match update.capacity {
            Some(Some(c)) => Some(Some(
                serde_json::to_string(&c).context("Failed to serialize capacity")?,
            )),
            Some(None) => Some(None),
            None => None,
        };
        let capacity_updated_at = if has_capacity { Some(Some(now)) } else { None };

        let storage_policies = match update.storage_policies {
            Some(Some(p)) => Some(Some(
                serde_json::to_string(&p).context("Failed to serialize storage policies")?,
            )),
            Some(None) => Some(None),
            None => None,
        };
        let storage_policies_updated_at = if has_storage_policies {
            Some(Some(now))
        } else {
            None
        };

        let user_settings = match update.user_settings {
            Some(Some(s)) => Some(Some(
                serde_json::to_string(&s).context("Failed to serialize user settings")?,
            )),
            Some(None) => Some(None),
            None => None,
        };
        let user_settings_updated_at = if has_user_settings {
            Some(Some(now))
        } else {
            None
        };

        Ok(Self {
            capacity,
            capacity_updated_at,
            storage_policies,
            storage_policies_updated_at,
            user_settings,
            user_settings_updated_at,
            updated_at: now,
        })
    }
}
