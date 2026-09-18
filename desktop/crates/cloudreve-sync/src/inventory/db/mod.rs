mod drive_props;
mod file_metadata;
mod tasks;
mod upload_sessions;

pub use tasks::RecentTasks;

use anyhow::{Context, Result, anyhow};
use diesel::Connection;
use diesel::r2d2::{ConnectionManager, Pool, PooledConnection};
use diesel::sqlite::SqliteConnection;
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use dirs::home_dir;
use std::fs;
use std::path::PathBuf;
use std::sync::Arc;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("./migrations/inventory");

/// SQLite-backed inventory database that relies on Diesel for schema management.
pub struct InventoryDb {
    pool: Arc<Pool<ConnectionManager<SqliteConnection>>>,
}

impl InventoryDb {
    /// Create or open the inventory database at the default location (~/.cloudreve/meta.db)
    pub fn new() -> Result<Self> {
        let db_path = Self::get_db_path()?;
        Self::with_path(db_path)
    }

    /// Create or open the inventory database at a specific path.
    /// The schema is automatically migrated to the latest version on startup.
    pub fn with_path(path: PathBuf) -> Result<Self> {
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).with_context(|| {
                format!(
                    "Failed to create inventory db parent dir {}",
                    parent.display()
                )
            })?;
        }

        let database_url = path
            .to_str()
            .map(|s| s.to_string())
            .ok_or_else(|| anyhow!("Invalid inventory database path"))?;

        run_migrations(&database_url)?;

        let manager = ConnectionManager::<SqliteConnection>::new(database_url);
        let pool = Pool::builder()
            .max_size(1)
            .build(manager)
            .context("Failed to build inventory database connection pool")?;

        Ok(Self {
            pool: Arc::new(pool),
        })
    }

    fn get_db_path() -> Result<PathBuf> {
        let home = home_dir().ok_or_else(|| anyhow!("Unable to determine home directory"))?;
        Ok(home.join(".cloudreve").join("meta.db"))
    }

    pub(crate) fn connection(
        &self,
    ) -> Result<PooledConnection<ConnectionManager<SqliteConnection>>> {
        self.pool
            .get()
            .context("Failed to get connection from inventory pool")
    }
}

fn run_migrations(database_url: &str) -> Result<()> {
    let mut conn = SqliteConnection::establish(database_url)
        .with_context(|| format!("Failed to open inventory database at {}", database_url))?;
    conn.run_pending_migrations(MIGRATIONS)
        .map_err(|err| anyhow!("Failed to run inventory database migrations: {err}"))?;
    Ok(())
}
