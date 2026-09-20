//! End-to-end smoke test for the Linux on-demand filesystem: mounts
//! `CloudreveFs` on a real FUSE mountpoint backed by a stub Cloudreve server
//! and exercises listing, attribute reads, and hydrate-on-open through the
//! kernel. Skips gracefully when FUSE is unavailable (CI containers).
#![cfg(target_os = "linux")]

use std::{ffi::OsString, path::Path, sync::Arc};

use axum::{
    Json, Router,
    routing::{get, post},
};
use cloudreve_api::{Client, ClientConfig};
use cloudreve_sync::{
    drive::{commands::MountCommand, fuse_fs},
    inventory::{InventoryDb, MetadataEntry},
};
use tokio::sync::mpsc;
use uuid::Uuid;

const FILE_BYTES: &[u8] = b"hello fuse world";

/// Minimal Cloudreve stub: `/file/url` points at `/dl/hello`, which serves the
/// file bytes. Authentication is irrelevant — the client just omits it.
async fn stub_server() -> String {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let base = format!("http://{}", listener.local_addr().unwrap());
    let download = format!("{base}/dl/hello");

    let app = Router::new()
        .route(
            "/api/v3/file/url",
            post(move || {
                let download = download.clone();
                async move {
                    Json(serde_json::json!({
                        "code": 0,
                        "msg": "",
                        "data": {
                            "urls": [{ "url": download }],
                            "expires": "2999-01-01T00:00:00Z",
                        },
                    }))
                }
            }),
        )
        .route("/dl/hello", get(|| async { FILE_BYTES }));
    tokio::spawn(axum::serve(listener, app).into_future());
    base
}

fn dir_names(path: &Path) -> Vec<OsString> {
    let mut names: Vec<OsString> = std::fs::read_dir(path)
        .unwrap()
        .map(|e| e.unwrap().file_name())
        .collect();
    names.sort();
    names
}

#[tokio::test(flavor = "multi_thread", worker_threads = 4)]
async fn fuse_mount_lists_and_hydrates_virtual_files() {
    let base = stub_server().await;

    let tmp = tempfile::tempdir().unwrap();
    let store = tmp.path().join("store");
    let mountpoint = tmp.path().join("mnt");
    std::fs::create_dir_all(&store).unwrap();
    std::fs::create_dir_all(&mountpoint).unwrap();

    let drive_id = Uuid::new_v4().to_string();
    let drive_uuid = Uuid::parse_str(&drive_id).unwrap();
    let inventory = Arc::new(InventoryDb::with_path(tmp.path().join("meta.db")).unwrap());

    // Remote tree: docs/ with a single virtual file inside.
    let remote_dir = store.join("docs");
    let remote_file = remote_dir.join("hello.txt");
    inventory
        .insert(&MetadataEntry::new(
            drive_uuid,
            remote_dir.to_string_lossy().to_string(),
            true,
        ))
        .unwrap();
    inventory
        .insert(
            &MetadataEntry::new(drive_uuid, remote_file.to_string_lossy().to_string(), false)
                .with_etag("entity-1")
                .with_size(FILE_BYTES.len() as i64),
        )
        .unwrap();

    // The fs sends synthesized watcher events for virtual-entry mutations;
    // drain them so sends never fail.
    let (command_tx, mut command_rx) = mpsc::unbounded_channel::<MountCommand>();
    tokio::spawn(async move { while command_rx.recv().await.is_some() {} });

    let ctx = Arc::new(fuse_fs::FuseContext {
        data_root: store.clone(),
        remote_base: "cloudreve://my/".to_string(),
        drive_id,
        inventory: inventory.clone(),
        cr_client: Arc::new(Client::new(ClientConfig::new(&base))),
        command_tx,
        runtime: tokio::runtime::Handle::current(),
    });

    let session = match fuse_fs::spawn(fuse_fs::CloudreveFs::new(ctx), &mountpoint) {
        Ok(session) => session,
        Err(e) => {
            eprintln!("FUSE unavailable in this environment ({e}); skipping smoke test");
            return;
        }
    };

    // Virtual remote tree is listed without any store content.
    assert_eq!(dir_names(&mountpoint), vec![OsString::from("docs")]);
    assert_eq!(
        dir_names(&mountpoint.join("docs")),
        vec![OsString::from("hello.txt")]
    );

    // Attributes come from remote metadata before hydration.
    let md = std::fs::symlink_metadata(mountpoint.join("docs/hello.txt")).unwrap();
    assert!(md.is_file());
    assert_eq!(md.len(), FILE_BYTES.len() as u64);
    assert!(!remote_file.exists(), "file must not be hydrated yet");

    // First read hydrates through the stub server.
    let content = std::fs::read(mountpoint.join("docs/hello.txt")).unwrap();
    assert_eq!(content, FILE_BYTES);
    assert!(
        remote_file.exists(),
        "hydrated bytes must land in the store"
    );

    // The hydration snapshot keeps the engine from seeing the store write as
    // a user modification.
    let meta = inventory
        .query_by_path(&remote_file.to_string_lossy())
        .unwrap()
        .unwrap();
    assert_eq!(meta.local_size, Some(FILE_BYTES.len() as i64));
    assert!(meta.local_updated_at.is_some());

    // Second read is served from the store.
    assert_eq!(std::fs::read(mountpoint.join("docs/hello.txt")).unwrap(), FILE_BYTES);

    // A nonexistent path errors instead of hydrating.
    assert!(std::fs::read(mountpoint.join("docs/missing.txt")).is_err());

    drop(session);
}
