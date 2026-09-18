# Inventory Module

A SQLite-backed inventory system for persisting file metadata in the Cloudreve sync service.  
Implementation uses the Diesel ORM with embedded migrations so the schema is automatically migrated to the latest version whenever the process starts.

## Overview

The `inventory` module provides a robust way to store and query file metadata for synchronized files. It uses SQLite as the storage backend and is designed for efficient lookups and updates.

## Database Schema

The inventory database contains a single table `file_metadata` with the following columns:

- `id` (INTEGER PRIMARY KEY): Auto-incrementing unique identifier
- `drive_id` (TEXT): UUID of the drive this file belongs to
- `local_path` (TEXT UNIQUE): Local filesystem path (unique constraint)
- `remote_uri` (TEXT): Remote URI on the Cloudreve server
- `created_at` (INTEGER): Unix timestamp when the record was created
- `updated_at` (INTEGER): Unix timestamp when the record was last updated
- `etag` (STRING): Entity tag for change detection
- `metadata` (TEXT): JSON-serialized key-value pairs (all strings)
- `props` (TEXT): Reserved JSON field for future use

### Indexes

The following indexes are created for optimal query performance:

- `idx_drive_id`: Index on `drive_id`
- `idx_local_path`: Index on `local_path`
- `idx_updated_at`: Index on `updated_at`

## Database Location

By default, the database file is stored at:

- Windows: `C:\Users\{username}\.cloudreve\meta.db`
- Linux/macOS: `~/.cloudreve/meta.db`

## Usage Examples

### Initialize the Database

```rust
use cloudreve_sync::inventory::InventoryDb;

// Create or open the database at the default location
let db = InventoryDb::new()?;

// Or specify a custom path
let db = InventoryDb::with_path("path/to/custom/meta.db".into())?;
```

### Insert a New Entry

```rust
use cloudreve_sync::inventory::{InventoryDb, MetadataEntry};
use uuid::Uuid;
use std::collections::HashMap;

let db = InventoryDb::new()?;
let drive_id = Uuid::new_v4();

// Create a basic entry
let entry = MetadataEntry::new(
    drive_id,
    "C:\\Users\\John\\Documents\\report.pdf",
    "/documents/report.pdf"
)
.with_etag("abc123xyz");

let id = db.insert(&entry)?;
println!("Inserted entry with ID: {}", id);

// Create an entry with metadata
let mut metadata = HashMap::new();
metadata.insert("content_type".to_string(), "application/pdf".to_string());
metadata.insert("size".to_string(), "1048576".to_string());

let entry = MetadataEntry::new(
    drive_id,
    "C:\\Users\\John\\Documents\\data.xlsx",
    "/documents/data.xlsx"
)
.with_etag("def456")
.with_metadata(metadata);

db.insert(&entry)?;
```

### Query by Local Path

```rust
let result = db.query_by_path("C:\\Users\\John\\Documents\\report.pdf")?;

if let Some(metadata) = result {
    println!("File found!");
    println!("  Drive ID: {}", metadata.drive_id);
    println!("  Remote URI: {}", metadata.remote_uri);
    println!("  ETag: {}", metadata.etag);
    println!("  Updated at: {}", metadata.updated_at);

    // Access custom metadata
    if let Some(content_type) = metadata.metadata.get("content_type") {
        println!("  Content Type: {}", content_type);
    }
} else {
    println!("File not found in inventory");
}
```

### Update an Existing Entry

```rust
// Update using the same local path
let entry = MetadataEntry::new(
    drive_id,
    "C:\\Users\\John\\Documents\\report.pdf",
    "/documents/report_v2.pdf"
)
.with_etag("new_etag_456");

let updated = db.update(&entry)?;
if updated {
    println!("Entry updated successfully");
} else {
    println!("Entry not found");
}
```

### Upsert (Insert or Update)

```rust
// Upsert will insert if the path doesn't exist, or update if it does
let entry = MetadataEntry::new(
    drive_id,
    "C:\\Users\\John\\Documents\\report.pdf",
    "/documents/report_v3.pdf"
)
.with_etag("latest_etag");

db.upsert(&entry)?;
```

### Query All Files for a Drive

```rust
let files = db.query_by_drive(&drive_id)?;

println!("Found {} files for drive {}", files.len(), drive_id);
for file in files {
    println!("  {} -> {}", file.local_path, file.remote_uri);
}
```

### Delete Operations

```rust
// Delete by local path
let deleted = db.delete_by_path("C:\\Users\\John\\Documents\\report.pdf")?;
if deleted {
    println!("File deleted from inventory");
}

// Delete all files for a drive
let count = db.delete_by_drive(&drive_id)?;
println!("Deleted {} files", count);

// Clear entire database
db.clear()?;
```

### Get Database Statistics

```rust
let count = db.count()?;
println!("Total entries in inventory: {}", count);
```

## Advanced Usage with Props

The `props` field is reserved for future JSON data. You can use it to store arbitrary structured data:

```rust
use serde_json::json;

let props = json!({
    "sync_status": "synced",
    "conflict_version": null,
    "last_sync": 1699012345,
    "custom_flags": {
        "pinned": true,
        "favorite": false
    }
});

let entry = MetadataEntry::new(
    drive_id,
    "C:\\Users\\John\\Documents\\important.docx",
    "/documents/important.docx"
)
.with_etag("xyz789")
.with_props(props);

db.insert(&entry)?;

// Later, retrieve and use the props
let result = db.query_by_path("C:\\Users\\John\\Documents\\important.docx")?;
if let Some(metadata) = result {
    if let Some(props) = metadata.props {
        if let Some(status) = props.get("sync_status") {
            println!("Sync status: {}", status);
        }
    }
}
```

## Thread Safety

The `InventoryDb` holds an internal Diesel connection pool (`r2d2`) for thread-safe access to SQLite. It's safe to clone and share across threads:

```rust
use std::sync::Arc;

let db = Arc::new(InventoryDb::new()?);

// Clone and use in multiple threads
let db_clone = db.clone();
tokio::spawn(async move {
    let result = db_clone.query_by_path("some/path")?;
    // ... do something with result
    Ok::<_, Box<dyn std::error::Error + Send + Sync>>(())
});
```

## Error Handling

All database operations return `Result<T>` where the error type is `Box<dyn std::error::Error + Send + Sync>`. This allows for easy propagation with the `?` operator:

```rust
use cloudreve_sync::inventory::{InventoryDb, Result};

fn my_function() -> Result<()> {
    let db = InventoryDb::new()?;
    let entry = /* ... */;
    db.insert(&entry)?;
    Ok(())
}
```

## Testing

The module includes comprehensive unit tests. Run them with:

```bash
cargo test --package cloudreve-sync --lib inventory::db::tests
```

## Schema Migrations

Migrations live under `migrations/inventory` and are embedded into the binary via `diesel_migrations`. To introduce a schema change:

1. Create a new timestamped folder inside `migrations/inventory` (e.g. `0002_add_new_column`).
2. Add `up.sql`/`down.sql` files with the required statements. Prefer idempotent SQL (`IF NOT EXISTS`) so existing installs upgrade cleanly.
3. At runtime the new migration is applied automatically before any inventory queries run.
